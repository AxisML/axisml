package app

import (
	"context"
	"fmt"

	"github.com/axisml/axisml/components/compute/internal/config"
	"github.com/axisml/axisml/components/compute/internal/db"
	quotamod "github.com/axisml/axisml/components/compute/internal/quota"
	poolmod "github.com/axisml/axisml/components/compute/internal/resourcepool"
	tenantmod "github.com/axisml/axisml/components/compute/internal/tenant"
)

// Bootstrap idempotently seeds the default tenant / resource pool / quota.
// Invoked by the Helm post-install Job (and safe to re-run on upgrade).
func Bootstrap(ctx context.Context, cfg config.Config) error {
	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	pools := poolmod.NewService(gormDB)
	quotas := quotamod.NewService(gormDB, pools)
	tenants := tenantmod.NewService(gormDB, quotas, pools)

	pool, err := pools.EnsureDefault(ctx, cfg.BootstrapPool)
	if err != nil {
		return fmt.Errorf("ensure pool: %w", err)
	}
	tenant, err := tenants.EnsureDefault(ctx, cfg.BootstrapTenant, cfg.BootstrapTenantNamespace)
	if err != nil {
		return fmt.Errorf("ensure tenant: %w", err)
	}
	if _, err := quotas.EnsureDefault(ctx, tenant.ID, pool.ID, cfg.BootstrapResourceMax); err != nil {
		return fmt.Errorf("ensure quota: %w", err)
	}
	return nil
}
