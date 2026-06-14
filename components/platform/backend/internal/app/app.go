// Package app wires the platform-backend process: the HTTP API server, the
// health-probe server, and graceful shutdown. The main package's job is just
// flag parsing.
//
// Platform is the user-facing aggregator: it fronts the internal services
// (cluster-manager / compute / artifacts) over HTTP and adds identity, RBAC,
// and orchestration. It is NOT a Kubernetes controller — it holds no client to
// the API server and runs no reconciler.
//
// Status: the server is not implemented yet. NewRouter returns an engine with
// the health/version plumbing in place but no resource handlers; the API
// surface it will serve is declared (as DTOs) in internal/server and rendered
// to docs/openapi/platform.yaml. Resource routes land with the implementation.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/platform/backend/internal/server"
)

// Config groups the runtime knobs the binary exposes.
type Config struct {
	APIBindAddress    string
	ProbesBindAddress string
}

// DefaultConfig returns the standard listen addresses.
func DefaultConfig() Config {
	return Config{APIBindAddress: ":8080", ProbesBindAddress: ":8082"}
}

// Run is the long-running entry point. Cancel ctx (e.g. via a SIGTERM handler)
// to trigger a graceful HTTP shutdown.
func Run(ctx context.Context, cfg Config) error {
	api := &http.Server{Addr: cfg.APIBindAddress, Handler: NewRouter()}
	probes := &http.Server{Addr: cfg.ProbesBindAddress, Handler: NewProbeRouter()}

	errCh := make(chan error, 2)
	go serve(api, "api", errCh)
	go serve(probes, "probes", errCh)
	slog.Info("platform-backend listening", "api", cfg.APIBindAddress, "probes", cfg.ProbesBindAddress)

	select {
	case <-ctx.Done():
	case err := <-errCh:
		_ = api.Close()
		_ = probes.Close()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = api.Shutdown(shutdownCtx)
	_ = probes.Shutdown(shutdownCtx)
	return nil
}

func serve(s *http.Server, name string, errCh chan<- error) {
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- fmt.Errorf("%s: %w", name, err)
	}
}

// NewRouter builds the API engine. Resource handlers are not implemented yet;
// until they are, every /api route falls through to a 501 so callers get an
// honest "not implemented" rather than a bare 404.
func NewRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// TODO(platform): mount resource handlers under /api/v1 (auth, tenants,
	// workspaces, jobs, mlservices, models/images/datasets, resource-pools,
	// dashboard, audit) per docs/openapi/platform.yaml.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, server.Problem{
			Title:  "Not Implemented",
			Status: http.StatusNotImplemented,
			Detail: "platform-backend is not implemented yet; see docs/openapi/platform.yaml for the planned API.",
		})
	})
	return r
}

// NewProbeRouter serves the liveness and readiness probes. With no downstream
// dependencies wired yet, readiness simply mirrors liveness.
func NewProbeRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, server.HealthStatus{Status: "ok"}) }
	r.GET("/healthz", ok)
	r.GET("/readyz", ok)
	return r
}
