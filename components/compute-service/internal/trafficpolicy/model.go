package trafficpolicy

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the traffic-policy state machine
// (compute-service.md §4.5).
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

// TrafficPolicy is the GORM-backed `traffic_policies` row. Phase column =
// high-frequency CR status.phase mirror; the rest of the status sub-tree
// {message, endpoint, backends[], conditions[]} lives in `status jsonb`.
// Only spec.backends[*].weight (and role on canary promote) is mutable;
// mode / endpoint / backend tuple are frozen at create.
type TrafficPolicy struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Namespace          string         `gorm:"size:253;not null;column:namespace"`
	Name               string         `gorm:"size:64;not null"`
	Mode               string         `gorm:"size:16;not null"`
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

func (TrafficPolicy) TableName() string { return "traffic_policies" }

// StatusFields mirrors the CR status sub-tree compute persists.
type StatusFields struct {
	Message    string             `json:"message,omitempty"`
	Endpoint   string             `json:"endpoint,omitempty"`
	Backends   []BackendStatusRow `json:"backends,omitempty"`
	Conditions []ConditionRow     `json:"conditions,omitempty"`
}

// BackendStatusRow is one entry inside status.backends[].
type BackendStatusRow struct {
	ServiceName string `json:"serviceName"`
	Weight      int32  `json:"weight"`
	Ready       bool   `json:"ready"`
}

// ConditionRow is one entry inside status.conditions[].
type ConditionRow struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}
