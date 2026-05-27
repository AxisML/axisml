package service

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

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Service, error) {
	var s Service
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) GetByNamespaceName(ctx context.Context, namespace, name string) (*Service, error) {
	var s Service
	if err := r.db.WithContext(ctx).
		Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListByNamespace(ctx context.Context, namespace, kind string, limit, offset int, labelClause string, labelArgs []any) ([]Service, int64, error) {
	var rows []Service
	var total int64
	q := r.db.WithContext(ctx).Model(&Service{}).Where("namespace = ? AND deleted_at IS NULL", namespace)
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
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

func (r *Repository) Create(ctx context.Context, s *Service) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&Service{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) MarkDeleting(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Service{}).Where("id = ?", id).Updates(map[string]any{
		"phase":      string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

type WorkSet struct {
	Creating  []Service
	Deleting  []Service
	SpecDirty []Service
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
