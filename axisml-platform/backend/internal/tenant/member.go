package tenant

import (
	"context"
	"errors"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// ListMembers returns a tenant's members enriched with user profiles.
func (s *Service) ListMembers(ctx context.Context, tenant string) ([]server.Member, error) {
	if _, err := s.getRow(ctx, tenant); err != nil {
		return nil, err
	}
	roles, err := s.roles.ListByTenant(ctx, tenant)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "list members", err)
	}
	out := make([]server.Member, 0, len(roles))
	for i := range roles {
		u, err := s.users.GetByID(ctx, roles[i].UserID)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ClassInternal, "load member", err)
		}
		out = append(out, toMember(u, roles[i]))
	}
	return out, nil
}

// AddMember binds an existing user to a tenant role.
func (s *Service) AddMember(ctx context.Context, tenant, account, role string) (*server.Member, error) {
	if _, err := s.getRow(ctx, tenant); err != nil {
		return nil, err
	}
	if err := validateMemberRole(role); err != nil {
		return nil, err
	}
	u, err := s.resolveAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if _, err := s.roles.Get(ctx, u.ID, tenant); err == nil {
		return nil, apperrors.New(apperrors.ClassConflict, "user is already a member").WithReason("member-exists")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup member", err)
	}
	ur, err := s.roles.Set(ctx, u.ID, tenant, role)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "add member", err)
	}
	s.bust(ctx, u.ID)
	return ptr(toMember(u, *ur)), nil
}

// UpdateMember changes a member's role atomically, protecting the last
// tenant-admin (the count + write happen in one locked transaction).
func (s *Service) UpdateMember(ctx context.Context, tenant, userID, role string) (*server.Member, error) {
	if _, err := s.getRow(ctx, tenant); err != nil {
		return nil, err
	}
	if err := validateMemberRole(role); err != nil {
		return nil, err
	}
	ur, err := s.roles.GuardedSet(ctx, userID, tenant, role)
	if err != nil {
		return nil, mapMemberErr(err, "update member")
	}
	s.bust(ctx, userID)
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "load member", err)
	}
	return ptr(toMember(u, *ur)), nil
}

// RemoveMember unbinds a member atomically, protecting the last tenant-admin.
func (s *Service) RemoveMember(ctx context.Context, tenant, userID string) error {
	if _, err := s.getRow(ctx, tenant); err != nil {
		return err
	}
	if err := s.roles.GuardedRemove(ctx, userID, tenant); err != nil {
		return mapMemberErr(err, "remove member")
	}
	s.bust(ctx, userID)
	return nil
}

// mapMemberErr translates store sentinels into the contract problem codes.
func mapMemberErr(err error, op string) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return server.NotFound("member not found")
	case errors.Is(err, store.ErrLastTenantAdmin):
		return apperrors.New(apperrors.ClassConflict, "cannot remove the last tenant-admin").WithReason("last-tenant-admin")
	default:
		return apperrors.Wrap(apperrors.ClassInternal, op, err)
	}
}

func validateMemberRole(role string) error {
	if role != string(auth.RoleTenantAdmin) && role != string(auth.RoleUser) {
		return apperrors.New(apperrors.ClassValidation, "role must be tenant-admin or user").WithReason("invalid-role")
	}
	return nil
}

func toMember(u *store.User, ur store.UserRole) server.Member {
	return server.Member{
		UserID:      server.UUID(u.ID),
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Email:       server.Email(u.Email),
		RoleName:    server.RoleName(ur.Role),
		AddedAt:     ur.CreatedAt,
	}
}

func ptr[T any](v T) *T { return &v }
