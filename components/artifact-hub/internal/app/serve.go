package app

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/db"
	"github.com/axisml/axisml/components/artifact-hub/internal/k8sclient"
	"github.com/axisml/axisml/components/artifact-hub/internal/metrics"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	"github.com/axisml/axisml/components/artifact-hub/pkg/logging"
)

// Serve boots the long-running artifacts service: HTTP API on every replica,
// GC worker behind leader election.
func Serve(ctx context.Context, cfg config.Config) error {
	log, err := logging.New(cfg.LogDevelopment)
	if err != nil {
		return err
	}
	ctrl.SetLogger(log)

	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	metrics.Register()

	mgr, err := k8sclient.NewManager(cfg)
	if err != nil {
		return err
	}

	modules, runnables, err := BuildModules(cfg, gormDB, mgr, log)
	if err != nil {
		return err
	}

	srv, err := server.New(server.Options{
		Addr:    cfg.APIBindAddress,
		Log:     log,
		Modules: modules,
		// Readiness intentionally checks PG only — not zot. zot flakes
		// shouldn't take artifacts out of rotation: most reads (resolve)
		// don't touch zot at all, and individual write paths (initiate /
		// complete) report their own backend errors. Per design §2.3.
		Ready: func(ctx context.Context) error {
			return db.Ping(ctx, gormDB)
		},
	})
	if err != nil {
		return err
	}
	if err := mgr.Add(srv); err != nil {
		return err
	}
	for _, r := range runnables {
		if err := mgr.Add(r); err != nil {
			return err
		}
	}

	if ctx == nil {
		ctx = ctrl.SetupSignalHandler()
	}
	return mgr.Start(ctx)
}
