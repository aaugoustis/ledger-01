// Command migrate applies the embedded goose migrations.
//
//	go run ./cmd/migrate up|down|status
//	go run ./cmd/migrate create <name>
//
// On AWS Lambda (AWS_LAMBDA_RUNTIME_API set) it runs "up" once per invocation, so a deploy step
// can invoke it; migrations never run on API cold start.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/db"
	"github.com/pressly/goose/v3"
)

func main() {
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(func(ctx context.Context) (string, error) { return migrate(ctx, "up") })
		return
	}
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd == "create" {
		if len(os.Args) < 3 {
			fail(fmt.Errorf("usage: migrate create <name>"))
		}
		goose.SetSequential(true)
		if err := goose.Create(nil, filepath.Join("internal", "db", "migrations"), os.Args[2], "sql"); err != nil {
			fail(err)
		}
		return
	}
	out, err := migrate(context.Background(), cmd)
	if err != nil {
		fail(err)
	}
	fmt.Print(out)
}

func migrate(ctx context.Context, cmd string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL, 2)
	if err != nil {
		return "", err
	}
	defer pool.Close()

	var out string
	switch cmd {
	case "up":
		res, err := db.MigrateUp(ctx, pool)
		if err != nil {
			return "", err
		}
		for _, r := range res {
			out += fmt.Sprintf("applied %s (%s)\n", r.Source.Path, r.Duration)
		}
		if out == "" {
			out = "no pending migrations\n"
		}
	case "down":
		r, err := db.MigrateDown(ctx, pool)
		if err != nil {
			return "", err
		}
		out = fmt.Sprintf("rolled back %s\n", r.Source.Path)
	case "status":
		st, err := db.MigrationStatus(ctx, pool)
		if err != nil {
			return "", err
		}
		for _, s := range st {
			out += fmt.Sprintf("%-8s %s\n", s.State, s.Source.Path)
		}
	default:
		return "", fmt.Errorf("unknown command %q (up, down, status, create)", cmd)
	}
	return out, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "migrate:", err)
	os.Exit(1)
}
