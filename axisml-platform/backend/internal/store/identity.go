package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
)

// IdentityProvider resolves a verified user id into an auth.Identity (satisfies
// auth.IdentityStore).
type IdentityProvider struct {
	users *UserRepo
	roles *RoleRepo
}

// NewIdentityProvider constructs an IdentityProvider.
func NewIdentityProvider(db *gorm.DB) *IdentityProvider {
	return &IdentityProvider{users: NewUserRepo(db), roles: NewRoleRepo(db)}
}

// LoadIdentity builds the caller identity (rejects disabled accounts).
func (p *IdentityProvider) LoadIdentity(ctx context.Context, userID string) (*auth.Identity, error) {
	u, err := p.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.Disabled {
		return nil, fmt.Errorf("user disabled")
	}
	roles, err := p.roles.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	bindings := make(map[string]auth.Role, len(roles))
	for _, r := range roles {
		bindings[r.TenantName] = auth.Role(r.Role)
	}
	return &auth.Identity{
		UserID:             u.ID,
		Username:           u.Username,
		IsSystemAdmin:      u.IsSystemAdmin,
		MustChangePassword: u.MustChangePassword,
		Bindings:           bindings,
	}, nil
}
