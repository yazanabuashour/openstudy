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
		return errors.New("usage: scripts/validate-agent-skill.sh")
	}
	if err := validateAgentSkill("."); err != nil {
		return err
	}
	_, err := fmt.Fprintln(stdout, "validated OpenStudy agent skill")
	return err
}

func validateAgentSkill(root string) error {
	path := filepath.Join(root, "skills", "openstudy", "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read skills/openstudy/SKILL.md: %w", err)
	}
	body := strings.ReplaceAll(string(content), "\r\n", "\n")
	required := []string{
		"---",
		"name: OpenStudy",
		"description:",
		"license: MIT",
		"compatibility:",
		"openstudy cards",
		"openstudy review",
		"openstudy sources",
		"openstudy windows",
		"direct SQLite",
		"HTTP",
		"MCP",
		"source-built",
		"raw database reads",
		"ad hoc scripts",
		"unsupported transports",
		"card.front",
		"card.back",
		"rating",
		"grader",
		"Source references are provenance pointers only",
		"Record only explicit grades",
	}
	for _, want := range required {
		if !strings.Contains(body, want) {
			return fmt.Errorf("skills/openstudy/SKILL.md missing %q", want)
		}
	}
	disallowed := []string{
		"query SQLite directly",
		"read the database directly",
		"HTTP endpoint",
		"MCP server",
		"go run ./cmd/openstudy",
	}
	for _, text := range disallowed {
		if strings.Contains(body, text) {
			return fmt.Errorf("skills/openstudy/SKILL.md documents unsupported workflow %q", text)
		}
	}
	return nil
}
