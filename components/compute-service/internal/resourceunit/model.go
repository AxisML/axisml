package resourceunit

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ResourceUnit is a reusable spec template within a single resource pool.
type ResourceUnit struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	PoolID       uuid.UUID      `gorm:"type:uuid;not null;column:pool_id"`
	Name         string         `gorm:"size:64;not null"`
	Description  string         `gorm:"type:text;not null;default:''"`
	Requests     datatypes.JSON `gorm:"type:jsonb;not null"`
	Limits       datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	NodeSelector datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (ResourceUnit) TableName() string { return "resource_units" }
