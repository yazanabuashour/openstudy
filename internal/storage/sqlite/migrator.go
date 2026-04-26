package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	Name string
	SQL  string
}

func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	available, err := loadMigrations()
	if err != nil {
		return err
	}
	if err := ensureMigrationTable(ctx, db); err != nil {
		return err
	}

	pending, err := pendingMigrations(ctx, db, available)
	if err != nil {
		return err
	}
	for _, next := range pending {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, next.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", next.Name, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`,
			next.Name,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", next.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func PendingMigrations(ctx context.Context, db *sql.DB) ([]migration, error) {
	available, err := loadMigrations()
	if err != nil {
		return nil, err
	}
	if err := ensureMigrationTable(ctx, db); err != nil {
		return nil, err
	}
	return pendingMigrations(ctx, db, available)
}

func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  name TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
)`)
	return err
}

func pendingMigrations(ctx context.Context, db *sql.DB, available []migration) ([]migration, error) {
	applied, err := appliedMigrationNames(ctx, db)
	if err != nil {
		return nil, err
	}
	pending := make([]migration, 0, len(available))
	for _, next := range available {
		if _, ok := applied[next.Name]; !ok {
			pending = append(pending, next)
		}
	}
	return pending, nil
}

func appliedMigrationNames(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	names := map[string]struct{}{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration{Name: entry.Name(), SQL: string(body)})
	}
	slices.SortFunc(migrations, func(left, right migration) int {
		if left.Name < right.Name {
			return -1
		}
		if left.Name > right.Name {
			return 1
		}
		return 0
	})
	return migrations, nil
}
