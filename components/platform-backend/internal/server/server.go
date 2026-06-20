package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Module wires its routes into the /api/v1 group. Modules apply their own auth
// middleware (they hold the Authenticator).
type Module interface {
	Register(rg *gin.RouterGroup)
}

// Options configures the API server.
type Options struct {
	Addr        string
	Log         *slog.Logger
	Modules     []Module
	JWKSHandler gin.HandlerFunc // served at /.well-known/jwks.json
	Ready       func(context.Context) error
}

// Server wraps the gin engine.
type Server struct {
	addr   string
	engine *gin.Engine
	log    *slog.Logger
}

// New builds the API engine with the standard middleware chain.
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
	if opts.JWKSHandler != nil {
		r.GET("/.well-known/jwks.json", opts.JWKSHandler)
	}

	api := r.Group("/api/v1")
	for _, m := range opts.Modules {
		m.Register(api)
	}

	// Unregistered routes: a documented-but-unwired /api/v1 endpoint is "not
	// implemented yet" (501); everything else is 404 — both as problem+json so
	// clients always get a parseable RFC 7807 body, never a bare empty 404.
	r.NoRoute(func(c *gin.Context) {
		c.Header("Content-Type", "application/problem+json")
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
			c.JSON(http.StatusNotImplemented, Problem{
				Type:     URI(problemTypeBase + "not-implemented"),
				Title:    "Not Implemented",
				Status:   http.StatusNotImplemented,
				Detail:   "This endpoint is declared in the API contract but not yet implemented.",
				Instance: c.Request.URL.Path,
				Code:     "not-implemented",
			})
			return
		}
		c.JSON(http.StatusNotFound, Problem{
			Type:     URI(problemTypeBase + "not-found"),
			Title:    "Not Found",
			Status:   http.StatusNotFound,
			Instance: c.Request.URL.Path,
			Code:     "not-found",
		})
	})

	return &Server{addr: opts.Addr, engine: r, log: opts.Log}, nil
}

// Start runs the HTTP server until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{Addr: s.addr, Handler: s.engine, ReadHeaderTimeout: 10 * time.Second}
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

// Engine exposes the gin engine for in-process httptest.
func (s *Server) Engine() *gin.Engine { return s.engine }
