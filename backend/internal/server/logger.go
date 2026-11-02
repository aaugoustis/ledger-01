package server

import (
	"log/slog"
	"os"

	"github.com/franco-roura/ledger-01/backend/internal/config"
)

// NewLogger returns a text logger in development and JSON (CloudWatch-friendly) otherwise.
func NewLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.IsDev() {
		return slog.New(slog.NewTextHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
