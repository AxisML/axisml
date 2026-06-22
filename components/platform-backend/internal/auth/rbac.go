// Package auth implements Platform's control-plane authentication and RBAC:
// bcrypt login, RS256 JWT issue/verify + JWKS, the session blacklist, and the
// gin middleware chain (RequireAuthenticated / RequireSystemAdmin /
// RequireTenantRole / Require*Owner). Contract: backend.md §4 + auth.md.
package auth

import "context"

// Role is one of the three hard-coded RBAC roles (auth.md §3). They are ordered
// system-admin > tenant-admin > user for all "≥ role" comparisons.
type Role string

const (
	RoleSystemAdmin Role = "system-admin"
	RoleTenantAdmin Role = "tenant-admin"
	RoleUser        Role = "user"
)

func rank(r Role) int {
	switch r {
	case RoleSystemAdmin:
		return 3
	case RoleTenantAdmin:
		return 2
	case RoleUser:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether r is at least as privileged as min.
func (r Role) AtLeast(min Role) bool { return rank(r) >= rank(min) }

// Identity is the resolved caller context attached to each authenticated
// request. Bindings maps a tenant identifier to the caller's role in it
// (tenant-admin | user); system-admin is global and short-circuits all
// tenant/owner checks.
type Identity struct {
	UserID             string
	Username           string
	IsSystemAdmin      bool
	MustChangePassword bool
	Bindings           map[string]Role
}

// RoleIn returns the caller's effective role in a tenant. system-admin yields
// (system-admin, true) for any tenant.
func (id *Identity) RoleIn(tenant string) (Role, bool) {
	if id.IsSystemAdmin {
		return RoleSystemAdmin, true
	}
	r, ok := id.Bindings[tenant]
	return r, ok
}

// HasTenantRole reports whether the caller holds at least min in tenant.
func (id *Identity) HasTenantRole(tenant string, min Role) bool {
	if id.IsSystemAdmin {
		return true
	}
	r, ok := id.Bindings[tenant]
	return ok && r.AtLeast(min)
}

// VisibleTenants returns the tenant identifiers the caller is bound to (used to
// scope cross-tenant fan-out for non-admins).
func (id *Identity) VisibleTenants() []string {
	out := make([]string, 0, len(id.Bindings))
	for t := range id.Bindings {
		out = append(out, t)
	}
	return out
}

// Permissions projects the caller's capabilities as stable machine codes for
// GET /auth/me. The frontend localises off these; they are not authorization
// inputs (the middleware re-derives authz from role/binding each request).
func (id *Identity) Permissions() []string {
	if id.IsSystemAdmin {
		return []string{
			"users:write", "tenants:write", "quotas:write", "resourcepools:write",
			"members:write", "workloads:write", "artifacts:write", "tenants:read-all",
		}
	}
	perms := map[string]struct{}{}
	for _, r := range id.Bindings {
		perms["workloads:write"] = struct{}{}
		perms["artifacts:write"] = struct{}{}
		if r.AtLeast(RoleTenantAdmin) {
			perms["members:write"] = struct{}{}
		}
	}
	out := make([]string, 0, len(perms))
	for p := range perms {
		out = append(out, p)
	}
	return out
}

// IdentityStore loads the resolved Identity for a verified user id.
type IdentityStore interface {
	LoadIdentity(ctx context.Context, userID string) (*Identity, error)
}

// SessionStore is the JWT session blacklist keyed by jti (auth.md §2).
type SessionStore interface {
	Create(ctx context.Context, jti, userID string, expiresAtUnix int64) error
	IsActive(ctx context.Context, jti string) (bool, error)
	Revoke(ctx context.Context, jti string) error
}
