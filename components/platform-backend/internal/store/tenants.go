package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantRepo is the data access for the durable tenant record.
type TenantRepo struct{ db *gorm.DB }

// NewTenantRepo constructs a TenantRepo.
func NewTenantRepo(db *gorm.DB) *TenantRepo { return &TenantRepo{db: db} }

// Create inserts a tenant row, assigning an id when empty.
func (r *TenantRepo) Create(ctx context.Context, t *Tenant) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.Labels == nil {
		t.Labels = StrMap{}
	}
	if t.Annotations == nil {
		t.Annotations = StrMap{}
	}
	now := time.Now().UTC()
	t.CreatedAt, t.UpdatedAt = now, now
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByIdentifier returns a tenant by its identifier.
func (r *TenantRepo) GetByIdentifier(ctx context.Context, identifier string) (*Tenant, error) {
	var t Tenant
	err := r.db.WithContext(ctx).First(&t, "identifier = ?", identifier).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &t, err
}

// List returns tenants ordered by creation time. When scope is non-nil, results
// are restricted to those identifiers (non-admin visibility); nil = all.
func (r *TenantRepo) List(ctx context.Context, q string, scope []string, limit, offset int) ([]Tenant, error) {
	var tenants []Tenant
	tx := r.db.WithContext(ctx).Model(&Tenant{})
	if scope != nil {
		if len(scope) == 0 {
			return []Tenant{}, nil
		}
		tx = tx.Where("identifier IN ?", scope)
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("identifier ILIKE ? OR display_name ILIKE ?", like, like)
	}
	err := tx.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tenants).Error
	return tenants, err
}

// UpdateMeta saves editable display metadata.
func (r *TenantRepo) UpdateMeta(ctx context.Context, t *Tenant) error {
	t.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&Tenant{}).Where("id = ?", t.ID).
		Updates(map[string]any{
			"display_name":     t.DisplayName,
			"description":      t.Description,
			"labels":           t.Labels,
			"annotations":      t.Annotations,
			"last_modified_by": t.LastModifiedBy,
			"updated_at":       t.UpdatedAt,
		}).Error
}

// SetSuspended sets or clears the suspended_at gate.
func (r *TenantRepo) SetSuspended(ctx context.Context, identifier string, at *time.Time) error {
	return r.db.WithContext(ctx).Model(&Tenant{}).Where("identifier = ?", identifier).
		Updates(map[string]any{"suspended_at": at, "updated_at": time.Now().UTC()}).Error
}

// Delete hard-deletes a tenant row.
func (r *TenantRepo) Delete(ctx context.Context, identifier string) error {
	return r.db.WithContext(ctx).Where("identifier = ?", identifier).Delete(&Tenant{}).Error
}
