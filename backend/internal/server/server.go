// Package server builds the API's single http.Handler. cmd/api serves it with net/http locally;
// cmd/lambda wraps the very same handler for API Gateway on AWS Lambda.
package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/franco-roura/ledger-01/backend/gen/ledger/v1/ledgerv1connect"
	"github.com/franco-roura/ledger-01/backend/internal/auth"
	"github.com/franco-roura/ledger-01/backend/internal/config"
	"github.com/franco-roura/ledger-01/backend/internal/db/dbq"
	"github.com/franco-roura/ledger-01/backend/internal/ledger"
	"github.com/franco-roura/ledger-01/backend/internal/mcp"
	"github.com/franco-roura/ledger-01/backend/internal/oauth"
	"github.com/franco-roura/ledger-01/backend/internal/rpc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Config config.Config
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

// New wires services, auth and the Connect handlers into one http.Handler.
func New(d Deps) http.Handler {
	svc := ledger.New(d.Pool)
	sessions := auth.NewSessions(dbq.New(d.Pool), d.Config.SessionTTL)
	api := rpc.New(svc, d.Logger, d.Config.EmailAllowed)

	opts := connect.WithHandlerOptions(
		connect.WithInterceptors(sessions.Interceptor()),
		connect.WithReadMaxBytes(rpc.ReadMaxBytes),
	)
	mux := http.NewServeMux()
	mux.Handle(ledgerv1connect.NewUserServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewHouseholdServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewWalletServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewCategoryServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewLabelServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewTransactionServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewDashboardServiceHandler(api, opts))
	mux.Handle(ledgerv1connect.NewBudgetServiceHandler(api, opts))
	// Bank file uploads are the only large requests; their cap replaces the default one.
	mux.Handle(ledgerv1connect.NewImportServiceHandler(api, opts, connect.WithReadMaxBytes(rpc.ImportReadMaxBytes)))
	mux.Handle(ledgerv1connect.NewBackupServiceHandler(api, opts, connect.WithReadMaxBytes(rpc.BackupReadMaxBytes)))

	auth.NewHandler(d.Config, sessions, svc, d.Logger).Register(mux)

	// MCP clients (such as Claude) sign in through the OAuth server and call read-only tools on /mcp.
	store := oauth.NewStore(d.Pool)
	mux.Handle(ledgerv1connect.NewOAuthServiceHandler(rpc.NewOAuth(api, store, oauth.Issuer(d.Config)), opts))
	oauthHandler := oauth.NewHandler(d.Config, store, sessions, d.Logger)
	oauthHandler.Register(mux)
	mux.Handle("POST "+oauth.MCPPath, oauthHandler.Protect(mcp.New(svc, d.Logger)))
	// Stateless: there is no event stream to open or session to close.
	mux.HandleFunc(oauth.MCPPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	var h http.Handler = mux
	h = stripBasePath(d.Config.APIBasePath, h)
	h = recoverer(d.Logger, h)
	h = requestLog(d.Logger, h)
	return h
}

// stripBasePath removes a gateway/CloudFront prefix (e.g. "/api") when present. Paths without
// it (such as /auth/*, routed unprefixed) pass through untouched.
func stripBasePath(base string, next http.Handler) http.Handler {
	if base == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := strings.CutPrefix(r.URL.Path, base); ok && (p == "" || strings.HasPrefix(p, "/")) {
			r2 := r.Clone(r.Context())
			r2.URL.Path = p
			r2.URL.RawPath = ""
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverer(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.ErrorContext(r.Context(), "panic", "value", v, "stack", string(debug.Stack()))
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func requestLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		if r.URL.Path == "/healthz" {
			return
		}
		log.InfoContext(r.Context(), "request", "method", r.Method, "path", r.URL.Path, "status", sw.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}
