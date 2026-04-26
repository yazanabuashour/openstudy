package skilltest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenStudySkillPolicy(t *testing.T) {
	body := readSkill(t)
	required := []string{
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
	}
	for _, want := range required {
		if !strings.Contains(body, want) {
			t.Fatalf("skill missing %q", want)
		}
	}

	disallowed := []string{
		"query SQLite directly",
		"read the database directly",
		"HTTP endpoint",
		"MCP server",
	}
	for _, text := range disallowed {
		if strings.Contains(body, text) {
			t.Fatalf("skill documents unsupported workflow %q", text)
		}
	}
}

func readSkill(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "skills", "openstudy", "SKILL.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	return string(body)
}
