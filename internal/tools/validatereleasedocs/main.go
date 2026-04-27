package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var releaseTagPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: scripts/validate-release-docs.sh [tag]")
	}
	tag := ""
	if len(args) == 1 {
		tag = strings.TrimSpace(args[0])
	}
	if err := validateReleaseDocs(".", tag); err != nil {
		return err
	}
	if tag != "" {
		_, err := fmt.Fprintf(stdout, "validated release docs for %s\n", tag)
		return err
	}
	_, err := fmt.Fprintln(stdout, "validated release docs")
	return err
}

func validateReleaseDocs(root string, tag string) error {
	content, err := readRepoText(root, filepath.Join("docs", "release-verification.md"))
	if err != nil {
		return err
	}
	if err := validateReleaseVerificationDoc("docs/release-verification.md", content); err != nil {
		return err
	}
	if tag == "" {
		return nil
	}
	return validateTaggedReleaseDocs(root, tag)
}

func readRepoText(root string, relPath string) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s not found", filepath.ToSlash(relPath))
		}
		return "", fmt.Errorf("read %s: %w", filepath.ToSlash(relPath), err)
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n"), nil
}

func validateReleaseVerificationDoc(path string, content string) error {
	normalized := strings.Join(strings.Fields(content), " ")
	required := []string{
		"openstudy_<version>_<os>_<arch>.tar.gz",
		"openstudy_<version>_skill.tar.gz",
		"openstudy_<version>_source.tar.gz",
		"openstudy_<version>_checksums.txt",
		"openstudy_<version>_sbom.json",
		"install.sh",
		"checksums",
		"SBOM",
		"attestations",
		"gpt-5.4-mini",
		"mise exec -- go run ./scripts/agent-eval/os7nh run",
		"immutable",
		"new patch release",
		"direct SQLite",
		"HTTP",
		"MCP",
		"source-built",
	}
	for _, want := range required {
		if !strings.Contains(normalized, want) {
			return fmt.Errorf("%s missing release verification term %q", path, want)
		}
	}
	return nil
}

func validateTaggedReleaseDocs(root string, tag string) error {
	if !releaseTagPattern.MatchString(tag) {
		return fmt.Errorf("tag must match vMAJOR.MINOR.PATCH: %q", tag)
	}
	notesPath := filepath.Join("docs", "release-notes", tag+".md")
	notes, err := readRepoText(root, notesPath)
	if err != nil {
		return err
	}
	if err := validateReleaseNotes(filepath.ToSlash(notesPath), notes, tag); err != nil {
		return err
	}
	changelog, err := readRepoText(root, "CHANGELOG.md")
	if err != nil {
		return err
	}
	releaseURL := "https://github.com/yazanabuashour/openstudy/releases/tag/" + tag
	if !strings.Contains(changelog, releaseURL) {
		return fmt.Errorf("CHANGELOG.md must link to %s", releaseURL)
	}
	return nil
}

func validateReleaseNotes(notesPath string, content string, tag string) error {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "# OpenStudy "+tag {
		return fmt.Errorf("%s must start with %q", notesPath, "# OpenStudy "+tag)
	}
	if !hasMarkdownHeading(lines, "## Changed") {
		return fmt.Errorf("%s must include ## Changed", notesPath)
	}
	if !hasMarkdownHeading(lines, "## Verification") {
		return fmt.Errorf("%s must include ## Verification", notesPath)
	}
	normalized := strings.Join(strings.Fields(content), " ")
	assetVersion := strings.TrimPrefix(tag, "v")
	required := []string{
		"docs/evals/results/os7nh-" + tag + ".md",
		"gpt-5.4-mini",
		"openstudy_" + assetVersion + "_checksums.txt",
		"attestations",
	}
	for _, want := range required {
		if !strings.Contains(normalized, want) {
			return fmt.Errorf("%s missing release-note term %q", notesPath, want)
		}
	}
	if err := validateNoHardWrappedProse(notesPath, lines); err != nil {
		return err
	}
	return nil
}

func hasMarkdownHeading(lines []string, heading string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == heading {
			return true
		}
	}
	return false
}

func validateNoHardWrappedProse(notesPath string, lines []string) error {
	inFence := false
	previousPlainLine := 0
	previousListLine := 0
	for i, line := range lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			previousPlainLine = 0
			previousListLine = 0
			continue
		}
		if inFence || trimmed == "" {
			previousPlainLine = 0
			previousListLine = 0
			continue
		}
		if previousListLine != 0 && !isListItemLine(line) && !isMarkdownSectionBoundary(trimmed) {
			return fmt.Errorf("%s line %d appears to hard-wrap list item from line %d; keep release-note list items on one source line", notesPath, lineNumber, previousListLine)
		}
		if isListItemLine(line) {
			previousPlainLine = 0
			previousListLine = lineNumber
			continue
		}
		previousListLine = 0
		if !isPlainProseLine(line) {
			previousPlainLine = 0
			continue
		}
		if previousPlainLine != 0 {
			return fmt.Errorf("%s line %d appears to hard-wrap prose from line %d; keep release-note prose paragraphs on one source line", notesPath, lineNumber, previousPlainLine)
		}
		previousPlainLine = lineNumber
	}
	return nil
}

func isPlainProseLine(line string) bool {
	if line == "" || strings.TrimSpace(line) == "" {
		return false
	}
	if line != strings.TrimLeft(line, " \t") {
		return false
	}
	trimmed := strings.TrimSpace(line)
	nonProsePrefixes := []string{
		"#",
		"- ",
		"* ",
		"+ ",
		">",
		"|",
		"```",
		"~~~",
		"[",
		"<!--",
	}
	for _, prefix := range nonProsePrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return false
		}
	}
	if isOrderedListItem(trimmed) || isMarkdownRule(trimmed) {
		return false
	}
	return true
}

func isListItemLine(line string) bool {
	if line != strings.TrimLeft(line, " \t") {
		return false
	}
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") ||
		strings.HasPrefix(trimmed, "+ ") ||
		isOrderedListItem(trimmed)
}

func isMarkdownSectionBoundary(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") || isMarkdownRule(trimmed) || strings.HasPrefix(trimmed, "<!--")
}

func isOrderedListItem(line string) bool {
	dot := strings.IndexByte(line, '.')
	if dot <= 0 || dot == len(line)-1 || line[dot+1] != ' ' {
		return false
	}
	for _, r := range line[:dot] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isMarkdownRule(line string) bool {
	if len(line) < 3 {
		return false
	}
	for _, marker := range []rune{'-', '*', '_'} {
		allMarker := true
		for _, r := range line {
			if r != marker {
				allMarker = false
				break
			}
		}
		if allMarker {
			return true
		}
	}
	return false
}
