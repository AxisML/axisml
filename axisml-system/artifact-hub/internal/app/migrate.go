package app

import (
	"context"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/config"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/db"
)

// Migrate applies pending DB migrations and exits.
func Migrate(_ context.Context, cfg config.Config) error {
	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	return db.Migrate(gormDB)
}
