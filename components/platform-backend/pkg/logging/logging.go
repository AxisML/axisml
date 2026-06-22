// Package logging builds the process-wide slog.Logger. Platform is not a
// controller-runtime binary, so it logs through the standard library rather
// than zap/logr.
package logging

import (
	"log/slog"
	"os"
)

// New returns a slog.Logger. development=true uses a human-readable text
// handler at debug level; otherwise structured JSON at info level.
func New(development bool) *slog.Logger {
	if development {
		return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
