package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Authenticator wires the JWT signer, identity store and session blacklist into
// the gin middleware chain.
type Authenticator struct {
	Signer     *Signer
	Identities IdentityStore
	Sessions   SessionStore
}

// NewAuthenticator constructs an Authenticator.
func NewAuthenticator(signer *Signer, identities IdentityStore, sessions SessionStore) *Authenticator {
	return &Authenticator{Signer: signer, Identities: identities, Sessions: sessions}
}

func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func fail(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

// RequireAuthenticated verifies the bearer JWT, checks the session blacklist and
// loads the caller identity onto the context.
func (a *Authenticator) RequireAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := bearer(c)
		if tok == "" {
			fail(c, apperrors.New(apperrors.ClassUnauthorized, "missing bearer token").WithReason("unauthenticated"))
			return
		}
		claims, err := a.Signer.Verify(tok)
		if err != nil {
			fail(c, apperrors.Wrap(apperrors.ClassUnauthorized, "invalid bearer token", err).WithReason("unauthenticated"))
			return
		}
		active, err := a.Sessions.IsActive(c.Request.Context(), claims.ID)
		if err != nil {
			fail(c, apperrors.Wrap(apperrors.ClassInternal, "session lookup failed", err))
			return
		}
		if !active {
			fail(c, apperrors.New(apperrors.ClassUnauthorized, "session revoked or expired").WithReason("unauthenticated"))
			return
		}
		id, err := a.Identities.LoadIdentity(c.Request.Context(), claims.Subject)
		if err != nil {
			fail(c, apperrors.Wrap(apperrors.ClassUnauthorized, "identity not found", err).WithReason("unauthenticated"))
			return
		}
		// Enforce a forced password reset server-side: until the user changes
		// their password, every authenticated endpoint is blocked except the few
		// needed to read the session and perform the change itself. Without this,
		// MustChangePassword is merely advisory (the SPA honours it but a direct
		// API client — e.g. the bootstrap admin with the default password — could
		// exercise the whole API).
		if id.MustChangePassword && !passwordChangeExempt(c.FullPath()) {
			fail(c, apperrors.New(apperrors.ClassForbidden,
				"password change required before continuing").WithReason("password-change-required"))
			return
		}
		c.Set(ctxJTIKey, claims.ID)
		setIdentity(c, id)
		c.Next()
	}
}

// passwordChangeExempt reports whether a route remains reachable while the
// caller still owes a password change: reading the session, logging out,
// refreshing the token, and the change-password endpoint itself. Matched on the
// gin route pattern (FullPath) so the :id param is handled robustly.
func passwordChangeExempt(fullPath string) bool {
	switch fullPath {
	case "/api/v1/auth/me",
		"/api/v1/auth/logout",
		"/api/v1/auth/refresh",
		"/api/v1/users/:id/password":
		return true
	default:
		return false
	}
}

// RequireSystemAdmin admits only global system-admins. Must run after
// RequireAuthenticated.
func (a *Authenticator) RequireSystemAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := Current(c)
		if id == nil {
			fail(c, apperrors.New(apperrors.ClassUnauthorized, "unauthenticated").WithReason("unauthenticated"))
			return
		}
		if !id.IsSystemAdmin {
			fail(c, apperrors.New(apperrors.ClassForbidden, "system-admin required").WithReason("forbidden"))
			return
		}
		c.Next()
	}
}

// RequireTenantRole admits callers holding at least min in the tenant named by
// the path parameter tenantParam. system-admin short-circuits.
func (a *Authenticator) RequireTenantRole(min Role, tenantParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := Current(c)
		if id == nil {
			fail(c, apperrors.New(apperrors.ClassUnauthorized, "unauthenticated").WithReason("unauthenticated"))
			return
		}
		if err := checkTenantRole(id, c.Param(tenantParam), min); err != nil {
			fail(c, err)
			return
		}
		c.Next()
	}
}

// checkTenantRole distinguishes "no binding" (404, don't leak existence) from
// "insufficient role" (403): a member who can see the tenant but lacks the
// privilege gets forbidden, not not-found.
func checkTenantRole(id *Identity, tenant string, min Role) error {
	if id.IsSystemAdmin {
		return nil
	}
	role, ok := id.Bindings[tenant]
	if !ok {
		return apperrors.New(apperrors.ClassNotFound, "tenant not found").WithReason("not-found")
	}
	if !role.AtLeast(min) {
		return apperrors.New(apperrors.ClassForbidden, "insufficient role").WithReason("forbidden")
	}
	return nil
}

// RequireActiveTenantRole is RequireTenantRole for name-addressed endpoints that
// carry the tenant in the X-Axisml-Tenant header rather than the path.
func (a *Authenticator) RequireActiveTenantRole(min Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := Current(c)
		if id == nil {
			fail(c, apperrors.New(apperrors.ClassUnauthorized, "unauthenticated").WithReason("unauthenticated"))
			return
		}
		tenant := ActiveTenant(c)
		if tenant == "" {
			fail(c, apperrors.New(apperrors.ClassValidation, "active tenant required").WithReason("active-tenant-required"))
			return
		}
		if err := checkTenantRole(id, tenant, min); err != nil {
			fail(c, err)
			return
		}
		c.Next()
	}
}
