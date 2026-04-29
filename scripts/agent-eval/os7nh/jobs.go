package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func executeRun(ctx context.Context, config runConfig, stdout io.Writer, runner jobRunner) error {
	start := time.Now()
	jobs, err := buildJobs(config)
	if err != nil {
		return err
	}
	cache := cacheConfig{Mode: config.CacheMode, RunRoot: config.RunRoot}
	results := runJobs(ctx, config, jobs, cache, runner)
	elapsed := roundSeconds(time.Since(start).Seconds())
	totalAgent := totalAgentWallSeconds(results)
	effectiveSpeedup := 0.0
	parallelEfficiency := 0.0
	if elapsed > 0 {
		effectiveSpeedup = roundSeconds(totalAgent / elapsed)
	}
	if config.Parallel > 0 && effectiveSpeedup > 0 {
		parallelEfficiency = roundSeconds(effectiveSpeedup / float64(config.Parallel))
	}
	rep := report{
		Metadata: reportMetadata{
			GeneratedAt:              time.Now().UTC(),
			Model:                    modelName,
			ReasoningEffort:          reasoningEffort,
			Harness:                  "codex exec --json --full-auto from throwaway run directories; single-turn scenarios use --ephemeral and the installed openstudy runner",
			ConfiguredParallelism:    config.Parallel,
			CacheMode:                cache.Mode,
			HarnessElapsedSeconds:    elapsed,
			EffectiveParallelSpeedup: effectiveSpeedup,
			ParallelEfficiency:       parallelEfficiency,
			PhaseTotals:              aggregatePhaseTimings(results),
			RunRootArtifactReference: "<run-root>",
			RawLogPlaceholder:        "<run-root>/<variant>/<scenario>/events.jsonl",
			Variants:                 selectedVariants(config),
			Scenarios:                selectedScenarioIDs(config),
			ReleaseBlocking:          true,
			RawLogsCommitted:         false,
			RawLogsNote:              "Raw Codex event logs and SQLite databases remain under <run-root> and are not committed.",
		},
		Results:        results,
		ProductionGate: buildProductionGateSummary(results),
	}
	if err := os.MkdirAll(config.ReportDir, 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}
	jsonPath := filepath.Join(config.ReportDir, config.ReportName+".json")
	markdownPath := filepath.Join(config.ReportDir, config.ReportName+".md")
	if err := writeJSON(jsonPath, rep); err != nil {
		return fmt.Errorf("write JSON report: %w", err)
	}
	if err := writeMarkdownReport(markdownPath, rep); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "wrote %s and %s\n", filepath.ToSlash(jsonPath), filepath.ToSlash(markdownPath)); err != nil {
		return err
	}
	if !rep.ProductionGate.PassesGate {
		return errors.New("production gate failed")
	}
	return nil
}

func buildJobs(config runConfig) ([]evalJob, error) {
	variants := selectedVariants(config)
	scenarios := selectedScenarios(config)
	if len(scenarios) == 0 {
		return nil, errors.New("no scenarios selected")
	}
	jobs := make([]evalJob, 0, len(variants)*len(scenarios))
	for _, variant := range variants {
		for _, sc := range scenarios {
			jobs = append(jobs, evalJob{Index: len(jobs), Variant: variant, Scenario: sc})
		}
	}
	return jobs, nil
}

func runJobs(ctx context.Context, config runConfig, jobs []evalJob, cache cacheConfig, runner jobRunner) []jobResult {
	results := make([]jobResult, len(jobs))
	jobCh := make(chan evalJob)
	var wg sync.WaitGroup
	workers := min(config.Parallel, max(1, len(jobs)))
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				results[job.Index] = runner(ctx, config, job, cache)
			}
		}()
	}
	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)
	wg.Wait()
	return results
}

