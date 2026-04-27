package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yazanabuashour/openstudy/internal/localruntime"
	"github.com/yazanabuashour/openstudy/internal/study"
)

func TestParseRunConfigDefaults(t *testing.T) {
	config, err := parseRunConfig(nil, &strings.Builder{})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if config.Parallel != defaultParallel {
		t.Fatalf("parallel = %d, want %d", config.Parallel, defaultParallel)
	}
	if config.CacheMode != cacheModeShared {
		t.Fatalf("cache mode = %q, want %q", config.CacheMode, cacheModeShared)
	}
	if config.ReportName != "os7nh-latest" {
		t.Fatalf("report name = %q", config.ReportName)
	}
}

func TestParseRunConfigRejectsInvalidInputs(t *testing.T) {
	for _, args := range [][]string{
		{"--parallel", "0"},
		{"--cache-mode", "bad"},
		{"unexpected"},
		{"--report-name", ""},
	} {
		if _, err := parseRunConfig(args, &strings.Builder{}); err == nil {
			t.Fatalf("parse config %v succeeded, want error", args)
		}
	}
}

func TestModelPinIsReleaseGateModel(t *testing.T) {
	if modelName != "gpt-5.4-mini" {
		t.Fatalf("modelName = %q, want gpt-5.4-mini", modelName)
	}
}

