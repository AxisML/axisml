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

func (r *Repository) GetByTenantName(ctx context.Context, tenantID uuid.UUID, name string) (*Service, error) {
	var s Service
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND deleted_at IS NULL", tenantID, name).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Service, int64, error) {
	var rows []Service
	var total int64
	q := r.db.WithContext(ctx).Model(&Service{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
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
		"status":     string(StatusDeleting),
		"deleted_at": time.Now().UTC(),
	}).Error
}

// WorkSet groups the rows that match reconciler predicates.
type WorkSet struct {
	Creating  []Service
	Deleting  []Service
	SpecDirty []Service
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
