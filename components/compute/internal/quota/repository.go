package quota

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Quota, error) {
	var q Quota
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *Repository) GetByTenantPoolName(ctx context.Context, tenantID, poolID uuid.UUID, name string) (*Quota, error) {
	var q Quota
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND pool_id = ? AND name = ? AND deleted_at IS NULL", tenantID, poolID, name).
		First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

// FindByTenantName looks up a quota row by `(tenant_id, name)` only. The
// schema's unique key is `(tenant_id, pool_id, name)`, so two pools may
// declare the same quota name — the API path `/tenants/:tenant/quotas/:quota`
// elides pool, so we explicitly surface that ambiguity instead of silently
// returning the first row sorted by created_at.
func (r *Repository) FindByTenantName(ctx context.Context, tenantID uuid.UUID, name string) ([]Quota, error) {
	var rows []Quota
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ? AND deleted_at IS NULL", tenantID, name).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]Quota, error) {
	var quotas []Quota
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at ASC").
		Find(&quotas).Error; err != nil {
		return nil, err
	}
	return quotas, nil
}

func (r *Repository) ListByTenantPaginated(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Quota, int64, error) {
	var quotas []Quota
	var total int64
	q := r.db.WithContext(ctx).Model(&Quota{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at ASC").Limit(limit).Offset(offset).Find(&quotas).Error; err != nil {
		return nil, 0, err
	}
	return quotas, total, nil
}

func (r *Repository) Create(ctx context.Context, tx *gorm.DB, q *Quota) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Create(q).Error
}

func (r *Repository) Update(ctx context.Context, tx *gorm.DB, id uuid.UUID, fields map[string]any) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Quota{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(fields).Error
}

func (r *Repository) UpdateUsed(ctx context.Context, id uuid.UUID, used datatypes.JSON) error {
	return r.db.WithContext(ctx).Model(&Quota{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("used", used).Error
}

func (r *Repository) UpdateStatus(ctx context.Context, tx *gorm.DB, id uuid.UUID, status Status) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Quota{}).
		Where("id = ?", id).
		Update("status", string(status)).Error
}

func (r *Repository) SoftDelete(ctx context.Context, tx *gorm.DB, id uuid.UUID) error {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	return tx.Model(&Quota{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"status":     string(StatusDeleted),
			"deleted_at": time.Now().UTC(),
		}).Error
}

func (r *Repository) ListAllActiveByTenant(ctx context.Context, tx *gorm.DB, tenantID uuid.UUID) ([]Quota, error) {
	if tx == nil {
		tx = r.db.WithContext(ctx)
	}
	var quotas []Quota
	if err := tx.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at ASC").
		Find(&quotas).Error; err != nil {
		return nil, err
	}
	return quotas, nil
}
