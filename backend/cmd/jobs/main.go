// Command jobs runs Ledger/01's scheduled work, which the API never does in the background: it
// tops up recurring transactions so every rule has its occurrences pre-created through the
// horizon (today + 3 months), deletes expired bank-import uploads, and clears spent OAuth codes
// and expired tokens.
//
//	go run ./cmd/jobs              run once
//	go run ./cmd/jobs -every 1h    run now and then on a ticker (local only)
//
// On AWS Lambda (AWS_LAMBDA_RUNTIME_API set) each invocation runs once; an EventBridge schedule
// invokes it. Every run is idempotent, so overlapping or repeated runs are harmless.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/db"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/oauth"
	"github.com/franco-roura/ledger-01/backend/internal/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "jobs:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := server.NewLogger(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The pool outlives invocations on Lambda, like the API's.
	pool, err := db.Open(ctx, cfg.DatabaseURL, 2)
	if err != nil {
		return err
	}
	defer pool.Close()

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(func(ctx context.Context) (string, error) { return once(ctx, pool) })
		return nil
	}

	every := flag.Duration("every", 0, "run again on this interval (0 runs once)")
	flag.Parse()
	for {
		out, err := once(ctx, pool)
		if err != nil {
			log.Error("jobs failed", "err", err)
			if *every == 0 {
				return err
			}
		} else {
			log.Info("jobs done", "result", out)
		}
		if *every == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(*every):
		}
	}
}

// once runs every job a single time; a failing job doesn't stop the others.
func once(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	svc := ledger.New(pool)
	res, err := svc.TopUpRecurring(ctx, ledger.Today())
	purged, perr := svc.PurgeExpiredImports(ctx, time.Now())
	cleared, oerr := oauth.NewStore(pool).DeleteExpired(ctx)
	return fmt.Sprintf("recurring: %d rules topped up, %d occurrences created; imports: %d expired uploads deleted; oauth: %d expired rows deleted",
		res.Rules, res.Occurrences, purged, cleared), errors.Join(err, perr, oerr)
}
