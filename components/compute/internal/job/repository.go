package job

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

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Job, error) {
	var j Job
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&j).Error; err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) GetByTenantName(ctx context.Context, tenantID uuid.UUID, name string) (*Job, error) {
	var j Job
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND deleted_at IS NULL", tenantID, name).
		First(&j).Error; err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Job, int64, error) {
	var rows []Job
	var total int64
	q := r.db.WithContext(ctx).Model(&Job{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) Create(ctx context.Context, j *Job) error {
	return r.db.WithContext(ctx).Create(j).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) MarkDeleting(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Job{}).Where("id = ?", id).Updates(map[string]any{
		"status":     string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

// WorkSet groups rows that match reconciler predicates.
type WorkSet struct {
	Creating  []Job
	Canceling []Job
	Deleting  []Job
}

// workSetBatch caps each predicate's payload per tick (see tenant repository
// for rationale).
const workSetBatch = 100

func (r *Repository) FindWorkSet(ctx context.Context) (WorkSet, error) {
	var ws WorkSet
	if err := r.db.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", string(StatusCreating)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Creating).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("status = ?", string(StatusCanceling)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Canceling).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("status = ?", string(StatusDeleting)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Deleting).Error; err != nil {
		return ws, err
	}
	return ws, nil
}
