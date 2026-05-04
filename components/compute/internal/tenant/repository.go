package tenant

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var t Tenant
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetActiveByName(ctx context.Context, name string) (*Tenant, error) {
	var t Tenant
	if err := r.db.WithContext(ctx).
		Where("name = ? AND deleted_at IS NULL", name).
		First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) ListActive(ctx context.Context, limit, offset int) ([]Tenant, int64, error) {
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

func (r *Repository) Create(ctx context.Context, tx *gorm.DB, t *Tenant) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Create(t).Error
}

func (r *Repository) Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, fields map[string]any) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Tenant{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) MarkDeleting(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	now := time.Now().UTC()
	return tx.Model(&Tenant{}).Where("id = ?", id).Updates(map[string]any{
		"status":     string(StatusDeleting),
		"deleted_at": now,
	}).Error
}

// FindWorkSet returns rows matching reconciler predicates.
type WorkSet struct {
	Creating  []Tenant
	Deleting  []Tenant
	SpecDirty []Tenant
}

func (r *Repository) FindWorkSet(ctx context.Context) (WorkSet, error) {
	var ws WorkSet
	if err := r.db.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", string(StatusCreating)).
		Find(&ws.Creating).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("status = ?", string(StatusDeleting)).
		Find(&ws.Deleting).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("desired_spec_hash <> applied_spec_hash AND deleted_at IS NULL").
		Find(&ws.SpecDirty).Error; err != nil {
		return ws, err
	}
	return ws, nil
}
