// Package db opens the Postgres pool and owns the embedded goose migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open creates a connection pool. On Lambda it is called once per cold start, so keep maxConns
// small: every concurrent instance holds its own pool.
func Open(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	return pool, nil
}

func provider(pool *pgxpool.Pool) (*goose.Provider, *sql.DB, error) {
	sqlDB := stdlib.OpenDBFromPool(pool)
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, mustSub(migrations, "migrations"))
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return p, sqlDB, nil
}

// MigrateUp applies every pending migration.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) ([]*goose.MigrationResult, error) {
	p, sqlDB, err := provider(pool)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()
	return p.Up(ctx)
}

// MigrateDown rolls back the most recent migration.
func MigrateDown(ctx context.Context, pool *pgxpool.Pool) (*goose.MigrationResult, error) {
	p, sqlDB, err := provider(pool)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()
	return p.Down(ctx)
}

// MigrationStatus lists every migration and whether it is applied.
func MigrationStatus(ctx context.Context, pool *pgxpool.Pool) ([]*goose.MigrationStatus, error) {
	p, sqlDB, err := provider(pool)
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()
	return p.Status(ctx)
}
