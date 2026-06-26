package module

import (
	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/cluster-manager/internal/server"
)

// RequireUser returns the middleware that rejects requests without a valid
// X-Axisml-User header (401). Cluster Manager handlers read the header directly
// rather than from a context key, so a composition root that mounts the routes
// on its OWN gin engine (Lite's axisml-core) gates the /api/v1 group with this.
// The Kubernetes binary applies it in internal/app and ignores this re-export.
func RequireUser() gin.HandlerFunc { return server.RequireUser }
