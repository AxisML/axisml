package service

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the service state machine.
type Status string

const (
	StatusCreating Status = "Creating"
	StatusPending  Status = "Pending"
	StatusReady    Status = "Ready"
	StatusDegraded Status = "Degraded"
	StatusFailed   Status = "Failed"
	StatusDeleting Status = "Deleting"
	StatusDeleted  Status = "Deleted"
)

// Service is the GORM-backed `services` row. Partition key is `namespace`
// (bare string). Quota lives only on the rendered MLService CR via
// spec.scheduling.quota — an opaque ElasticQuota CR name passed through
// from Platform.
type Service struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Namespace          string         `gorm:"size:253;not null;column:namespace"`
	PoolID             uuid.UUID      `gorm:"type:uuid;not null;column:pool_id"`
	ResourceUnitID     uuid.UUID      `gorm:"type:uuid;not null;column:resource_unit_id"`
	Name               string         `gorm:"size:64;not null"`
	DisplayName        string         `gorm:"type:text;not null;default:''"`
	Description        string         `gorm:"type:text;not null;default:''"`
	OwnerUser          string         `gorm:"type:text;not null;default:''"`
	Spec               datatypes.JSON `gorm:"type:jsonb;not null"`
	DesiredSpecHash    string         `gorm:"type:text;not null;default:'';column:desired_spec_hash"`
	AppliedSpecHash    string         `gorm:"type:text;not null;default:'';column:applied_spec_hash"`
	RequestedResources datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Replicas           int32          `gorm:"not null;default:1"`
	ReadyReplicas      int32          `gorm:"not null;default:0;column:ready_replicas"`
	Endpoint           string         `gorm:"type:text;not null;default:''"`
	Status             string         `gorm:"size:16;not null"`
	Message            string         `gorm:"type:text;not null;default:''"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

func (Service) TableName() string { return "services" }
