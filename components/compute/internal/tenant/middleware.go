package tenant

import (
	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Middleware resolves the URL :tenant segment to a UUID and stashes it on
// the gin context as "tenantID". Downstream handlers (quota, job, service)
// consume that key.
func Middleware(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("tenant")
		if name == "" {
			_ = c.Error(apperrors.New(apperrors.CodeValidation, "missing tenant in path"))
			c.Abort()
			return
		}
		id, err := svc.GetID(c.Request.Context(), name)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Set("tenantID", id)
		c.Set("tenantName", name)
		c.Next()
	}
}
