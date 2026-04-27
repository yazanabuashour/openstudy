package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAgentSkillAcceptsProductionSkill(t *testing.T) {
	if err := validateAgentSkill(filepath.Join("..", "..", "..")); err != nil {
		t.Fatalf("validate production skill: %v", err)
	}
}

func TestValidateAgentSkillRejectsBypassDocumentation(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "openstudy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	body := productionSkillFixture() + "\nAgents may query SQLite directly.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := validateAgentSkill(root); err == nil || !strings.Contains(err.Error(), "unsupported workflow") {
		t.Fatalf("validate error = %v, want unsupported workflow", err)
	}
}

func productionSkillFixture() string {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", "openstudy", "SKILL.md"))
	if err != nil {
		panic(err)
	}
	return string(body)
}
