package app

import (
	"context"
	"fmt"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	"github.com/axisml/axisml/components/compute-service/internal/db"
)

// Bootstrap runs the embedded golang-migrate migrations. Per the new
// design, compute-service no longer seeds ResourcePool — the pool lives
// in the cluster-scoped ResourcePool CRD, owned by cluster-manager.
// Seed pool / unit / tenant rows are installed by the axisml-system Helm
// chart's post-install hooks.
func Bootstrap(_ context.Context, cfg config.Config) error {
	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
