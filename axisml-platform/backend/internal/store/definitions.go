package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Definition table names.
const (
	TableJobs        = "jobs"
	TableExperiments = "experiments"
	TableModels      = "models"
	TableImages      = "images"
)

// DefinitionRepo is the generic data access for the four name-level definition
// tables (jobs / experiments / models / images). The table is selected at
// construction; rows are soft-deleted (same-name reuse after delete).
type DefinitionRepo struct {
	db    *gorm.DB
	table string
}

// NewDefinitionRepo binds a repo to one definition table.
func NewDefinitionRepo(db *gorm.DB, table string) *DefinitionRepo {
	return &DefinitionRepo{db: db, table: table}
}

func (r *DefinitionRepo) active(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table(r.table).Where("deleted_at IS NULL")
}

// Create inserts a definition.
func (r *DefinitionRepo) Create(ctx context.Context, d *Definition) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.Labels == nil {
		d.Labels = StrMap{}
	}
	if d.Annotations == nil {
		d.Annotations = StrMap{}
	}
	if len(d.Spec) == 0 {
		d.Spec = JSONB("{}")
	}
	now := time.Now().UTC()
	d.CreatedAt, d.UpdatedAt = now, now
	return r.db.WithContext(ctx).Table(r.table).Create(d).Error
}

// GetByName returns an active definition by (tenant, name).
func (r *DefinitionRepo) GetByName(ctx context.Context, tenant, name string) (*Definition, error) {
	var d Definition
	err := r.active(ctx).Where("tenant_name = ? AND name = ?", tenant, name).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &d, err
}

// List returns active definitions for a tenant scope. owner != "" filters by
// owner_user; q is a name/displayName substring.
func (r *DefinitionRepo) List(ctx context.Context, tenants []string, owner, q string, limit, offset int) ([]Definition, error) {
	var out []Definition
	tx := r.active(ctx)
	if tenants != nil {
		if len(tenants) == 0 {
			return []Definition{}, nil
		}
		tx = tx.Where("tenant_name IN ?", tenants)
	}
	if owner != "" {
		tx = tx.Where("owner_user = ?", owner)
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name ILIKE ? OR display_name ILIKE ?", like, like)
	}
	err := tx.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, err
}

// Update saves mutable metadata + spec.
func (r *DefinitionRepo) Update(ctx context.Context, d *Definition) error {
	d.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Table(r.table).Where("id = ?", d.ID).
		Updates(map[string]any{
			"display_name": d.DisplayName,
			"description":  d.Description,
			"labels":       d.Labels,
			"annotations":  d.Annotations,
			"spec":         d.Spec,
			"updated_at":   d.UpdatedAt,
		}).Error
}

// SoftDelete marks a definition deleted (the name becomes reusable).
func (r *DefinitionRepo) SoftDelete(ctx context.Context, tenant, name string) error {
	now := time.Now().UTC()
	return r.active(ctx).Where("tenant_name = ? AND name = ?", tenant, name).
		Update("deleted_at", now).Error
}
