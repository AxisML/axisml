// Package identity implements the Auth and Users tags: bcrypt login, JWT
// issue/refresh, session revocation, /auth/me, and system-admin user CRUD
// (auth.md §2, backend.md §3.1).
package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/store"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Service holds identity business logic.
type Service struct {
	users      *store.UserRepo
	roles      *store.RoleRepo
	sessions   auth.SessionStore
	ident      *store.IdentityProvider
	signer     *auth.Signer
	invalidate func(ctx context.Context, userID string)
}

// NewService constructs an identity Service. sessions is an auth.SessionStore so
// the caching decorator can write-through login/logout; *store.SessionRepo
// satisfies it directly.
func NewService(users *store.UserRepo, roles *store.RoleRepo, sessions auth.SessionStore, ident *store.IdentityProvider, signer *auth.Signer) *Service {
	return &Service{users: users, roles: roles, sessions: sessions, ident: ident, signer: signer}
}

// OnIdentityChange registers a hook invoked after any account mutation that
// affects a user's resolved identity, so a cached identity can be busted. No-op
// when unset.
func (s *Service) OnIdentityChange(f func(ctx context.Context, userID string)) *Service {
	s.invalidate = f
	return s
}

func (s *Service) bust(ctx context.Context, userID string) {
	if s.invalidate != nil {
		s.invalidate(ctx, userID)
	}
}

// LoginResult bundles a freshly issued token with the caller's profile.
type LoginResult struct {
	Token     string
	ExpiresAt time.Time
	User      *store.User
	Roles     []store.UserRole
}

// Login verifies credentials and issues a login JWT + session.
func (s *Service) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	u, err := s.users.GetByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.New(apperrors.ClassUnauthorized, "invalid credentials").WithReason("invalid-credentials")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup user", err)
	}
	if u.Disabled {
		return nil, apperrors.New(apperrors.ClassUnauthorized, "account disabled").WithReason("account-disabled")
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, apperrors.New(apperrors.ClassUnauthorized, "invalid credentials").WithReason("invalid-credentials")
	}
	return s.issue(ctx, u)
}

// Refresh issues a new token for the caller and revokes the presenting jti.
func (s *Service) Refresh(ctx context.Context, userID, oldJTI string) (*LoginResult, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.New(apperrors.ClassUnauthorized, "unauthenticated").WithReason("unauthenticated")
	}
	res, err := s.issue(ctx, u)
	if err != nil {
		return nil, err
	}
	_ = s.sessions.Revoke(ctx, oldJTI)
	return res, nil
}

func (s *Service) issue(ctx context.Context, u *store.User) (*LoginResult, error) {
	jti := uuid.NewString()
	token, exp, err := s.signer.Issue(u.ID, u.Username, jti, time.Now().UTC())
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "issue token", err)
	}
	if err := s.sessions.Create(ctx, jti, u.ID, exp.Unix()); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "create session", err)
	}
	roles, err := s.roles.ListByUser(ctx, u.ID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "load roles", err)
	}
	return &LoginResult{Token: token, ExpiresAt: exp, User: u, Roles: roles}, nil
}

// Logout revokes a session.
func (s *Service) Logout(ctx context.Context, jti string) error {
	return s.sessions.Revoke(ctx, jti)
}

// RolesOf returns a user's tenant bindings.
func (s *Service) RolesOf(ctx context.Context, userID string) ([]store.UserRole, error) {
	return s.roles.ListByUser(ctx, userID)
}

// ---- Users CRUD -------------------------------------------------------------

// CreateUser creates a Platform user.
func (s *Service) CreateUser(ctx context.Context, username, displayName, email, password string) (*store.User, error) {
	if _, err := s.users.GetByUsername(ctx, username); err == nil {
		return nil, apperrors.New(apperrors.ClassConflict, "username already exists").WithReason("user-exists")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup user", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "hash password", err)
	}
	u := &store.User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  displayName,
		Email:        email,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperrors.New(apperrors.ClassConflict, "username already exists").WithReason("user-exists")
		}
		return nil, apperrors.Wrap(apperrors.ClassInternal, "create user", err)
	}
	return u, nil
}

// GetUser returns a user by id.
func (s *Service) GetUser(ctx context.Context, id string) (*store.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.New(apperrors.ClassNotFound, "user not found").WithReason("not-found")
	}
	return u, err
}

// ListUsers searches users.
func (s *Service) ListUsers(ctx context.Context, q string, limit, offset int) ([]store.User, error) {
	return s.users.List(ctx, q, limit, offset)
}

// UpdateUser patches profile / disabled. A nil disabled leaves the account
// state untouched (so a bare profile edit cannot re-enable a disabled user).
func (s *Service) UpdateUser(ctx context.Context, id, displayName, email string, disabled *bool) (*store.User, error) {
	u, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if displayName != "" {
		u.DisplayName = displayName
	}
	if email != "" {
		u.Email = email
	}
	if disabled != nil {
		u.Disabled = *disabled
	}
	if err := s.users.Update(ctx, u); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "update user", err)
	}
	s.bust(ctx, u.ID) // disabled/profile change alters the resolved identity
	return u, nil
}

// DeleteUser removes a user.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if _, err := s.GetUser(ctx, id); err != nil {
		return err
	}
	// Revoke (and drop cached) sessions before the row is deleted: the FK cascade
	// removes the DB session rows, but the positive Redis cache would otherwise
	// read active until its TTL. Must run before Delete so the jtis are still
	// readable for cache invalidation.
	if _, err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "revoke sessions", err)
	}
	if err := s.users.Delete(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "delete user", err)
	}
	s.bust(ctx, id)
	return nil
}

// SetPassword sets a user's password. Self-service requires the current
// password; system-admin (isAdmin) may reset without it.
func (s *Service) SetPassword(ctx context.Context, id, current, next string, isAdmin bool) error {
	u, err := s.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if !isAdmin {
		if strings.TrimSpace(current) == "" || !auth.CheckPassword(u.PasswordHash, current) {
			return apperrors.New(apperrors.ClassValidation, "current password incorrect").WithReason("invalid-credentials")
		}
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "hash password", err)
	}
	if err := s.users.SetPassword(ctx, id, hash); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "set password", err)
	}
	s.bust(ctx, id) // clears must_change_password, part of the resolved identity
	// Invalidate all outstanding sessions: a changed/reset password must not
	// leave a previously issued (possibly leaked) token valid until its TTL.
	if _, err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "revoke sessions", err)
	}
	return nil
}
