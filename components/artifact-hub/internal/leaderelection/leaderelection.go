// Package leaderelection provides single-leader election backed by a
// PostgreSQL session-level advisory lock.
//
// artifact-hub is a PG-backed REST service that watches no CRDs, so its only
// cluster-singleton concern — the GC worker — is elected against the database
// that is already its sole source of truth, not against a Kubernetes Lease.
// This keeps the process free of client-go / controller-runtime and free of
// any Kubernetes RBAC.
//
// Mechanics: a dedicated *sql.Conn is pinned for the lifetime of leadership
// and holds the lock via pg_try_advisory_lock. Because advisory locks are
// session-scoped, a crashed leader's lock is released automatically by
// PostgreSQL when its connection drops — no lease TTL to tune. A watchdog
// pings the pinned connection so a surviving-but-partitioned leader abdicates
// promptly instead of running split-brain.
package leaderelection

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-logr/logr"
)

// Config configures the elector.
type Config struct {
	// DB is the pool to contend on. A single connection is pinned out of it
	// while leadership is held; the elected callback uses its own pool
	// connections, never this one.
	DB *sql.DB
	// Key is the advisory-lock key. All replicas of one logical service must
	// share the same key; distinct services on the same database must not
	// collide.
	Key int64
	// Retry is both the acquisition retry interval and the watchdog ping
	// cadence. Defaults to 2s when zero.
	Retry time.Duration
	Log   logr.Logger
}

// Run blocks until ctx is cancelled. Whenever this process holds the advisory
// lock it invokes fn with a context that is cancelled the instant leadership
// is lost (lock connection dropped) or ctx ends. If fn returns, leadership is
// released and re-contended on the next tick. Run itself returns nil on ctx
// cancellation; transient errors are logged and retried.
func Run(ctx context.Context, cfg Config, fn func(context.Context) error) error {
	retry := cfg.Retry
	if retry <= 0 {
		retry = 2 * time.Second
	}
	for ctx.Err() == nil {
		ran, err := attempt(ctx, cfg, retry, fn)
		if err != nil && ctx.Err() == nil {
			cfg.Log.Error(err, "leader election attempt failed; retrying")
		}
		// Whether we lost the contest or finished a leadership term, wait one
		// retry interval before contending again.
		_ = ran
		if !sleep(ctx, retry) {
			return nil
		}
	}
	return nil
}

// attempt contends once. It returns (true, ...) if leadership was won and fn
// ran to completion. fn's error is propagated only when leadership was held.
func attempt(ctx context.Context, cfg Config, retry time.Duration, fn func(context.Context) error) (bool, error) {
	conn, err := cfg.DB.Conn(ctx)
	if err != nil {
		return false, fmt.Errorf("acquire lock connection: %w", err)
	}
	defer conn.Close()

	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", cfg.Key).Scan(&locked); err != nil {
		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}
	if !locked {
		return false, nil
	}

	cfg.Log.Info("acquired leadership", "lockKey", cfg.Key)
	leaderCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Watchdog: the pinned connection is touched only here while fn runs, so
	// there is no concurrent use of conn. If the ping fails the lock is gone
	// (or unreachable) and we abdicate by cancelling leaderCtx.
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		t := time.NewTicker(retry)
		defer t.Stop()
		for {
			select {
			case <-leaderCtx.Done():
				return
			case <-t.C:
				if err := conn.PingContext(leaderCtx); err != nil && leaderCtx.Err() == nil {
					cfg.Log.Error(err, "lock connection lost; abdicating")
					cancel()
					return
				}
			}
		}
	}()

	runErr := fn(leaderCtx)

	// Stop the watchdog before reusing conn for the unlock so the connection
	// is never touched concurrently.
	cancel()
	<-watchDone

	// Best-effort explicit unlock; a dropped connection has already released
	// the lock server-side, so an error here is benign.
	uctx, ucancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = conn.ExecContext(uctx, "SELECT pg_advisory_unlock($1)", cfg.Key)
	ucancel()

	cfg.Log.Info("released leadership", "lockKey", cfg.Key)
	return true, runErr
}

// sleep waits d or until ctx is cancelled. Returns false if ctx ended.
func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
