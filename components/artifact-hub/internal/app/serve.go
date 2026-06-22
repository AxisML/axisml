package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/artifact-hub/internal/config"
	"github.com/axisml/axisml/components/artifact-hub/internal/db"
	"github.com/axisml/axisml/components/artifact-hub/internal/gc"
	"github.com/axisml/axisml/components/artifact-hub/internal/leaderelection"
	"github.com/axisml/axisml/components/artifact-hub/internal/metrics"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	"github.com/axisml/axisml/components/artifact-hub/pkg/httpx"
	"github.com/axisml/axisml/components/artifact-hub/pkg/logging"
)

// Serve boots the long-running artifacts service: the HTTP API, the metrics
// and probe listeners on every replica, and the GC worker behind a Postgres
// advisory-lock leader election. It owns no Kubernetes client — Postgres is
// both the source of truth and the leader-election backend.
func Serve(ctx context.Context, cfg config.Config) error {
	log, err := logging.New(cfg.LogDevelopment)
	if err != nil {
		return err
	}

	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	metrics.Register()

	modules, worker, err := BuildModules(cfg, gormDB, log)
	if err != nil {
		return err
	}

	// Readiness intentionally checks PG only — not zot. zot flakes shouldn't
	// take artifacts out of rotation: most reads (resolve) don't touch zot at
	// all, and individual write paths (initiate / complete) report their own
	// backend errors. Per design §2.3.
	ready := func(ctx context.Context) error { return db.Ping(ctx, gormDB) }

	srv, err := server.New(server.Options{
		Addr:    cfg.APIBindAddress,
		Log:     log,
		Modules: modules,
		Ready:   ready,
	})
	if err != nil {
		return err
	}

	if ctx == nil {
		var stop context.CancelFunc
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error { return srv.Start(gctx) })
	g.Go(func() error {
		return httpx.Serve(gctx, cfg.MetricsBindAddress, metrics.Handler(), log, "metrics")
	})
	g.Go(func() error {
		return httpx.Serve(gctx, cfg.ProbesBindAddress, server.ProbesHandler(ready), log, "probes")
	})
	g.Go(func() error { return runGC(gctx, cfg, gormDB, worker, log) })

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// runGC runs the GC worker, gated behind leader election when enabled. With
// election disabled (single replica / local dev) the worker runs directly.
func runGC(ctx context.Context, cfg config.Config, gormDB *gorm.DB, worker *gc.Worker, log logr.Logger) error {
	if !cfg.LeaderElect {
		return worker.Start(ctx)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("sql db handle for leader election: %w", err)
	}
	return leaderelection.Run(ctx, leaderelection.Config{
		DB:    sqlDB,
		Key:   cfg.LeaderLockKey,
		Retry: cfg.LeaderRetryPeriod,
		Log:   log.WithName("leader-election"),
	}, worker.Start)
}
