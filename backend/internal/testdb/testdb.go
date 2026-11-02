// Package testdb starts a throwaway Postgres (testcontainers) with migrations applied, shared by
// every test in a package. Tests need Docker running.
package testdb

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/franco-roura/ledger-01/backend/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Image matches docker-compose.yml so tests run against the same Postgres as development.
const Image = "postgres:18.6-alpine"

var (
	once    sync.Once
	pool    *pgxpool.Pool
	dsn     string
	initErr error
)

// Pool returns a migrated pool backed by a container started once per test binary. The container
// is reaped by testcontainers' Ryuk sidecar when the process exits.
func Pool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	once.Do(func() {
		ctx := context.Background()
		var c *postgres.PostgresContainer
		c, initErr = postgres.Run(ctx, Image,
			postgres.WithDatabase("ledger_test"),
			postgres.WithUsername("ledger"),
			postgres.WithPassword("ledger"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second),
			),
		)
		if initErr != nil {
			return
		}
		dsn, initErr = c.ConnectionString(ctx, "sslmode=disable")
		if initErr != nil {
			return
		}
		pool, initErr = db.Open(ctx, dsn, 8)
		if initErr != nil {
			return
		}
		_, initErr = db.MigrateUp(ctx, pool)
	})
	if initErr != nil {
		if os.Getenv("CI") == "" {
			t.Skipf("Postgres test container unavailable (is Docker running?): %v", initErr)
		}
		t.Fatalf("start Postgres: %v", initErr)
	}
	return pool
}

// DSN returns the connection string of the shared test database.
func DSN(t testing.TB) string {
	Pool(t)
	return dsn
}
