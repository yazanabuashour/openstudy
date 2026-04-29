package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultParallel   = 4
	modelName         = "gpt-5.4-mini"
	reasoningEffort   = "medium"
	productionVariant = "production"
	cacheModeShared   = "shared"
	cacheModeIsolated = "isolated"
)

func run(args []string, stdout io.Writer, stderr io.Writer, runner jobRunner) int {
	if len(args) == 0 || args[0] != "run" {
		_, _ = fmt.Fprintln(stderr, "usage: os7nh run [--parallel N] [--variant ids] [--scenario ids] [--run-root path] [--report-dir path] [--report-name name] [--codex-bin path] [--cache-mode shared|isolated]")
		return 2
	}
	config, err := parseRunConfig(args[1:], stderr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := executeRun(context.Background(), config, stdout, runner); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseRunConfig(args []string, stderr io.Writer) (runConfig, error) {
	fs := flag.NewFlagSet("os7nh run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	config := runConfig{CacheMode: cacheModeShared}
	fs.IntVar(&config.Parallel, "parallel", defaultParallel, "number of independent eval jobs to run concurrently")
	fs.StringVar(&config.Variant, "variant", "", "comma-separated variant ids")
	fs.StringVar(&config.Scenario, "scenario", "", "comma-separated scenario ids")
	fs.StringVar(&config.RunRoot, "run-root", "", "directory for isolated run artifacts")
	fs.StringVar(&config.ReportDir, "report-dir", filepath.Join("docs", "evals", "results"), "directory for reduced reports")
	fs.StringVar(&config.ReportName, "report-name", "os7nh-latest", "base filename for reduced reports")
	fs.StringVar(&config.CodexBin, "codex-bin", "codex", "codex executable")
	fs.StringVar(&config.RepoRoot, "repo-root", ".", "repository root to copy for each job")
	fs.StringVar(&config.CacheMode, "cache-mode", config.CacheMode, "Go cache mode: shared or isolated")
	if err := fs.Parse(args); err != nil {
		return runConfig{}, err
	}
	if fs.NArg() != 0 {
		return runConfig{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if config.Parallel < 1 {
		return runConfig{}, errors.New("--parallel must be at least 1")
	}
	if config.CacheMode != cacheModeShared && config.CacheMode != cacheModeIsolated {
		return runConfig{}, fmt.Errorf("--cache-mode must be %q or %q", cacheModeShared, cacheModeIsolated)
	}
	if strings.TrimSpace(config.ReportName) == "" {
		return runConfig{}, errors.New("--report-name must not be empty")
	}
	if config.RunRoot == "" {
		config.RunRoot = filepath.Join(os.TempDir(), fmt.Sprintf("openstudy-os7nh-%d", time.Now().UnixNano()))
	}
	return config, nil
}

func selectedVariants(config runConfig) []string {
	if strings.TrimSpace(config.Variant) == "" {
		return []string{productionVariant}
	}
	parts := splitCSV(config.Variant)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == productionVariant {
			out = append(out, part)
		}
	}
	return out
}

func selectedScenarioIDs(config runConfig) []string {
	scenarios := selectedScenarios(config)
	out := make([]string, 0, len(scenarios))
	for _, sc := range scenarios {
		out = append(out, sc.ID)
	}
	return out
}

func selectedScenarios(config runConfig) []scenario {
	all := allScenarios()
	if strings.TrimSpace(config.Scenario) == "" {
		return all
	}
	allowed := map[string]bool{}
	for _, id := range splitCSV(config.Scenario) {
		allowed[id] = true
	}
	out := []scenario{}
	for _, sc := range all {
		if allowed[sc.ID] {
			out = append(out, sc)
		}
	}
	return out
}

func splitCSV(value string) []string {
	parts := []string{}
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
