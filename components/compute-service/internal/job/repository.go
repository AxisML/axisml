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

// GetByNamespaceName looks up the live row for (namespace, name). Soft-
// deleted rows (deleted_at IS NOT NULL) are excluded so the name becomes
// reusable.
func (r *Repository) GetByNamespaceName(ctx context.Context, namespace, name string) (*Job, error) {
	var j Job
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
		First(&j).Error; err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) ListByNamespace(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]Job, int64, error) {
	var rows []Job
	var total int64
	q := r.db.WithContext(ctx).Model(&Job{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
	if labelClause != "" {
		q = q.Where(labelClause, labelArgs...)
	}
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
