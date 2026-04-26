package localruntime

import (
	"context"
	"database/sql"
	"time"

	"github.com/yazanabuashour/openstudy/internal/app"
	"github.com/yazanabuashour/openstudy/internal/storage/sqlite"
	"github.com/yazanabuashour/openstudy/internal/study"
)

const EnvDatabasePath = app.EnvDatabasePath

type Config struct {
	DatabasePath string
	Now          func() time.Time
}

type Paths struct {
	DataDir      string
	DatabasePath string
}

type Runtime struct {
	DB      *sql.DB
	Service *study.Service
	Paths   Paths
}

func ResolvePaths(config Config) (Paths, error) {
	dataDir, databasePath, err := app.ResolveLocalPaths(app.LocalPathConfig{
		DatabasePath: config.DatabasePath,
	})
	if err != nil {
		return Paths{}, err
	}
	return Paths{DataDir: dataDir, DatabasePath: databasePath}, nil
}

func Open(ctx context.Context, config Config) (*Runtime, error) {
	paths, err := ResolvePaths(config)
	if err != nil {
		return nil, err
	}
	db, err := sqlite.Open(paths.DatabasePath)
	if err != nil {
		return nil, err
	}
	if err := sqlite.ApplyMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	repo := sqlite.NewRepository(db)
	opts := []study.Option{}
	if config.Now != nil {
		opts = append(opts, study.WithClock(config.Now))
	}
	return &Runtime{
		DB:      db,
		Service: study.NewService(repo, opts...),
		Paths:   paths,
	}, nil
}

func (r *Runtime) Close() error {
	if r == nil || r.DB == nil {
		return nil
	}
	return r.DB.Close()
}
