package tenantresolver

import (
	"time"

	"github.com/google/uuid"
)

// Tenant is a read-only projection of the columns artifacts cares about
// from compute's `tenants` table. The package owns the only place where
// artifacts depends on compute's PG schema; integration tests assert the
// shape stays minimal so a future cross-org split (design §7) can swap
// this for an HTTP client to compute without churning callers.
type Tenant struct {
	ID        uuid.UUID  `gorm:"column:id;primaryKey"`
	Name      string     `gorm:"column:name"`
	Namespace string     `gorm:"column:namespace"`
	Status    string     `gorm:"column:status"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

// TableName binds the model to compute's existing `tenants` table.
func (Tenant) TableName() string { return "tenants" }
