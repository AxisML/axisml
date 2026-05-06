package auth

import (
	"context"

	"github.com/gin-gonic/gin"
)

// HeaderUser is the request header injected by Platform / Gateway.
const HeaderUser = "X-Axisml-User"

type ctxKey struct{}

// Middleware reads X-Axisml-User and stores it on the gin context + the
// request context. Artifacts is internal-only and does not authenticate;
// roles (X-Axisml-Roles) are not consumed in MVP per design §8.2.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.GetHeader(HeaderUser)
		if user == "" {
			user = "anonymous"
		}
		c.Set(string(HeaderUser), user)
		c.Request = c.Request.WithContext(WithUser(c.Request.Context(), user))
		c.Next()
	}
}

// WithUser returns a context carrying the user identity.
func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

// User extracts the identity from a context. Returns "anonymous" when absent.
func User(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	if v == "" {
		return "anonymous"
	}
	return v
}
