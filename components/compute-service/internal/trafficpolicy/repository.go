package trafficpolicy

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

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*TrafficPolicy, error) {
	var p TrafficPolicy
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) GetByNamespaceName(ctx context.Context, namespace, name string) (*TrafficPolicy, error) {
	var p TrafficPolicy
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
		First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) ListByNamespace(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]TrafficPolicy, int64, error) {
	var rows []TrafficPolicy
	var total int64
	q := r.db.WithContext(ctx).Model(&TrafficPolicy{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
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

func (r *Repository) Create(ctx context.Context, p *TrafficPolicy) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&TrafficPolicy{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) MarkDeleting(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&TrafficPolicy{}).Where("id = ?", id).Updates(map[string]any{
		"phase":      string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

// FindActiveReferencing returns active (non-deleted) policies in the namespace
// whose spec.backends[] references the given service name. Used to enforce the
// "a service is referenced by at most one active policy" occupancy rule
// (compute-service.md §4.5) — there is no DB constraint for it because the
// membership lives inside a jsonb array.
func (r *Repository) FindActiveReferencing(ctx context.Context, namespace, serviceName string) ([]TrafficPolicy, error) {
	var rows []TrafficPolicy
	err := r.db.WithContext(ctx).
		Where("namespace = ? AND deleted_at IS NULL AND EXISTS ("+
			"SELECT 1 FROM jsonb_array_elements(spec->'backends') AS b "+
			"WHERE b->>'serviceName' = ?)", namespace, serviceName).
		Find(&rows).Error
	return rows, err
}

type WorkSet struct {
	Creating  []TrafficPolicy
	Deleting  []TrafficPolicy
	SpecDirty []TrafficPolicy
}

const workSetBatch = 100

func (r *Repository) FindWorkSet(ctx context.Context) (WorkSet, error) {
	var ws WorkSet
	if err := r.db.WithContext(ctx).
		Where("phase = ? AND deleted_at IS NULL", string(StatusCreating)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Creating).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("phase = ?", string(StatusDeleting)).
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.Deleting).Error; err != nil {
		return ws, err
	}
	if err := r.db.WithContext(ctx).
		Where("generation <> observed_generation AND deleted_at IS NULL").
		Order("updated_at ASC").Limit(workSetBatch).
		Find(&ws.SpecDirty).Error; err != nil {
		return ws, err
	}
	return ws, nil
}
