package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReleaseDocsAcceptsProductionDoc(t *testing.T) {
	if err := validateReleaseDocs(filepath.Join("..", "..", "..")); err != nil {
		t.Fatalf("validate production release docs: %v", err)
	}
}

func TestValidateReleaseVerificationDocRejectsMissingTerms(t *testing.T) {
	err := validateReleaseVerificationDoc("doc.md", "# Release Verification\n\nopenstudy_<version>_checksums.txt\n")
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %v, want missing term", err)
	}
}