func TestBuildJobsFiltersScenarios(t *testing.T) {
	jobs, err := buildJobs(runConfig{Variant: productionVariant, Scenario: scenarioBypassRejection + "," + scenarioRoughCardCreate})
	if err != nil {
		t.Fatalf("build jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("jobs = %d, want 2", len(jobs))
	}
	if jobs[0].Scenario.ID != scenarioRoughCardCreate || jobs[1].Scenario.ID != scenarioBypassRejection {
		t.Fatalf("job order = %q, %q", jobs[0].Scenario.ID, jobs[1].Scenario.ID)
	}
}

func TestShouldSkipCopyExcludesPrivateAndGeneratedState(t *testing.T) {
	tests := map[string]bool{
		"AGENTS.md":                         true,
		".git/config":                       true,
		".beads/state":                      true,
		".dolt/config.json":                 true,
		".agents/skills/openstudy/SKILL.md": true,
		"docs/evals/results/os7nh.json":     true,
		"scripts/agent-eval/os7nh/main.go":  true,
		"tmp/openstudy.sqlite":              true,
		"tmp/events.jsonl":                  true,
		"cmd/openstudy/main.go":             false,
	}
	for rel, want := range tests {
		entry := fakeDirEntry{dir: strings.HasSuffix(rel, "/")}
		if got := shouldSkipCopy(rel, entry); got != want {
			t.Fatalf("shouldSkipCopy(%q) = %t, want %t", rel, got, want)
		}
	}
}

func TestInstallVariantCopiesProductionSkill(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "skills", "openstudy")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("skill body\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	repo := filepath.Join(t.TempDir(), "repo")
	if err := installVariant(root, repo, productionVariant); err != nil {
		t.Fatalf("install variant: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(repo, ".agents", "skills", "openstudy", "SKILL.md"))
	if err != nil {
		t.Fatalf("read installed skill: %v", err)
	}
	if string(got) != "skill body\n" {
		t.Fatalf("installed skill = %q", got)
	}
}

func TestParseMetricsClassifiesBypassAndRunnerUse(t *testing.T) {
	events := strings.Join([]string{
		`{"type":"message","item":{"role":"assistant","content":[{"type":"output_text","text":"OpenStudy routine work must use the installed runner."}]},"usage":{"input_tokens":100,"cached_input_tokens":25,"output_tokens":10}}`,
		`{"type":"tool_call","item":{"cmd":"sqlite3 .openstudy-eval/openstudy.sqlite 'select * from cards'; go run ./cmd/openstudy cards; rg --files; echo '{\"action\":\"create_card\"}' | openstudy cards"}}`,
		`{"type":"exec_command_begin","command":"find . -type f; python3 helper.py","parsed_cmd":["find",".","-type","f"]}`,
		`{"type":"exec_command_begin","action":{"cmd":"go env GOMODCACHE"}}`,
	}, "\n") + "\n"
	path := filepath.Join(t.TempDir(), "events.jsonl")
	if err := os.WriteFile(path, []byte(events), 0o644); err != nil {
		t.Fatalf("write events: %v", err)
	}
	parsed, err := parseMetrics(path)
	if err != nil {
		t.Fatalf("parse metrics: %v", err)
	}
	if !parsed.metrics.DirectSQLiteAccess || !parsed.metrics.SourceBuiltRunnerUsage || !parsed.metrics.BroadRepoSearch {
		t.Fatalf("metrics did not detect bypasses: %+v", parsed.metrics)
	}
	if !parsed.metrics.AdHocScriptUsage || !parsed.metrics.ModuleCacheInspection {
		t.Fatalf("metrics did not detect command events outside item: %+v", parsed.metrics)
	}
	if !parsed.metrics.RunnerCardsUsed {
		t.Fatalf("metrics did not detect runner cards use: %+v", parsed.metrics)
	}
	if parsed.metrics.InputTokens == nil || *parsed.metrics.InputTokens != 100 {
		t.Fatalf("usage = %+v", parsed.metrics)
	}
	if strings.Contains(parsed.metrics.BypassEvidence[0], os.TempDir()) {
		t.Fatalf("evidence was not sanitized: %+v", parsed.metrics.BypassEvidence)
	}
}

func TestVerifyFinalAnswerOnlyRejection(t *testing.T) {
	result, err := verifyFinalAnswerOnlyRejection("card.back is required; please provide it.", emptyMetrics(), []string{"card.back", "required"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !result.Passed {
		t.Fatalf("result = %+v", result)
	}
	m := emptyMetrics()
	m.ToolCalls = 1
	result, err = verifyFinalAnswerOnlyRejection("card.back is required.", m, []string{"card.back"})
	if err != nil {
		t.Fatalf("verify with tools: %v", err)
	}
	if result.Passed {
		t.Fatalf("tool-using rejection passed: %+v", result)
	}
}

func TestSeedAndVerifyDatabaseScenarios(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "data", "openstudy.sqlite")
	if err := seedScenario(ctx, dbPath, scenarioDueWindowReview); err != nil {
		t.Fatalf("seed: %v", err)
	}
	createReviewAttemptForTest(t, dbPath, "good")
	result, err := verifyReviewAttempt(ctx, dbPath, "good", emptyMetrics())
	if err != nil {
		t.Fatalf("verify review: %v", err)
	}
	if !result.Passed {
		t.Fatalf("review verification = %+v", result)
	}
}

func TestExecuteRunWritesSanitizedReports(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "reports")
	config := runConfig{
		Parallel:   2,
		Variant:    productionVariant,
		Scenario:   strings.Join(releaseBlockingScenarioIDs(), ","),
		RunRoot:    filepath.Join(t.TempDir(), "run"),
		ReportDir:  reportDir,
		ReportName: "os7nh-test",
		RepoRoot:   ".",
		CodexBin:   "codex",
		CacheMode:  cacheModeIsolated,
	}
	err := executeRun(context.Background(), config, &strings.Builder{}, func(_ context.Context, _ runConfig, job evalJob, _ cacheConfig) jobResult {
		now := time.Now().UTC()
		return jobResult{
			Variant:                 job.Variant,
			Scenario:                job.Scenario.ID,
			ScenarioTitle:           job.Scenario.Title,
			Passed:                  true,
			Status:                  "completed",
			Metrics:                 metrics{AssistantCalls: 1, EventTypeCounts: map[string]int{}},
			Verification:            verificationResult{Passed: true, DatabasePass: true, AssistantPass: true},
			PhaseTimings:            phaseTimings{AgentRun: 0.25, Total: 0.30},
			WallSeconds:             0.25,
			RawLogArtifactReference: "<run-root>/" + job.Variant + "/" + job.Scenario.ID + "/events.jsonl",
			StartedAt:               now,
			CompletedAt:             &now,
		}
	})
	if err != nil {
		t.Fatalf("execute run: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(reportDir, "os7nh-test.json"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var rep report
	if err := json.Unmarshal(content, &rep); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if rep.Metadata.Model != "gpt-5.4-mini" || rep.Metadata.RawLogsCommitted {
		t.Fatalf("metadata = %+v", rep.Metadata)
	}
	markdown, err := os.ReadFile(filepath.Join(reportDir, "os7nh-test.md"))
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	for _, want := range []string{"OpenStudy Agent Eval", "Production Gate", "<run-root>/production/missing-field-rejection/events.jsonl"} {
		if !strings.Contains(string(markdown), want) {
			t.Fatalf("markdown missing %q:\n%s", want, string(markdown))
		}
	}
}

func TestExecuteRunReturnsErrorWhenProductionGateFails(t *testing.T) {
	reportDir := filepath.Join(t.TempDir(), "reports")
	config := runConfig{
		Parallel:   1,
		Variant:    productionVariant,
		Scenario:   strings.Join(releaseBlockingScenarioIDs(), ","),
		RunRoot:    filepath.Join(t.TempDir(), "run"),
		ReportDir:  reportDir,
		ReportName: "os7nh-failed",
		RepoRoot:   ".",
		CodexBin:   "codex",
		CacheMode:  cacheModeIsolated,
	}
	err := executeRun(context.Background(), config, &strings.Builder{}, func(_ context.Context, _ runConfig, job evalJob, _ cacheConfig) jobResult {
		now := time.Now().UTC()
		return jobResult{
			Variant:                 job.Variant,
			Scenario:                job.Scenario.ID,
			ScenarioTitle:           job.Scenario.Title,
			Passed:                  job.Scenario.ID != scenarioBypassRejection,
			Status:                  "completed",
			Metrics:                 metrics{AssistantCalls: 1, EventTypeCounts: map[string]int{}},
			Verification:            verificationResult{Passed: job.Scenario.ID != scenarioBypassRejection, DatabasePass: true, AssistantPass: true},
			PhaseTimings:            phaseTimings{AgentRun: 0.25, Total: 0.30},
			WallSeconds:             0.25,
			RawLogArtifactReference: "<run-root>/" + job.Variant + "/" + job.Scenario.ID + "/events.jsonl",
			StartedAt:               now,
			CompletedAt:             &now,
		}
	})
	if err == nil || !strings.Contains(err.Error(), "production gate failed") {
		t.Fatalf("execute run error = %v, want production gate failure", err)
	}
	if _, err := os.Stat(filepath.Join(reportDir, "os7nh-failed.json")); err != nil {
		t.Fatalf("report was not written before failure: %v", err)
	}
}

type fakeDirEntry struct {
	dir bool
}

func (f fakeDirEntry) Name() string               { return "" }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func createReviewAttemptForTest(t *testing.T, dbPath string, rating string) {
	t.Helper()
	ctx := context.Background()
	rt, err := localruntime.Open(ctx, localruntime.Config{
		DatabasePath: dbPath,
		Now:          fixedClock(deterministicNow),
	})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer func() { _ = rt.Close() }()
	session, err := rt.Service.StartReviewSession(ctx, study.StartReviewSessionInput{})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	answeredAt, err := time.Parse(time.RFC3339, "2099-01-01T00:05:00Z")
	if err != nil {
		t.Fatalf("parse answered at: %v", err)
	}
	if _, err := rt.Service.RecordReview(ctx, study.RecordReviewInput{
		SessionID:  session.ID,
		CardID:     1,
		AnsweredAt: &answeredAt,
		AnswerText: ptr("checked current status"),
		Rating:     study.Rating(rating),
		Grader:     study.GraderSelf,
	}); err != nil {
		t.Fatalf("record review: %v", err)
	}
}

func ptr(value string) *string {
	return &value
}
