package module

import (
	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/artifact-hub/internal/auth"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
)

// The HTTP middleware an Artifact Hub handler needs lives in internal/. These
// thin re-exports let a composition root that mounts the routes on its OWN gin
// engine (Lite's axisml-core) install the same identity stamping and request
// validation the Kubernetes binary's server.New installs. The Kubernetes binary
// keeps using server.New directly and ignores these.

// RegisterValidators registers the custom gin binding validators the Artifact
// Hub handlers reference (axisml_name, axisml_version). Call once at startup
// before any handler binds a request body. Safe to call repeatedly.
func RegisterValidators() error { return server.RegisterValidators() }

// IdentityMiddleware returns the middleware that reads X-Axisml-User and stamps
// the caller identity onto the request context. Artifact Hub reads it back via
// its OWN package-private context key, so this exact middleware (distinct from
// Compute's) must also run on a shared engine for Artifact Hub routes.
func IdentityMiddleware() gin.HandlerFunc { return auth.Middleware() }
