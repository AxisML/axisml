package mlrun

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the job state machine (design §4.3).
type Status string

const (
	StatusCreating  Status = "Creating"
	StatusPending   Status = "Pending"
	StatusRunning   Status = "Running"
	StatusSucceeded Status = "Succeeded"
	StatusFailed    Status = "Failed"
	StatusCanceling Status = "Canceling"
	StatusCancelled Status = "Cancelled"
	StatusDeleting  Status = "Deleting"
	StatusDeleted   Status = "Deleted"
)

func IsTerminal(s Status) bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCancelled, StatusDeleted:
		return true
	}
	return false
}

// MLRun is the GORM-backed `mlruns` row. The row is keyed on (namespace, name);
// namespace is a bare string partition key (= tenants.name). Pool / unit
// names live on labels (axisml.io/resource-pool / -unit) for provenance,
// not in dedicated columns. The CR status sub-tree {message, startedAt,
// finishedAt, conditions[]} is persisted in `status jsonb`; the top-level
// `phase` column carries the high-frequency filter value (design §3.2 /
// database.md §3.2).
type MLRun struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Namespace   string         `gorm:"size:253;not null;column:namespace"`
	Name        string         `gorm:"size:64;not null"`
	DisplayName string         `gorm:"type:text;not null;default:''"`
	Description string         `gorm:"type:text;not null;default:''"`
	Owner       string         `gorm:"type:text;not null;default:''"`
	Labels      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Annotations datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Spec        datatypes.JSON `gorm:"type:jsonb;not null"`
	Phase       string         `gorm:"size:16;not null;default:'Creating'"`
	StatusJSON  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}';column:status"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func (MLRun) TableName() string { return "mlruns" }

// StatusFields mirrors the CR status sub-tree compute persists.
type StatusFields struct {
	Message    string         `json:"message,omitempty"`
	StartedAt  *time.Time     `json:"startedAt,omitempty"`
	FinishedAt *time.Time     `json:"finishedAt,omitempty"`
	Conditions []ConditionRow `json:"conditions,omitempty"`
}

// ConditionRow is one entry inside status.conditions[].
type ConditionRow struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}
