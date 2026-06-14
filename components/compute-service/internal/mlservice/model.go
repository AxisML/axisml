package mlservice

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

// MLService is the GORM-backed `mlservices` row. Phase column = high-frequency
// CR status.phase mirror; the rest of the status sub-tree {message,
// readyReplicas, endpoint, conditions[]} lives in `status jsonb`. Pool /
// unit names live on labels (axisml.io/resource-pool / -unit) for
// provenance, not in dedicated columns.
type MLService struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Namespace          string         `gorm:"size:253;not null;column:namespace"`
	Kind               string         `gorm:"size:16;not null;default:'service'"`
	Name               string         `gorm:"size:64;not null"`
	DisplayName        string         `gorm:"type:text;not null;default:''"`
	Description        string         `gorm:"type:text;not null;default:''"`
	Owner              string         `gorm:"type:text;not null;default:''"`
	Labels             datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Annotations        datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Spec               datatypes.JSON `gorm:"type:jsonb;not null"`
	Generation         int64          `gorm:"not null;default:1"`
	ObservedGeneration int64          `gorm:"not null;default:0;column:observed_generation"`
	Phase              string         `gorm:"size:16;not null;default:'Creating'"`
	StatusJSON         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}';column:status"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

func (MLService) TableName() string { return "mlservices" }

// StatusFields mirrors the CR status sub-tree compute persists.
type StatusFields struct {
	Message       string         `json:"message,omitempty"`
	ReadyReplicas int32          `json:"readyReplicas"`
	Endpoint      string         `json:"endpoint,omitempty"`
	Conditions    []ConditionRow `json:"conditions,omitempty"`
}

// ConditionRow is one entry inside status.conditions[].
type ConditionRow struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}
