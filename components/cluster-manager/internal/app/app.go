// Package app wires the cluster-manager process: HTTP server, K8s
// client, signal handling. The main package's job is just flag parsing
// and calling Run.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/axisml/axisml/components/cluster-manager/internal/k8sclient"
	"github.com/axisml/axisml/components/cluster-manager/internal/tenant"
)

// Config groups the runtime knobs the binary exposes.
type Config struct {
	APIBindAddress     string
	MetricsBindAddress string
	ProbesBindAddress  string
	NamespaceDenylist  []string
}

// Run is the long-running entry point. Cancel ctx (e.g. via SIGTERM
// handler) to trigger a graceful HTTP shutdown.
func Run(ctx context.Context, cfg Config) error {
	logger := log.FromContext(ctx).WithName("cluster-manager")

	c, err := k8sclient.Build()
	if err != nil {
		return fmt.Errorf("k8sclient: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api/v1")
	(&tenant.Handler{Client: c, NamespaceDenylist: cfg.NamespaceDenylist}).Register(api)

	probes := gin.New()
	probes.GET("/healthz", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
	probes.GET("/readyz", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	srv := &http.Server{Addr: cfg.APIBindAddress, Handler: r}
	probeSrv := &http.Server{Addr: cfg.ProbesBindAddress, Handler: probes}

	errCh := make(chan error, 2)
	go func() {
		logger.Info("listening", "addr", cfg.APIBindAddress)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
		_ = srv.Close()
		_ = probeSrv.Close()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = probeSrv.Shutdown(shutdownCtx)
	return nil
}
