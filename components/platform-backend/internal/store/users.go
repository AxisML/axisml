package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// UserRepo is the data access for users.
type UserRepo struct{ db *gorm.DB }

// NewUserRepo constructs a UserRepo.
func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// Create inserts a user, assigning an id when empty.
func (r *UserRepo) Create(ctx context.Context, u *User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	u.CreatedAt, u.UpdatedAt = now, now
	return r.db.WithContext(ctx).Create(u).Error
}

// GetByID returns a user by id.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// GetByUsername returns a user by username.
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).First(&u, "username = ?", username).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// GetByEmail returns a user by email (first match).
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).First(&u, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &u, err
}

// List returns users matching q (username/displayName/email substring) ordered
// by username, plus whether more rows exist beyond limit.
func (r *UserRepo) List(ctx context.Context, q string, limit, offset int) ([]User, error) {
	var users []User
	tx := r.db.WithContext(ctx).Model(&User{})
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("username ILIKE ? OR display_name ILIKE ? OR email ILIKE ?", like, like, like)
	}
	err := tx.Order("username ASC").Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// Update saves mutable profile fields.
func (r *UserRepo) Update(ctx context.Context, u *User) error {
	u.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", u.ID).
		Updates(map[string]any{
			"display_name":         u.DisplayName,
			"email":                u.Email,
			"disabled":             u.Disabled,
			"updated_at":           u.UpdatedAt,
			"must_change_password": u.MustChangePassword,
		}).Error
}

// SetPassword updates the password hash and clears must_change_password.
func (r *UserRepo) SetPassword(ctx context.Context, id, hash string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Updates(map[string]any{"password_hash": hash, "must_change_password": false, "updated_at": time.Now().UTC()}).Error
}

// Delete removes a user (cascades user_roles / sessions via FK).
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}
