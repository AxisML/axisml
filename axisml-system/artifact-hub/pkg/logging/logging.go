package logging

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a logr.Logger backed by zap, honoring the configured level
// (debug|info|warn|error) and format (json|console). An unrecognized level
// falls back to info; any format other than "console" uses structured JSON.
func New(level, format string) (logr.Logger, error) {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	if format == "console" {
		cfg.Encoding = "console"
		cfg.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	z, err := cfg.Build()
	if err != nil {
		return logr.Logger{}, err
	}
	return zapr.NewLogger(z), nil
}
