package app

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvDatabasePath = "OPENSTUDY_DATABASE_PATH"
	defaultDBName   = "openstudy.sqlite"
)

type LocalPathConfig struct {
	DatabasePath string
}

type localPathRuntime struct {
	getenv      func(string) string
	userHomeDir func() (string, error)
}

func ResolveLocalPaths(config LocalPathConfig) (string, string, error) {
	return resolveLocalPaths(config, localPathRuntime{
		getenv:      os.Getenv,
		userHomeDir: os.UserHomeDir,
	})
}

func resolveLocalPaths(config LocalPathConfig, rt localPathRuntime) (string, string, error) {
	if config.DatabasePath != "" {
		databasePath := cleanPath(config.DatabasePath)
		return cleanPath(filepath.Dir(databasePath)), databasePath, nil
	}

	if databasePath := rt.getenv(EnvDatabasePath); databasePath != "" {
		databasePath = cleanPath(databasePath)
		return cleanPath(filepath.Dir(databasePath)), databasePath, nil
	}

	homeDir, err := rt.userHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve user home directory: %w", err)
	}

	dataDir := defaultDataDir(cleanPath(homeDir), rt.getenv)
	return dataDir, filepath.Join(dataDir, defaultDBName), nil
}

func defaultDataDir(homeDir string, getenv func(string) string) string {
	if xdgDataHome := getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(cleanPath(xdgDataHome), "openstudy")
	}
	return filepath.Join(homeDir, ".local", "share", "openstudy")
}

func cleanPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
