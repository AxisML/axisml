package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/axisml/axisml/components/platform/internal/config"
	"github.com/axisml/axisml/components/platform/internal/db"
	"github.com/axisml/axisml/components/platform/internal/store"
	"github.com/axisml/axisml/components/platform/pkg/logging"
)

// Serve boots the API server and the probes server, applying migrations first.
func Serve(ctx context.Context, cfg config.Config) error {
	log := logging.New(cfg.Log.Level, cfg.Log.Format)

	gormDB, err := db.Open(cfg)
	if err != nil {
		return err
	}
	if err := db.Migrate(gormDB); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	api, err := NewAPIServer(cfg, gormDB, log)
	if err != nil {
		return err
	}
	probes := probeServer(config.ProbesBindAddress)

	go sweepExpiredSessions(ctx, store.NewSessionRepo(gormDB), config.SessionSweepInterval, log)

	errCh := make(chan error, 2)
	go func() {
		log.Info("probes listening", "addr", config.ProbesBindAddress)
		if err := probes.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("probes: %w", err)
		}
	}()
	go func() {
		if err := api.Start(ctx); err != nil {
			errCh <- fmt.Errorf("api: %w", err)
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		_ = probes.Close()
		if err != nil {
			return err
		}
	}
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = probes.Shutdown(shutCtx)
	return nil
}

// sweepExpiredSessions periodically purges expired session rows from PostgreSQL
// (Redis entries self-expire via TTL). It runs until ctx is cancelled.
func sweepExpiredSessions(ctx context.Context, sessions *store.SessionRepo, every time.Duration, log *slog.Logger) {
	if every <= 0 {
		return
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := sessions.DeleteExpired(ctx)
			if err != nil {
				log.Warn("session sweep failed", "error", err)
				continue
			}
			if n > 0 {
				log.Info("session sweep purged expired rows", "count", n)
			}
		}
	}
}
