package app

import (
	"context"
	"fmt"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	"github.com/axisml/axisml/components/compute-service/internal/db"
)

// Bootstrap runs the embedded golang-migrate migrations. compute-service does
// not seed ResourcePool or Tenant — those live in the cluster-scoped
// ResourcePool / Tenant CRDs owned by cluster-manager. Seed pool / unit rows
// are installed by the axisml-system Helm chart's post-install hooks.
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
