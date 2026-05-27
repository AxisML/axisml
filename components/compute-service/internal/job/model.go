package job

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the job state machine.
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

// IsTerminal returns whether the status is a terminal state.
func IsTerminal(s Status) bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCancelled, StatusDeleted:
		return true
	}
	return false
}

// Job is the GORM-backed `jobs` row. The row is keyed on (namespace, name);
// namespace is a bare string partition key with no compute-side existence
// check. PoolName / UnitName are provenance only — the underlying
// ResourcePool lives in the K8s CRD owned by cluster-manager (no FK).
type Job struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Namespace          string         `gorm:"size:253;not null;column:namespace"`
	PoolName           string         `gorm:"size:40;not null;column:pool_name"`
	UnitName           string         `gorm:"size:40;not null;column:unit_name"`
	Name               string         `gorm:"size:64;not null"`
	DisplayName        string         `gorm:"type:text;not null;default:''"`
	Description        string         `gorm:"type:text;not null;default:''"`
	OwnerUser          string         `gorm:"type:text;not null;default:''"`
	Spec               datatypes.JSON `gorm:"type:jsonb;not null"`
	RequestedResources datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Status             string         `gorm:"size:16;not null"`
	Message            string         `gorm:"type:text;not null;default:''"`
	StartedAt          *time.Time
	FinishedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

func (Job) TableName() string { return "jobs" }
