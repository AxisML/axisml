package module

import (
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/auth"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
)

// The HTTP middleware a Compute Service handler needs is form-neutral but lives
// in internal/. These thin re-exports let a composition root that mounts the
// routes on its OWN gin engine (Lite's axisml-core) install the same chain the
// Kubernetes binary's server.New installs. The Kubernetes binary keeps using
// server.New directly and ignores these.

// RegisterValidators registers the custom gin binding validators the Compute
// handlers reference (axisml_name, axisml_resource_unit). Call once at startup
// before any handler binds a request body. Safe to call repeatedly.
func RegisterValidators() error { return server.RegisterValidators() }

// RequestID returns the request-id middleware (stamps/propagates X-Request-ID).
func RequestID() gin.HandlerFunc { return server.RequestID() }

// AccessLog returns the structured per-request access-log middleware.
func AccessLog(log logr.Logger) gin.HandlerFunc { return server.AccessLog(log) }

// Recovery returns the panic-recovery middleware that renders an RFC 7807 error.
func Recovery(log logr.Logger) gin.HandlerFunc { return server.Recovery(log) }

// ErrorHandler returns the middleware that converts handler c.Error(err) calls
// into RFC 7807 problem+json responses. It must wrap the route handlers (added
// before route registration). The Compute and Artifact Hub handlers share this
// rendering, so a single instance serves both on a shared engine.
func ErrorHandler() gin.HandlerFunc { return server.ErrorHandler() }

// IdentityMiddleware returns the middleware that reads X-Axisml-User and stamps
// the caller identity onto the request context (defaults to anonymous). The
// Compute handlers read it back via the package-private context key, so this
// exact middleware (not Artifact Hub's) must run for Compute routes.
func IdentityMiddleware() gin.HandlerFunc { return auth.Middleware() }
