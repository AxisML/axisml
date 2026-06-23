package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// SessionRepo is the JWT session blacklist (satisfies auth.SessionStore).
type SessionRepo struct{ db *gorm.DB }

// NewSessionRepo constructs a SessionRepo.
func NewSessionRepo(db *gorm.DB) *SessionRepo { return &SessionRepo{db: db} }

// Create records an active session for a freshly issued token.
func (r *SessionRepo) Create(ctx context.Context, jti, userID string, expiresAtUnix int64) error {
	return r.db.WithContext(ctx).Create(&Session{
		JTI:       jti,
		UserID:    userID,
		ExpiresAt: time.Unix(expiresAtUnix, 0).UTC(),
		Revoked:   false,
	}).Error
}

// IsActive reports whether the jti is a known, unrevoked, unexpired session.
func (r *SessionRepo) IsActive(ctx context.Context, jti string) (bool, error) {
	var s Session
	err := r.db.WithContext(ctx).First(&s, "jti = ?", jti).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if s.Revoked || time.Now().UTC().After(s.ExpiresAt) {
		return false, nil
	}
	return true, nil
}

// Revoke marks a session revoked (forced logout).
func (r *SessionRepo) Revoke(ctx context.Context, jti string) error {
	return r.db.WithContext(ctx).Model(&Session{}).Where("jti = ?", jti).
		Update("revoked", true).Error
}

// DeleteExpired purges sessions whose tokens have already expired, returning the
// number of rows removed. Run periodically: IsActive ignores expired rows, but
// nothing else GCs them, so the table would grow unbounded.
func (r *SessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", time.Now().UTC()).Delete(&Session{})
	return res.RowsAffected, res.Error
}
