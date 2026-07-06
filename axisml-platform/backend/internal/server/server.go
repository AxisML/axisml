package server

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
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
	Addr    string
	Log     *slog.Logger
	Modules []Module
	// Middlewares are applied to the /api/v1 group ahead of module routes (e.g.
	// audit recording). Because a group middleware wraps the whole chain, its
	// post-c.Next() phase sees the resolved route params and the identity set by
	// each module's own auth middleware.
	Middlewares []gin.HandlerFunc
	JWKSHandler gin.HandlerFunc // served at /.well-known/jwks.json
	Ready       func(context.Context) error
	// StaticFS is the built SPA bundle (frontend). When set, non-API routes
	// serve static assets with an index.html fallback (client-side routing).
	// When nil, non-API routes 404 — the integration suite runs without it.
	StaticFS fs.FS
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
		ErrorHandler(opts.Log),
	)

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, HealthStatus{Status: "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if opts.Ready != nil {
			cctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := opts.Ready(cctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, HealthStatus{Status: "unavailable", Components: map[string]string{"ready": err.Error()}})
				return
			}
		}
		c.JSON(http.StatusOK, HealthStatus{Status: "ok"})
	})
	if opts.JWKSHandler != nil {
		r.GET("/.well-known/jwks.json", opts.JWKSHandler)
	}

	api := r.Group("/api/v1", opts.Middlewares...)
	for _, m := range opts.Modules {
		m.Register(api)
	}

	// Unregistered routes: a documented-but-unwired /api/v1 endpoint is "not
	// implemented yet" (501). Non-API routes serve the SPA bundle when present
	// (static asset or index.html fallback for client-side routing), else 404.
	// Both API errors are problem+json so clients always get a parseable RFC
	// 7807 body, never a bare empty response.
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/v1/") {
			c.Header("Content-Type", "application/problem+json")
			c.JSON(http.StatusNotImplemented, Problem{
				Type:     URI(problemTypeBase + "not-implemented"),
				Title:    "Not Implemented",
				Status:   http.StatusNotImplemented,
				Detail:   "This endpoint is declared in the API contract but not yet implemented.",
				Instance: p,
				Code:     "not-implemented",
			})
			return
		}
		if opts.StaticFS != nil {
			serveSPA(c, opts.StaticFS)
			return
		}
		c.Header("Content-Type", "application/problem+json")
		c.JSON(http.StatusNotFound, Problem{
			Type:     URI(problemTypeBase + "not-found"),
			Title:    "Not Found",
			Status:   http.StatusNotFound,
			Instance: p,
			Code:     "not-found",
		})
	})

	return &Server{addr: opts.Addr, engine: r, log: opts.Log}, nil
}

// serveSPA serves the SPA bundle: an existing file is returned as-is
// (fingerprinted /assets/* get an immutable long cache), and any other path —
// including directories, which must not yield a listing — falls back to
// index.html so the client-side router can take over.
func serveSPA(c *gin.Context, fsys fs.FS) {
	name := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
	if name != "" {
		if f, err := fsys.Open(name); err == nil {
			info, statErr := f.Stat()
			_ = f.Close()
			// Only serve regular files; a directory would otherwise render a
			// browsable listing of the bundle.
			if statErr == nil && !info.IsDir() {
				if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				http.FileServer(http.FS(fsys)).ServeHTTP(c.Writer, c.Request)
				return
			}
		}
	}
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", index)
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
