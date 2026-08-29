// Package app wires the cluster-manager process: HTTP server, K8s client,
// signal handling. The main package's job is just flag parsing.
//
// cluster-manager is a stateless REST shell over the ResourcePool CRD
// (cluster-scoped, axisml.io/v1alpha1) — no PG, no reconciler, no leader
// election. Multi-replica peer.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/k8sclient"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/k8sstore"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/promql"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/resourcepool"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
	clustermodule "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/module"
)

// Config groups the runtime knobs the binary exposes.
type Config struct {
	APIBindAddress     string
	MetricsBindAddress string
	ProbesBindAddress  string
	// PrometheusURL backs the per-pool metrics endpoint; empty disables it.
	PrometheusURL string
}

// Run is the long-running entry point. Cancel ctx (e.g. via SIGTERM
// handler) to trigger a graceful HTTP shutdown.
func Run(ctx context.Context, cfg Config) error {
	c, err := k8sclient.Build()
	if err != nil {
		return fmt.Errorf("k8sclient: %w", err)
	}
	return runWith(ctx, cfg, c)
}

// RunWith is the test-friendly variant accepting a prebuilt client (envtest).
func RunWith(ctx context.Context, cfg Config, c client.Client) error {
	return runWith(ctx, cfg, c)
}

func runWith(ctx context.Context, cfg Config, c client.Client) error {
	logger := log.FromContext(ctx).WithName("cluster-manager")

	metrics, err := promql.New(cfg.PrometheusURL)
	if err != nil {
		return fmt.Errorf("prometheus: %w", err)
	}
	r := NewRouter(c, metrics)
	probes := NewProbeRouter()

	srvAPI := &http.Server{Addr: cfg.APIBindAddress, Handler: r}
	probeSrv := &http.Server{Addr: cfg.ProbesBindAddress, Handler: probes}

	errCh := make(chan error, 2)
	go func() {
		logger.Info("listening", "addr", cfg.APIBindAddress)
		if err := srvAPI.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("api: %w", err)
		}
	}()
	go func() {
		if err := probeSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("probes: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		_ = srvAPI.Close()
		_ = probeSrv.Close()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srvAPI.Shutdown(shutdownCtx)
	_ = probeSrv.Shutdown(shutdownCtx)
	return nil
}

// NewRouter builds the gin router. Exposed so integration tests can drive
// the engine via httptest without booting the listener. metrics may be nil
// (the per-pool metrics route then reports metrics-unavailable).
func NewRouter(c client.Client, metrics resourcepool.MetricsQuerier) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health probes are also served on the API port (in addition to the dedicated
	// probes server) so clients reaching the API service can health-check it,
	// matching compute-service / platform.
	health := func(ctx *gin.Context) { ctx.JSON(http.StatusOK, srv.HealthStatus{Status: "ok"}) }
	r.GET("/healthz", health)
	r.GET("/readyz", health)

	// Kubernetes composition root: back the form-neutral stores with the
	// cluster-scoped CRs, then assemble via the shared pkg/module.
	mod := clustermodule.New(clustermodule.Deps{
		Pools:   k8sstore.NewResourcePoolStore(c),
		Tenants: k8sstore.NewTenantStore(c),
		Volumes: k8sstore.NewVolumeStore(c),
		Metrics: metrics,
	})
	api := r.Group("/api/v1", srv.RequireUser)
	mod.RegisterRoutes(api)

	return r
}

// NewProbeRouter builds the lightweight /healthz, /readyz router.
func NewProbeRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	probes := gin.New()
	probes.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, srv.HealthStatus{Status: "ok"})
	})
	probes.GET("/readyz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, srv.HealthStatus{Status: "ok"})
	})
	return probes
}
