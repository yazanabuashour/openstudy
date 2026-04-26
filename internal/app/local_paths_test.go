package app

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveLocalPathsUsesExplicitDatabasePath(t *testing.T) {
	dataDir, databasePath, err := resolveLocalPaths(LocalPathConfig{
		DatabasePath: "custom/openstudy.sqlite",
	}, localPathRuntime{
		getenv:      func(string) string { return "/ignored" },
		userHomeDir: func() (string, error) { return "/ignored-home", nil },
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if databasePath != filepath.Clean("custom/openstudy.sqlite") {
		t.Fatalf("databasePath = %q", databasePath)
	}
	if dataDir != "custom" {
		t.Fatalf("dataDir = %q", dataDir)
	}
}

func TestResolveLocalPathsUsesEnvOverride(t *testing.T) {
	dataDir, databasePath, err := resolveLocalPaths(LocalPathConfig{}, localPathRuntime{
		getenv: func(key string) string {
			if key == EnvDatabasePath {
				return "/tmp/openstudy/custom.sqlite"
			}
			return ""
		},
		userHomeDir: func() (string, error) { return "/ignored-home", nil },
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if databasePath != "/tmp/openstudy/custom.sqlite" {
		t.Fatalf("databasePath = %q", databasePath)
	}
	if dataDir != "/tmp/openstudy" {
		t.Fatalf("dataDir = %q", dataDir)
	}
}

func TestResolveLocalPathsUsesXDGDataHome(t *testing.T) {
	dataDir, databasePath, err := resolveLocalPaths(LocalPathConfig{}, localPathRuntime{
		getenv: func(key string) string {
			if key == "XDG_DATA_HOME" {
				return "/tmp/data-home"
			}
			return ""
		},
		userHomeDir: func() (string, error) { return "/home/tester", nil },
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if dataDir != "/tmp/data-home/openstudy" {
		t.Fatalf("dataDir = %q", dataDir)
	}
	if databasePath != "/tmp/data-home/openstudy/openstudy.sqlite" {
		t.Fatalf("databasePath = %q", databasePath)
	}
}

func TestResolveLocalPathsUsesHomeFallback(t *testing.T) {
	dataDir, databasePath, err := resolveLocalPaths(LocalPathConfig{}, localPathRuntime{
		getenv:      func(string) string { return "" },
		userHomeDir: func() (string, error) { return "/home/tester", nil },
	})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if dataDir != "/home/tester/.local/share/openstudy" {
		t.Fatalf("dataDir = %q", dataDir)
	}
	if databasePath != "/home/tester/.local/share/openstudy/openstudy.sqlite" {
		t.Fatalf("databasePath = %q", databasePath)
	}
}

func TestResolveLocalPathsPropagatesHomeDirErrors(t *testing.T) {
	_, _, err := resolveLocalPaths(LocalPathConfig{}, localPathRuntime{
		getenv:      func(string) string { return "" },
		userHomeDir: func() (string, error) { return "", errors.New("home unavailable") },
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
