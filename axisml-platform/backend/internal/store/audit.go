package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditEvent is one durable record of a Platform mutation. It backs the
// dashboard recent-activity feed and is scoped by tenant identifier.
type AuditEvent struct {
	ID        string    `gorm:"column:id;primaryKey"`
	Tenant    string    `gorm:"column:tenant"`
	Kind      string    `gorm:"column:kind"`
	Name      string    `gorm:"column:name"`
	Action    string    `gorm:"column:action"`
	Actor     string    `gorm:"column:actor"`
	Phase     string    `gorm:"column:phase"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName maps AuditEvent to the audit_events table.
func (AuditEvent) TableName() string { return "audit_events" }

// AuditRepo is the data access for audit events.
type AuditRepo struct{ db *gorm.DB }

// NewAuditRepo constructs an AuditRepo.
func NewAuditRepo(db *gorm.DB) *AuditRepo { return &AuditRepo{db: db} }

// Insert appends one audit event, assigning an id/timestamp when empty.
func (r *AuditRepo) Insert(ctx context.Context, e *AuditEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(e).Error
}

// ListByTenant returns a tenant's events, newest first, bounded by limit.
func (r *AuditRepo) ListByTenant(ctx context.Context, tenant string, limit int) ([]AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []AuditEvent
	err := r.db.WithContext(ctx).
		Where("tenant = ?", tenant).
		Order("created_at DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}
