package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	machineUnixPathPattern    = regexp.MustCompile(`/(Users|home)/[^/\s"'\\)]+`)
	machineWindowsPathPattern = regexp.MustCompile(`(?i)\b[A-Z]:\\Users\\[^\\\s"']+`)
)

var privateMarkers = []string{
	"private vault",
	"private study material",
	"workspace backup",
	"delivery history",
	"review history",
	"raw private log",
	"credential",
}

var forbiddenReferenceMarkers = []string{
	"open" + "health",
	"open" + "clerk",
	"open" + "brief",
}

var requiredExportIgnoreRules = []string{
	".beads/",
	".claude/",
	".dolt/",
	".agents/",
	"dist/",
	"bin/",
	"*.db",
	"*.sqlite",
	"*.sqlite3",
	"*.jsonl",
	".env",
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: scripts/validate-committed-artifacts.sh")
	}
	if err := validateCommittedArtifacts("."); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "validated committed artifacts")
	return err
}

func validateCommittedArtifacts(root string) error {
	if err := validateExportIgnoreRules(root); err != nil {
		return err
	}
	files, err := trackedFiles(root)
	if err != nil {
		return err
	}
	for _, rel := range files {
		if err := validateTrackedPath(rel); err != nil {
			return err
		}
		if !isPublicTextArtifact(rel) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		if err := validatePublicArtifactText(rel, strings.ReplaceAll(string(content), "\r\n", "\n")); err != nil {
			return err
		}
	}
	return nil
}

func trackedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list tracked files: %w", err)
	}
	parts := bytes.Split(output, []byte{0})
	files := []string{}
	for _, part := range parts {
		name := strings.TrimSpace(string(part))
		if name != "" {
			files = append(files, filepath.ToSlash(name))
		}
	}
	return files, nil
}

func validateTrackedPath(rel string) error {
	lower := strings.ToLower(filepath.ToSlash(rel))
	for _, suffix := range []string{".sqlite", ".sqlite3", ".db", ".db-wal", ".db-shm"} {
		if strings.HasSuffix(lower, suffix) {
			return fmt.Errorf("%s must not be committed", rel)
		}
	}
	if strings.HasSuffix(lower, ".jsonl") || strings.Contains(lower, "/events.jsonl") || strings.Contains(lower, "/raw-logs/") {
		return fmt.Errorf("%s appears to be raw eval log output", rel)
	}
	return nil
}

func isPublicTextArtifact(rel string) bool {
	rel = filepath.ToSlash(rel)
	ext := filepath.Ext(rel)
	switch ext {
	case ".md", ".json", ".yml", ".yaml", ".toml", ".txt", ".sh":
	default:
		return false
	}
	if strings.HasPrefix(rel, "docs/") ||
		strings.HasPrefix(rel, "skills/") ||
		strings.HasPrefix(rel, "scripts/") ||
		strings.HasPrefix(rel, ".github/") {
		return true
	}
	switch rel {
	case "AGENTS.md", "README.md", "CHANGELOG.md", "CONTRIBUTING.md", "SECURITY.md", "CODE_OF_CONDUCT.md":
		return true
	default:
		return false
	}
}

func validatePublicArtifactText(rel string, text string) error {
	lines := strings.Split(text, "\n")
	for lineIndex, line := range lines {
		lineNumber := lineIndex + 1
		if match := machineUnixPathPattern.FindString(line); match != "" && !strings.Contains(match, "/home/runner/") {
			return fmt.Errorf("%s:%d contains machine-absolute path %q", rel, lineNumber, match)
		}
		if match := machineWindowsPathPattern.FindString(line); match != "" {
			return fmt.Errorf("%s:%d contains machine-absolute path %q", rel, lineNumber, match)
		}
		if strings.Contains(line, "events.jsonl") && !containsRunRootPlaceholder(line) {
			return fmt.Errorf("%s:%d references raw eval logs without <run-root> placeholder", rel, lineNumber)
		}
		if strings.Contains(line, `"raw_logs_committed": true`) {
			return fmt.Errorf("%s:%d marks raw eval logs as committed", rel, lineNumber)
		}
		lower := strings.ToLower(line)
		for _, marker := range forbiddenReferenceMarkers {
			if strings.Contains(lower, marker) {
				return fmt.Errorf("%s:%d contains forbidden local reference marker %q", rel, lineNumber, marker)
			}
		}
		context := paragraphContext(lines, lineIndex)
		if isNeutralPolicyLine(context) {
			continue
		}
		for _, marker := range privateMarkers {
			if strings.Contains(lower, marker) {
				return fmt.Errorf("%s:%d contains private-data marker %q", rel, lineNumber, marker)
			}
		}
	}
	return nil
}

func validateExportIgnoreRules(root string) error {
	content, err := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New(".gitattributes not found")
		}
		return fmt.Errorf("read .gitattributes: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	rules := map[string]bool{}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "export-ignore" {
			rules[fields[0]] = true
		}
	}
	for _, required := range requiredExportIgnoreRules {
		if !rules[required] {
			return fmt.Errorf(".gitattributes missing export-ignore rule for %s", required)
		}
	}
	return nil
}

func paragraphContext(lines []string, index int) string {
	start := index
	for start > 0 && strings.TrimSpace(lines[start-1]) != "" {
		start--
	}
	end := index
	for end+1 < len(lines) && strings.TrimSpace(lines[end+1]) != "" {
		end++
	}
	return strings.ToLower(strings.Join(lines[start:end+1], " "))
}

func containsRunRootPlaceholder(line string) bool {
	return strings.Contains(line, "<run-root>/") ||
		strings.Contains(line, `\u003crun-root\u003e/`)
}

func isNeutralPolicyLine(lower string) bool {
	return strings.Contains(lower, "must not contain") ||
		strings.Contains(lower, "do not add") ||
		strings.Contains(lower, "do not include") ||
		strings.Contains(lower, "without including") ||
		strings.Contains(lower, "no private") ||
		strings.Contains(lower, "must not copy") ||
		strings.Contains(lower, "avoid") ||
		strings.Contains(lower, "review docs") ||
		strings.Contains(lower, "failure examples") ||
		strings.Contains(lower, "fixture-like data includes") ||
		strings.Contains(lower, "add private material") ||
		strings.Contains(lower, "reject") ||
		strings.Contains(lower, "redact") ||
		strings.Contains(lower, "private-data") ||
		strings.Contains(lower, "private data") ||
		strings.Contains(lower, "security") ||
		strings.Contains(lower, "vulnerability") ||
		strings.Contains(lower, "exposure") ||
		strings.Contains(lower, "risk") ||
		strings.Contains(lower, "leak") ||
		strings.Contains(lower, "import or model") ||
		strings.Contains(lower, "import or copy") ||
		strings.Contains(lower, "copied private")
}
