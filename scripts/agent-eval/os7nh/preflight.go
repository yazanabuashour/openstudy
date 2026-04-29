package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func preflightEvalContext(repoRoot string, repoDir string, runDir string, paths evalPaths, cache cacheConfig, codexBin string) error {
	sourceSkill := filepath.Join(repoRoot, "skills", "openstudy", "SKILL.md")
	installedSkill := filepath.Join(repoDir, ".agents", "skills", "openstudy", "SKILL.md")
	sourceBytes, err := os.ReadFile(sourceSkill)
	if err != nil {
		return err
	}
	installedBytes, err := os.ReadFile(installedSkill)
	if err != nil {
		return err
	}
	if !bytes.Equal(sourceBytes, installedBytes) {
		return errors.New("installed production skill does not match shipped SKILL.md")
	}
	if _, err := os.Stat(filepath.Join(repoDir, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return errors.New("production eval repo must not contain AGENTS.md")
		}
		return err
	}
	cmd := exec.Command(codexBin, "debug", "prompt-input", "Use OpenStudy to list due cards.")
	cmd.Dir = repoDir
	cmd.Env = evalEnv(runDir, paths, cache)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	rendered := string(output)
	if !containsOpenStudySkillDiscovery(rendered) {
		return errors.New("rendered prompt is missing openstudy skill discovery")
	}
	if !strings.Contains(rendered, ".agents/skills/openstudy/SKILL.md") {
		return errors.New("rendered prompt does not point openstudy to the installed project skill")
	}
	if containsOpenStudyAgentsInstructions(rendered) {
		return errors.New("rendered prompt contains OpenStudy product instructions from AGENTS.md")
	}
	return nil
}

func containsOpenStudySkillDiscovery(rendered string) bool {
	return strings.Contains(rendered, "- OpenStudy:") || strings.Contains(rendered, "- openstudy:")
}

func containsOpenStudyAgentsInstructions(rendered string) bool {
	const marker = "# AGENTS.md instructions"
	index := strings.Index(rendered, marker)
	if index < 0 {
		return false
	}
	agentsText := strings.ToLower(rendered[index:])
	for _, forbidden := range []string{
		"openstudy",
		"direct sqlite",
		"source-built",
		"automation runtime",
		"planning-only",
	} {
		if strings.Contains(agentsText, forbidden) {
			return true
		}
	}
	return false
}
