package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// IsNotFound surfaces the GORM "no row" sentinel.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// GetByID loads a repo by primary key.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Repo, error) {
	var row Repo
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetByTenantKindName loads a tenant-scoped repo by its natural key.
// Soft-deleted rows are excluded.
func (r *Repository) GetByTenantKindName(ctx context.Context, tenant, kind, name string) (*Repo, error) {
	var row Repo
	if err := r.db.WithContext(ctx).
		Where("tenant_name = ? AND kind = ? AND name = ? AND deleted_at IS NULL", tenant, kind, name).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListByTenant returns active rows scoped to the given tenant.
func (r *Repository) ListByTenant(ctx context.Context, tenant string, kind string, limit, offset int) ([]Repo, int64, error) {
	var rows []Repo
	var total int64
	q := r.db.WithContext(ctx).Model(&Repo{}).
		Where("tenant_name = ? AND deleted_at IS NULL", tenant)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Create persists a new repo row.
func (r *Repository) Create(ctx context.Context, tx *gorm.DB, row *Repo) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Create(row).Error
}

// Update applies the given column map to a repo row.
func (r *Repository) Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, fields map[string]any) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Repo{}).Where("id = ?", id).Updates(fields).Error
}

// MarkDeleting transitions a repo to the Deleting state. The row stays in
// place (deleted_at is left null until cascade GC completes) so that the
// natural-key uniqueness keeps holding the slot.
func (r *Repository) MarkDeleting(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Repo{}).Where("id = ?", id).Updates(map[string]any{
		"status": StatusDeleting,
	}).Error
}

// MarkDeleted finalizes the row after cascade GC succeeds.
func (r *Repository) MarkDeleted(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	now := time.Now().UTC()
	return tx.Model(&Repo{}).Where("id = ?", id).Updates(map[string]any{
		"status":     StatusDeleted,
		"deleted_at": now,
	}).Error
}
