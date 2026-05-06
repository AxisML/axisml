package tenantresolver

import (
	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/artifacts/pkg/errors"
)

// Gin context keys populated by Middleware.
const (
	CtxKeyTenantID        = "tenantID"
	CtxKeyTenantName      = "tenantName"
	CtxKeyTenantNamespace = "tenantNamespace"
)

// Middleware resolves the :tenant URL segment to a tenant row and stashes
// id/name/namespace on the gin context for downstream handlers.
func Middleware(r *Resolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("tenant")
		if name == "" {
			_ = c.Error(apperrors.New(apperrors.CodeValidation, "missing tenant in path"))
			c.Abort()
			return
		}
		t, err := r.Resolve(c.Request.Context(), name)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Set(CtxKeyTenantID, t.ID)
		c.Set(CtxKeyTenantName, t.Name)
		c.Set(CtxKeyTenantNamespace, t.Namespace)
		c.Next()
	}
}
