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
// System-defined tenants (auth.md §2). It is idempotent: existing rows are left
// as-is.
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
	if err := importTenants(ctx, cfg, tenants, log); err != nil {
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

// importTenants imports every System-defined tenant into the Platform tenants
// table. The Tenant CRs are owned by the System layer and created before
// Platform installs — Standard's chart seeds the built-in `default`; Lite's
// axisml-core serves one or more read-only tenants from its static config.
// Platform discovers them via cluster-manager rather than creating its own,
// which would double-create. Idempotent: tenants already in the table are left
// as-is.
func importTenants(ctx context.Context, cfg config.Config, tenants *store.TenantRepo, log *slog.Logger) error {
	cm, err := clustermanager.New(cfg.System.ClusterManager, config.UpstreamTimeout)
	if err != nil {
		return err
	}
	ctx = auth.WithUser(ctx, cfg.Bootstrap.Username)
	list, err := cm.ListTenants(ctx, "")
	if err != nil {
		// The System layer owns the Tenant CRs; they may not be applied yet, or
		// cluster-manager may be unreachable during init. Skip and succeed — the
		// rows are imported on a later bootstrap/serve once the tenants exist.
		log.Warn("bootstrap: tenants not yet discoverable from cluster-manager; will import later", "err", err)
		return nil
	}

	for i := range list {
		t := &list[i]
		if _, err := tenants.GetByIdentifier(ctx, t.Name); err == nil {
			log.Info("bootstrap: tenant already present", "tenant", t.Name)
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}

		displayName, description := t.Name, ""
		if t.Name == config.DefaultTenant {
			displayName, description = "Default", "Built-in tenant hosting public artifacts."
		}
		row := &store.Tenant{
			Identifier:          t.Name,
			KubernetesNamespace: t.Namespace.Name,
			DisplayName:         displayName,
			Description:         description,
			Owner:               cfg.Bootstrap.Username,
			LastModifiedBy:      cfg.Bootstrap.Username,
		}
		if err := tenants.Create(ctx, row); err != nil {
			return fmt.Errorf("import tenant row %q: %w", t.Name, err)
		}
		log.Info("bootstrap: imported tenant from cluster-manager",
			"tenant", t.Name, "namespace", t.Namespace.Name)
	}
	return nil
}
