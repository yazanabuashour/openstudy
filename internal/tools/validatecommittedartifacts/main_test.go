package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTrackedPathRejectsSQLiteAndRawLogs(t *testing.T) {
	for _, rel := range []string{
		"tmp/openstudy.sqlite",
		"tmp/openstudy.db",
		"docs/evals/results/events.jsonl",
		"docs/evals/raw-logs/run.json",
	} {
		if err := validateTrackedPath(rel); err == nil {
			t.Fatalf("validateTrackedPath(%q) succeeded, want error", rel)
		}
	}
	if err := validateTrackedPath("docs/evals/results/README.md"); err != nil {
		t.Fatalf("validateTrackedPath README: %v", err)
	}
}

func TestValidatePublicArtifactTextRejectsAbsolutePathsAndUnplacedRawLogs(t *testing.T) {
	if err := validatePublicArtifactText("docs/evals/results/report.md", "Raw log: /tmp/run/events.jsonl"); err == nil || !strings.Contains(err.Error(), "<run-root>") {
		t.Fatalf("raw log error = %v", err)
	}
	absolutePath := filepath.Join(string(os.PathSeparator), "Users", "example", "openstudy")
	if err := validatePublicArtifactText("docs/evals/results/report.md", "Path: "+absolutePath); err == nil || !strings.Contains(err.Error(), "machine-absolute") {
		t.Fatalf("absolute path error = %v", err)
	}
	if err := validatePublicArtifactText("docs/evals/results/report.md", "Raw log: <run-root>/production/events.jsonl"); err != nil {
		t.Fatalf("placeholder raw log: %v", err)
	}
}

func TestValidateCommittedArtifactsAcceptsFixtureRepo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "# Fixture\n\nRaw log: <run-root>/production/example/events.jsonl\n")
	runGit(t, root, "init")
	runGit(t, root, "add", "README.md")
	if err := validateCommittedArtifacts(root); err != nil {
		t.Fatalf("validate committed artifacts: %v", err)
	}
}

func writeFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	// Tests use the system Git only to exercise git ls-files behavior.
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
