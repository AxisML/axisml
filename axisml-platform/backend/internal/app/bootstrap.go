package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/config"
	"github.com/axisml/axisml/axisml-platform/backend/internal/db"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	"github.com/axisml/axisml/axisml-platform/backend/pkg/logging"
)

// Bootstrap runs migrations then seeds the initial system-admin and imports the
// built-in `default` tenant (auth.md §2). It is idempotent: existing rows are
// left as-is.
func Bootstrap(ctx context.Context, cfg config.Config) error {
	log := logging.New(cfg.Log.Level, cfg.Log.Format)

	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	users := store.NewUserRepo(gormDB)
	tenants := store.NewTenantRepo(gormDB)

	if err := seedAdmin(ctx, cfg, users, log); err != nil {
		return err
	}
	if err := importDefaultTenant(ctx, cfg, tenants, log); err != nil {
		return err
	}
	return nil
}

func seedAdmin(ctx context.Context, cfg config.Config, users *store.UserRepo, log *slog.Logger) error {
	if _, err := users.GetByUsername(ctx, cfg.Bootstrap.Username); err == nil {
		log.Info("bootstrap: admin already present", "username", cfg.Bootstrap.Username)
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	hash, err := auth.HashPassword(cfg.Bootstrap.Password)
	if err != nil {
		return err
	}
	admin := &store.User{
		Username:           cfg.Bootstrap.Username,
		PasswordHash:       hash,
		MustChangePassword: true,
		IsSystemAdmin:      true,
		DisplayName:        "Administrator",
	}
	if err := users.Create(ctx, admin); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	log.Info("bootstrap: created system-admin", "username", cfg.Bootstrap.Username)
	return nil
}

// importDefaultTenant imports the System-seeded `default` tenant into the
// Platform tenants table. The Tenant CR + its namespace are owned by the System
// chart's seed.tenant (created before Platform installs); Platform discovers it
// via cluster-manager rather than creating its own, which would double-create.
func importDefaultTenant(ctx context.Context, cfg config.Config, tenants *store.TenantRepo, log *slog.Logger) error {
	if _, err := tenants.GetByIdentifier(ctx, config.DefaultTenant); err == nil {
		log.Info("bootstrap: default tenant already present", "tenant", config.DefaultTenant)
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	cm, err := clustermanager.New(cfg.System.ClusterManager, config.UpstreamTimeout)
	if err != nil {
		return err
	}
	ctx = auth.WithUser(ctx, cfg.Bootstrap.Username)
	t, err := cm.GetTenant(ctx, config.DefaultTenant)
	if err != nil {
		// The System chart's seed.tenant owns the default Tenant CR; it may not be
		// applied yet, or cluster-manager may be unreachable during init. Skip and
		// succeed — the row is imported on a later bootstrap/serve once the CR exists.
		log.Warn("bootstrap: default tenant not yet discoverable from cluster-manager; will import later",
			"tenant", config.DefaultTenant, "err", err)
		return nil
	}

	row := &store.Tenant{
		Identifier:          t.Name,
		KubernetesNamespace: t.Namespace.Name,
		DisplayName:         "Default",
		Description:         "Built-in tenant hosting public artifacts.",
		Owner:               cfg.Bootstrap.Username,
		LastModifiedBy:      cfg.Bootstrap.Username,
	}
	if err := tenants.Create(ctx, row); err != nil {
		return fmt.Errorf("import default tenant row: %w", err)
	}
	log.Info("bootstrap: imported default tenant from cluster-manager",
		"tenant", t.Name, "namespace", t.Namespace.Name)
	return nil
}
