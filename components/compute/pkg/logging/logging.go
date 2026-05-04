package logging

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// New returns a logr.Logger backed by zap. development=true uses a
// human-friendly console encoder; otherwise structured JSON.
func New(development bool) (logr.Logger, error) {
	var z *zap.Logger
	var err error
	if development {
		z, err = zap.NewDevelopment()
	} else {
		z, err = zap.NewProduction()
	}
	if err != nil {
		return logr.Logger{}, err
	}
	return zapr.NewLogger(z), nil
}
