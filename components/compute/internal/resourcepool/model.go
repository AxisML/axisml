package resourcepool

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ResourcePool is the GORM-backed persistence row for `resource_pools`.
type ResourcePool struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name         string         `gorm:"size:64;not null"`
	Description  string         `gorm:"type:text;not null;default:''"`
	NodeSelector datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Tolerations  datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	Metadata     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// TableName overrides the default plural to match the migration.
func (ResourcePool) TableName() string { return "resource_pools" }