func codexJobRunner(ctx context.Context, config runConfig, job evalJob, cache cacheConfig) jobResult {
	start := time.Now()
	result := jobResult{
		Variant:       job.Variant,
		Scenario:      job.Scenario.ID,
		ScenarioTitle: job.Scenario.Title,
		Status:        "failed",
		StartedAt:     start.UTC(),
		PromptSummary: promptSummary(job.Scenario),
		Metrics:       emptyMetrics(),
	}
	timings := phaseTimings{}
	jobDir := filepath.Join(config.RunRoot, job.Variant, job.Scenario.ID)
	repoDir := filepath.Join(jobDir, "repo")
	paths := scenarioPaths(repoDir)
	if err := timedPhase(&timings.PrepareRunDir, func() error { return prepareRunDir(jobDir, paths, cache) }); err != nil {
		result.Error = err.Error()
		return result
	}
	if err := timedPhase(&timings.CopyRepo, func() error { return copyRepo(config.RepoRoot, repoDir) }); err != nil {
		result.Error = fmt.Sprintf("copy repo: %v", err)
		return result
	}
	if err := timedPhase(&timings.InstallVariant, func() error {
		if err := installVariant(config.RepoRoot, repoDir, job.Variant); err != nil {
			return err
		}
		if err := buildOpenStudyRunner(repoDir, jobDir, paths, cache); err != nil {
			return err
		}
		return preflightEvalContext(config.RepoRoot, repoDir, jobDir, paths, cache, config.CodexBin)
	}); err != nil {
		result.Error = fmt.Sprintf("configure variant: %v", err)
		return result
	}
	if cache.Mode == cacheModeIsolated {
		if err := timedPhase(&timings.WarmCache, func() error { return warmGoModules(repoDir, jobDir, paths, cache) }); err != nil {
			result.Error = fmt.Sprintf("warm go modules: %v", err)
			return result
		}
	}
	if err := timedPhase(&timings.SeedData, func() error { return seedScenario(ctx, paths.DatabasePath, job.Scenario.ID) }); err != nil {
		result.Error = fmt.Sprintf("seed scenario: %v", err)
		return result
	}

	eventsPath := filepath.Join(jobDir, "events.jsonl")
	stderrPath := filepath.Join(jobDir, "stderr.log")
	exitCode, runErr, wallSeconds := runCodex(ctx, config, repoDir, jobDir, paths, cache, job.Scenario.Prompt, eventsPath, stderrPath)
	timings.AgentRun = wallSeconds
	parseStart := time.Now()
	parsed, parseErr := parseMetrics(eventsPath)
	timings.ParseMetrics = roundSeconds(time.Since(parseStart).Seconds())
	verifyStart := time.Now()
	verification, verifyErr := verifyScenario(ctx, paths.DatabasePath, job.Scenario.ID, parsed.finalMessage, parsed.metrics)
	timings.Verify = roundSeconds(time.Since(verifyStart).Seconds())
	if parseErr != nil {
		parsed.metrics.CommandMetricLimitations = fmt.Sprintf("failed to parse event log: %v", parseErr)
	}
	if verifyErr != nil {
		verification = verificationResult{Passed: false, Details: fmt.Sprintf("verification error: %v", verifyErr)}
	}

	completed := time.Now().UTC()
	timings.Total = roundSeconds(time.Since(start).Seconds())
	result.CompletedAt = &completed
	result.ExitCode = exitCode
	result.WallSeconds = wallSeconds
	result.PhaseTimings = timings.rounded()
	result.Metrics = parsed.metrics
	result.Verification = verification
	result.RawLogArtifactReference = fmt.Sprintf("<run-root>/%s/%s/events.jsonl", job.Variant, job.Scenario.ID)
	result.Passed = runErr == nil && parseErr == nil && verifyErr == nil && verification.Passed
	if result.Passed {
		result.Status = "completed"
	} else {
		if runErr != nil {
			result.Error = runErr.Error()
		} else if parseErr != nil {
			result.Error = parseErr.Error()
		} else if verifyErr != nil {
			result.Error = verifyErr.Error()
		}
	}
	_ = writeJSON(filepath.Join(jobDir, "run-summary.json"), result)
	return result
}

func scenarioPaths(repoDir string) evalPaths {
	return evalPaths{
		DatabasePath: filepath.Join(repoDir, ".openstudy-eval", "openstudy.sqlite"),
	}
}

func evalPathsFor(runDir string, paths evalPaths, cache cacheConfig) evalPaths {
	out := paths
	out.CodexHome = filepath.Join(runDir, "codex-home")
	out.ZDotDir = filepath.Join(runDir, "zdotdir")
	out.Temp = filepath.Join(runDir, "tmp")
	if cache.Mode == cacheModeShared {
		out.GoCache = filepath.Join(cache.RunRoot, "shared-cache", "gocache")
		out.GoModCache = filepath.Join(cache.RunRoot, "shared-cache", "gomodcache")
	} else {
		out.GoCache = filepath.Join(runDir, "gocache")
		out.GoModCache = filepath.Join(runDir, "gomodcache")
	}
	return out
}

func prepareRunDir(runDir string, paths evalPaths, cache cacheConfig) error {
	effective := evalPathsFor(runDir, paths, cache)
	for _, dir := range []string{runDir, effective.ZDotDir, effective.Temp, effective.GoCache, effective.GoModCache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return setupEvalCodexHome(effective.CodexHome)
}

func setupEvalCodexHome(dst string) error {
	srcRoot, err := sourceCodexHome()
	if err != nil {
		return err
	}
	authBytes, err := os.ReadFile(filepath.Join(srcRoot, "auth.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing Codex auth at %s; run codex login before running evals", filepath.Join(srcRoot, "auth.json"))
		}
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, "auth.json"), authBytes, 0o600)
}

