package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/clients/clustermanager"
	"github.com/axisml/axisml/components/platform/internal/config"
	"github.com/axisml/axisml/components/platform/internal/db"
	"github.com/axisml/axisml/components/platform/internal/store"
	"github.com/axisml/axisml/components/platform/pkg/logging"
)

// Bootstrap runs migrations then seeds the initial system-admin and the built-in
// `default` tenant (auth.md §2). It is idempotent: existing rows are left as-is.
func Bootstrap(ctx context.Context, cfg config.Config) error {
	log := logging.New(cfg.LogDevelopment)

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
	if err := seedDefaultTenant(ctx, cfg, tenants, log); err != nil {
		return err
	}
	return nil
}

func seedAdmin(ctx context.Context, cfg config.Config, users *store.UserRepo, log *slog.Logger) error {
	if _, err := users.GetByUsername(ctx, cfg.BootstrapUsername); err == nil {
		log.Info("bootstrap: admin already present", "username", cfg.BootstrapUsername)
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	hash, err := auth.HashPassword(cfg.BootstrapPassword)
	if err != nil {
		return err
	}
	admin := &store.User{
		Username:           cfg.BootstrapUsername,
		PasswordHash:       hash,
		MustChangePassword: true,
		IsSystemAdmin:      true,
		DisplayName:        "Administrator",
	}
	if err := users.Create(ctx, admin); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	log.Info("bootstrap: created system-admin", "username", cfg.BootstrapUsername)
	return nil
}

func seedDefaultTenant(ctx context.Context, cfg config.Config, tenants *store.TenantRepo, log *slog.Logger) error {
	if _, err := tenants.GetByIdentifier(ctx, cfg.BootstrapTenant); err == nil {
		log.Info("bootstrap: default tenant already present", "tenant", cfg.BootstrapTenant)
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	row := &store.Tenant{
		Identifier:          cfg.BootstrapTenant,
		KubernetesNamespace: cfg.BootstrapTenantNS,
		DisplayName:         "Default",
		Description:         "Built-in tenant hosting public artifacts.",
		Owner:               cfg.BootstrapUsername,
		LastModifiedBy:      cfg.BootstrapUsername,
	}
	if err := tenants.Create(ctx, row); err != nil {
		return fmt.Errorf("seed default tenant row: %w", err)
	}

	cm, err := clustermanager.New(cfg.ClusterManagerURL, cfg.UpstreamTimeout)
	if err != nil {
		return err
	}
	ctx = auth.WithUser(ctx, cfg.BootstrapUsername)
	if _, err := cm.CreateTenant(ctx, clustermanager.CreateTenantInput{
		Name:          cfg.BootstrapTenant,
		NamespaceName: cfg.BootstrapTenantNS,
		DisplayName:   "Default",
	}); err != nil {
		// Best-effort: cluster-manager may not be reachable yet during init.
		// Roll back the durable row so the tenant can be (re)created later via
		// the API, and let bootstrap succeed so admin + migrations are in place.
		_ = tenants.Delete(ctx, cfg.BootstrapTenant)
		log.Warn("bootstrap: default tenant CR not yet materialised (cluster-manager unreachable?); create it later via the API",
			"tenant", cfg.BootstrapTenant, "err", err)
		return nil
	}
	log.Info("bootstrap: created default tenant", "tenant", cfg.BootstrapTenant, "namespace", cfg.BootstrapTenantNS)
	return nil
}
