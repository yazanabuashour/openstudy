package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: scripts/validate-release-docs.sh")
	}
	if err := validateReleaseDocs("."); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "validated release docs")
	return err
}

func validateReleaseDocs(root string) error {
	content, err := readRepoText(root, filepath.Join("docs", "release-verification.md"))
	if err != nil {
		return err
	}
	return validateReleaseVerificationDoc("docs/release-verification.md", content)
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