func sourceCodexHome() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("CODEX_HOME")); configured != "" {
		return configured, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

func runCodex(ctx context.Context, config runConfig, repoDir string, runDir string, paths evalPaths, cache cacheConfig, prompt string, eventsPath string, stderrPath string) (int, error, float64) {
	stdoutFile, err := os.Create(eventsPath)
	if err != nil {
		return 1, err, 0
	}
	defer func() { _ = stdoutFile.Close() }()
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		return 1, err, 0
	}
	defer func() { _ = stderrFile.Close() }()
	args := codexArgs(config.CodexBin, repoDir, runDir, cache, prompt)
	cmdCtx, cancel := context.WithTimeout(ctx, 7*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Dir = repoDir
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	cmd.Stdin = strings.NewReader("")
	cmd.Env = evalEnv(runDir, paths, cache)
	start := time.Now()
	err = cmd.Run()
	wall := roundSeconds(time.Since(start).Seconds())
	exitCode := commandExitCode(err)
	if cmdCtx.Err() == context.DeadlineExceeded {
		return -1, cmdCtx.Err(), wall
	}
	return exitCode, err, wall
}

func codexArgs(codexBin string, repoDir string, runDir string, cache cacheConfig, prompt string) []string {
	args := []string{
		codexBin, "exec", "--json", "--ephemeral", "--full-auto", "--skip-git-repo-check", "--ignore-user-config", "-C", repoDir,
		"--add-dir", runDir,
		"-m", modelName,
		"-c", fmt.Sprintf("model_reasoning_effort=%q", reasoningEffort),
		"-c", "shell_environment_policy.inherit=all",
	}
	if cache.Mode == cacheModeShared {
		args = append(args, "--add-dir", filepath.Join(cache.RunRoot, "shared-cache"))
	}
	return append(args, prompt)
}

func evalEnv(runDir string, paths evalPaths, cache cacheConfig) []string {
	effective := evalPathsFor(runDir, paths, cache)
	env := filteredEnv(os.Environ(),
		"CODEX_HOME",
		"OPENSTUDY_DATABASE_PATH",
		"GOCACHE",
		"GOMODCACHE",
		"TMPDIR",
		"PATH",
		"ZDOTDIR",
	)
	pathValue := filepath.Join(runDir, "bin")
	if existing := os.Getenv("PATH"); existing != "" {
		pathValue += string(os.PathListSeparator) + existing
	}
	return append(env,
		"CODEX_HOME="+effective.CodexHome,
		"ZDOTDIR="+effective.ZDotDir,
		"OPENSTUDY_DATABASE_PATH="+effective.DatabasePath,
		"GOCACHE="+effective.GoCache,
		"GOMODCACHE="+effective.GoModCache,
		"TMPDIR="+effective.Temp,
		"PATH="+pathValue,
	)
}

func filteredEnv(env []string, keys ...string) []string {
	blocked := map[string]struct{}{}
	for _, key := range keys {
		blocked[key] = struct{}{}
	}
	out := []string{}
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, skip := blocked[key]; skip {
				continue
			}
		}
		out = append(out, entry)
	}
	return out
}

func warmGoModules(repoDir string, runDir string, paths evalPaths, cache cacheConfig) error {
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = repoDir
	cmd.Env = evalEnv(runDir, paths, cache)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func buildOpenStudyRunner(repoDir string, runDir string, paths evalPaths, cache cacheConfig) error {
	binDir := filepath.Join(runDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(binDir, "openstudy"), "./cmd/openstudy")
	cmd.Dir = repoDir
	cmd.Env = evalEnv(runDir, paths, cache)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func copyRepo(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if shouldSkipCopy(rel, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	})
}

func shouldSkipCopy(rel string, entry fs.DirEntry) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	switch parts[0] {
	case ".git", ".beads", ".dolt", ".agents":
		return true
	case "AGENTS.md":
		return true
	}
	slash := filepath.ToSlash(rel)
	if strings.HasPrefix(slash, "docs/evals/results/") {
		return true
	}
	if slash == "scripts/agent-eval/os7nh" || strings.HasPrefix(slash, "scripts/agent-eval/os7nh/") {
		return true
	}
	return !entry.IsDir() && (strings.HasSuffix(slash, ".sqlite") || strings.HasSuffix(slash, ".db") || strings.HasSuffix(slash, ".sqlite3") || strings.HasSuffix(slash, ".jsonl"))
}

func installVariant(repoRoot string, repoDir string, variant string) error {
	if variant != productionVariant {
		return fmt.Errorf("unsupported variant %q", variant)
	}
	dest := filepath.Join(repoDir, ".agents", "skills", "openstudy")
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return copyDir(filepath.Join(repoRoot, "skills", "openstudy"), dest)
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	})
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
