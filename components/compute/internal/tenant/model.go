package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the tenant state machine in PG. Active/Deleted are
// driven by the Informer; Creating/Suspended/Deleting by the API.
type Status string

const (
	StatusCreating  Status = "Creating"
	StatusActive    Status = "Active"
	StatusSuspended Status = "Suspended"
	StatusDeleting  Status = "Deleting"
	StatusDeleted   Status = "Deleted"
)

// Tenant is the GORM-backed `tenants` row.
type Tenant struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name            string         `gorm:"size:64;not null"`
	Namespace       string         `gorm:"size:253;not null"`
	DisplayName     string         `gorm:"type:text;not null;default:''"`
	Spec            datatypes.JSON `gorm:"type:jsonb;not null"`
	DesiredSpecHash string         `gorm:"type:text;not null;default:'';column:desired_spec_hash"`
	AppliedSpecHash string         `gorm:"type:text;not null;default:'';column:applied_spec_hash"`
	Status          string         `gorm:"size:16;not null"`
	Message         string         `gorm:"type:text;not null;default:''"`
	Annotations     datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

func (Tenant) TableName() string { return "tenants" }
