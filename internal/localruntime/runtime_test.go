package localruntime

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenInitializesServiceAndDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "openstudy.sqlite")
	runtime, err := Open(context.Background(), Config{DatabasePath: dbPath})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer func() {
		_ = runtime.Close()
	}()

	if runtime.Paths.DatabasePath != dbPath {
		t.Fatalf("database path = %q, want %q", runtime.Paths.DatabasePath, dbPath)
	}
	if runtime.Service == nil {
		t.Fatal("expected service")
	}
}
