package auth

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Downstream identity / scope headers (backend.md §5.1–5.2).
const (
	HeaderUser   = "X-Axisml-User"   // injected outbound to System-layer services
	HeaderTenant = "X-Axisml-Tenant" // active tenant for name-addressed endpoints
)

const (
	ctxIdentityKey = "axisml.identity"
	ctxJTIKey      = "axisml.jti"
)

type userCtxKey struct{}

// setIdentity stores the resolved identity on the gin context.
func setIdentity(c *gin.Context, id *Identity) {
	c.Set(ctxIdentityKey, id)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), userCtxKey{}, id.Username))
}

// Current returns the authenticated identity, or nil if the request is
// unauthenticated (middleware not run / failed).
func Current(c *gin.Context) *Identity {
	v, ok := c.Get(ctxIdentityKey)
	if !ok {
		return nil
	}
	id, _ := v.(*Identity)
	return id
}

// UsernameFromContext extracts the username carried on a request context (for
// outbound client calls). Empty when absent.
func UsernameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userCtxKey{}).(string)
	return v
}

// WithUser carries a username on a context for outbound identity injection in
// non-HTTP code paths (e.g. bootstrap).
func WithUser(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, userCtxKey{}, username)
}

// ActiveTenant returns the X-Axisml-Tenant header value (empty when unset).
func ActiveTenant(c *gin.Context) string { return c.GetHeader(HeaderTenant) }

// JTI returns the verified token's jti (for logout / refresh revocation).
func JTI(c *gin.Context) string { return c.GetString(ctxJTIKey) }
