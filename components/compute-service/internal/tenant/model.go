// Package tenant implements compute-service's authority over the Tenant
// domain. PG `tenants` is the source of truth; the Tenant CR is derived.
package tenant

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Tenant is the GORM-backed persistence row for the `tenants` table.
type Tenant struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Name               string         `gorm:"size:40;not null"`
	DisplayName        string         `gorm:"type:text;not null;default:''"`
	Description        string         `gorm:"type:text;not null;default:''"`
	Owner              string         `gorm:"type:text;not null;default:''"`
	Labels             datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Annotations        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Spec               datatypes.JSON `gorm:"type:jsonb;not null"`
	Generation         int64          `gorm:"not null;default:1"`
	ObservedGeneration int64          `gorm:"not null;default:0"`
	Phase              string         `gorm:"type:text;not null;default:'Creating'"`
	Status             datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	LastModifiedBy     string         `gorm:"type:text;not null;default:''"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

func (Tenant) TableName() string { return "tenants" }

// Phase values for tenants.phase. Mirrors design §4.1.
const (
	PhaseCreating = "Creating"
	PhaseActive   = "Active"
	PhaseFailed   = "Failed"
	PhaseDeleting = "Deleting"
	PhaseDeleted  = "Deleted"
)
