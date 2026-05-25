package app

import (
	"context"
	"fmt"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	"github.com/axisml/axisml/components/compute-service/internal/db"
	poolmod "github.com/axisml/axisml/components/compute-service/internal/resourcepool"
)

// Bootstrap idempotently seeds the default ResourcePool. Tenant + Quota
// seeding belongs to cluster-manager + tenant-operator.
func Bootstrap(ctx context.Context, cfg config.Config) error {
	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	pools := poolmod.NewService(gormDB)
	if _, err := pools.EnsureDefault(ctx, cfg.BootstrapPool); err != nil {
		return fmt.Errorf("ensure pool: %w", err)
	}
	return nil
}
