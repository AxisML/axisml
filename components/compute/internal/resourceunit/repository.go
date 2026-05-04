package resourceunit

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*ResourceUnit, error) {
	var u ResourceUnit
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetByPoolName(ctx context.Context, poolID uuid.UUID, name string) (*ResourceUnit, error) {
	var u ResourceUnit
	if err := r.db.WithContext(ctx).Where("pool_id = ? AND name = ? AND deleted_at IS NULL", poolID, name).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (r *Repository) ListByPool(ctx context.Context, poolID uuid.UUID, limit, offset int) ([]ResourceUnit, int64, error) {
	var units []ResourceUnit
	var total int64
	q := r.db.WithContext(ctx).Model(&ResourceUnit{}).Where("pool_id = ? AND deleted_at IS NULL", poolID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&units).Error; err != nil {
		return nil, 0, err
	}
	return units, total, nil
}

func (r *Repository) Create(ctx context.Context, u *ResourceUnit) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&ResourceUnit{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields).Error
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := timeNow()
	return r.db.WithContext(ctx).Model(&ResourceUnit{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now).Error
}
