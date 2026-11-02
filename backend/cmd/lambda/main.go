// Command lambda runs the Ledger/01 API on AWS Lambda behind API Gateway (HTTP API, payload v2).
// It serves exactly the same http.Handler as cmd/api; this is the only package (with cmd/migrate)
// that imports AWS libraries. Build: `task build:lambda` (linux/arm64, provided.al2023).
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/db"
	"github.com/franco-roura/ledger-01/backend/internal/server"
)

// Handler adapts an http.Handler to API Gateway v2 events.
type Handler func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

func newHandler(h http.Handler) Handler {
	return httpadapter.NewV2(h).ProxyWithContext
}

func main() {
	// Everything here runs once per cold start; the pool is reused across invocations.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	pool, err := db.Open(context.Background(), cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db:", err)
		os.Exit(1)
	}
	h := server.New(server.Deps{Config: cfg, Pool: pool, Logger: server.NewLogger(cfg)})
	lambda.Start(newHandler(h))
}
