package tenant

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository encapsulates tenants-table CRUD.
type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var t Tenant
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*Tenant, error) {
	var t Tenant
	if err := r.db.WithContext(ctx).Where("name = ? AND deleted_at IS NULL", name).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// List returns all active tenants, paginated by created_at DESC.
func (r *Repository) List(ctx context.Context, limit, offset int) ([]Tenant, int64, error) {
	var rows []Tenant
	var total int64
	q := r.db.WithContext(ctx).Model(&Tenant{}).Where("deleted_at IS NULL")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) Create(ctx context.Context, t *Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// Update applies the supplied fields to an active row.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&Tenant{}).
		Where("id = ? AND deleted_at IS NULL", id).Updates(fields).Error
}

// SoftDelete marks the tenant as Deleting and stamps deleted_at = now().
func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&Tenant{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"phase":      PhaseDeleting,
			"deleted_at": &now,
			"generation": gorm.Expr("generation + 1"),
		}).Error
}
