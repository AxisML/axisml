package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"

	"github.com/axisml/axisml/components/artifact-hub/internal/auth"
	"github.com/axisml/axisml/components/artifact-hub/internal/metrics"
	"github.com/axisml/axisml/components/artifact-hub/pkg/httpx"
)

// Module is anything that wires its routes into a /api/v1 sub-group.
type Module interface {
	Register(rg *gin.RouterGroup)
}

// Server wraps a gin engine into a long-running HTTP listener.
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
	// Capabilities, when non-nil, is served unauthenticated at
	// GET /api/v1/capabilities (the deployment-form capability document).
	Capabilities any
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

	if opts.Capabilities != nil {
		r.GET("/api/v1/capabilities", func(c *gin.Context) { c.JSON(http.StatusOK, opts.Capabilities) })
	}

	api := r.Group("/api/v1")
	for _, m := range opts.Modules {
		m.Register(api)
	}

	return &Server{addr: opts.Addr, engine: r, log: opts.Log}, nil
}

// Start runs the HTTP API server until ctx is cancelled, then drains in-flight
// requests. The API serves on every replica (no leader election).
func (s *Server) Start(ctx context.Context) error {
	return httpx.Serve(ctx, s.addr, s.engine, s.log, "http api")
}

// ProbesHandler builds the kubelet liveness/readiness endpoints. They run on
// a dedicated port so probe traffic is independent of the API listener; ready
// is consulted on /readyz (nil means "always ready").
func ProbesHandler(ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			cctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ready(cctx); err != nil {
				http.Error(w, "not ready: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
		}
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

// Engine exposes the underlying Gin engine for tests that drive the API
// in-process via httptest.NewRecorder + engine.ServeHTTP.
func (s *Server) Engine() *gin.Engine { return s.engine }
