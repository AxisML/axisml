package quota

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the quota state machine (driven by Tenant Informer).
type Status string

const (
	StatusCreating Status = "Creating"
	StatusActive   Status = "Active"
	StatusDeleting Status = "Deleting"
	StatusDeleted  Status = "Deleted"
)

// Quota is the GORM-backed persistence row for `quotas`.
type Quota struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID      `gorm:"type:uuid;not null;column:tenant_id"`
	PoolID    uuid.UUID      `gorm:"type:uuid;not null;column:pool_id"`
	Name      string         `gorm:"size:64;not null"`
	Spec      datatypes.JSON `gorm:"type:jsonb;not null"`
	Status    string         `gorm:"size:16;not null"`
	Used      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (Quota) TableName() string { return "quotas" }
