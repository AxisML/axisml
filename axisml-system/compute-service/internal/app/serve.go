package app

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/config"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/db"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/k8sclient"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/metrics"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/promclient"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/logging"
)

// Serve boots the long-running compute service: HTTP API on every replica,
// reconcilers + informers behind leader election.
func Serve(ctx context.Context, cfg config.Config) error {
	log, err := logging.New(cfg.Log.Level, cfg.Log.Format)
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

	mgr, err := k8sclient.NewManager()
	if err != nil {
		return err
	}

	promProvider, err := promclient.New(cfg.Prometheus.URL)
	if err != nil {
		return fmt.Errorf("prometheus client: %w", err)
	}

	modules, runnables, caps, err := BuildModules(gormDB, mgr, log, promProvider)
	if err != nil {
		return err
	}

	srv, err := server.New(server.Options{
		Addr:         config.APIBindAddress,
		Log:          log,
		Modules:      modules,
		Capabilities: caps,
		Ready: func(ctx context.Context) error {
			sqlDB, err := gormDB.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
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
