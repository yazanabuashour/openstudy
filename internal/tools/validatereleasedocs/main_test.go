package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReleaseDocsAcceptsProductionDoc(t *testing.T) {
	if err := validateReleaseDocs(filepath.Join("..", "..", ".."), ""); err != nil {
		t.Fatalf("validate production release docs: %v", err)
	}
}

func TestValidateTaggedReleaseDocsAcceptsProductionReleaseNotes(t *testing.T) {
	if err := validateReleaseDocs(filepath.Join("..", "..", ".."), "v0.1.0"); err != nil {
		t.Fatalf("validate tagged release docs: %v", err)
	}
}

func TestValidateReleaseNotesUsesRequestedTag(t *testing.T) {
	content := "# OpenStudy v0.1.1\n\n## Changed\n\n- Patch release.\n\n## Verification\n\n- docs/evals/results/os7nh-v0.1.1.md uses gpt-5.4-mini and openstudy_0.1.1_checksums.txt with attestations.\n"
	if err := validateReleaseNotes("notes.md", content, "v0.1.1"); err != nil {
		t.Fatalf("validate future-tag release notes: %v", err)
	}
}

func TestValidateReleaseVerificationDocRejectsMissingTerms(t *testing.T) {
	err := validateReleaseVerificationDoc("doc.md", "# Release Verification\n\nopenstudy_<version>_checksums.txt\n")
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %v, want missing term", err)
	}
}

func TestValidateReleaseNotesRejectsHardWrappedProse(t *testing.T) {
	content := "# OpenStudy v0.1.0\n\nThis line starts a paragraph.\nThis line hard-wraps it.\n\n## Changed\n\n- Added release notes.\n\n## Verification\n\n- docs/evals/results/os7nh-v0.1.0.md uses gpt-5.4-mini and openstudy_0.1.0_checksums.txt with attestations.\n"
	err := validateReleaseNotes("notes.md", content, "v0.1.0")
	if err == nil || !strings.Contains(err.Error(), "hard-wrap") {
		t.Fatalf("error = %v, want hard-wrap rejection", err)
	}
}
