// Package httpx hosts small HTTP helpers shared across the artifact-hub
// process. It carries no operator/controller-runtime machinery: artifact-hub
// is a PG-backed REST service, so each long-running listener is a plain
// net/http server orchestrated by the caller.
package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

// Serve runs h on addr until ctx is cancelled, then drains in-flight requests
// with a bounded grace period. It returns nil on graceful shutdown and a
// non-nil error only if the listener fails for a reason other than a normal
// close. The name is used purely for log lines (e.g. "metrics", "probes").
func Serve(ctx context.Context, addr string, h http.Handler, log logr.Logger, name string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info(name+" server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		return nil
	case err := <-errCh:
		return err
	}
}
