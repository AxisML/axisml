package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"

	"github.com/axisml/axisml/components/artifacts/internal/auth"
	"github.com/axisml/axisml/components/artifacts/internal/metrics"
)

// Module is anything that wires its routes into a /api/v1 sub-group.
type Module interface {
	Register(rg *gin.RouterGroup)
}

// Server wraps a gin engine and exposes Manager-friendly Start/NeedLeaderElection.
type Server struct {
	addr   string
	engine *gin.Engine
	log    logr.Logger
}

// Options configures a server. Logger and address must be supplied.
type Options struct {
	Addr    string
	Log     logr.Logger
	Modules []Module
	// Ready is consulted on /readyz; nil means "always ready". Wire dependency
	// pings (DB) here so kubelet only routes traffic once the replica is
	// actually serviceable.
	Ready func(context.Context) error
}

// New returns a Server with the standard middleware chain installed.
func New(opts Options) (*Server, error) {
	if err := RegisterValidators(); err != nil {
		return nil, err
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		RequestID(),
		AccessLog(opts.Log),
		Recovery(opts.Log),
		metrics.GinMiddleware(),
		auth.Middleware(),
		ErrorHandler(),
	)
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/readyz", func(c *gin.Context) {
		if opts.Ready != nil {
			cctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := opts.Ready(cctx); err != nil {
				c.String(http.StatusServiceUnavailable, "not ready: %v", err)
				return
			}
		}
		c.String(http.StatusOK, "ok")
	})

	api := r.Group("/api/v1")
	for _, m := range opts.Modules {
		m.Register(api)
	}

	return &Server{addr: opts.Addr, engine: r, log: opts.Log}, nil
}

// Start runs the HTTP server until ctx is cancelled. Implements
// sigs.k8s.io/controller-runtime/pkg/manager.Runnable.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server listening", "addr", s.addr)
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

// NeedLeaderElection returns false: the API is served from every replica.
func (s *Server) NeedLeaderElection() bool { return false }

// Engine exposes the underlying Gin engine for tests that drive the API
// in-process via httptest.NewRecorder + engine.ServeHTTP.
func (s *Server) Engine() *gin.Engine { return s.engine }
