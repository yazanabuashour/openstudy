package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestApplyMigrationsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := Open(filepath.Join(t.TempDir(), "data", "openstudy.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("apply migrations first time: %v", err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("apply migrations second time: %v", err)
	}

	pending, err := PendingMigrations(ctx, db)
	if err != nil {
		t.Fatalf("pending migrations: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending migrations = %d, want 0", len(pending))
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE name = '0001_study_schema.sql'`).Scan(&count); err != nil {
		t.Fatalf("count migration row: %v", err)
	}
	if count != 1 {
		t.Fatalf("migration row count = %d, want 1", count)
	}
}

func TestOpenCreatesDatabaseDirectory(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "nested", "data", "openstudy.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
}
