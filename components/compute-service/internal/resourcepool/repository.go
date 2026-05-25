package resourcepool

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository wraps the GORM table.
type Repository struct{ db *gorm.DB }

// NewRepository constructs a Repository with the supplied GORM handle.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Get fetches by ID.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*ResourcePool, error) {
	var p ResourcePool
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByName fetches by name (active rows only).
func (r *Repository) GetByName(ctx context.Context, name string) (*ResourcePool, error) {
	var p ResourcePool
	if err := r.db.WithContext(ctx).Where("name = ? AND deleted_at IS NULL", name).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// IsNotFound is a convenience for callers that want to translate to apperrors.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// List returns all active pools, paginated.
func (r *Repository) List(ctx context.Context, limit, offset int) ([]ResourcePool, int64, error) {
	var pools []ResourcePool
	var total int64
	q := r.db.WithContext(ctx).Model(&ResourcePool{}).Where("deleted_at IS NULL")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&pools).Error; err != nil {
		return nil, 0, err
	}
	return pools, total, nil
}

// Create inserts. Caller must allocate the UUID.
func (r *Repository) Create(ctx context.Context, p *ResourcePool) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// Update applies the supplied fields by ID (active rows only).
func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&ResourcePool{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields).Error
}

// SoftDelete marks deleted_at and is idempotent.
func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := timeNow()
	return r.db.WithContext(ctx).Model(&ResourcePool{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now).Error
}
