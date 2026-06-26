// Package logging builds the process-wide slog.Logger. Platform is not a
// controller-runtime binary, so it logs through the standard library rather
// than zap/logr.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger honoring the configured level (debug|info|warn|error)
// and format (json|console). An unrecognized level falls back to info; any
// format other than "console" uses structured JSON.
func New(level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	if format == "console" {
		return slog.New(slog.NewTextHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
