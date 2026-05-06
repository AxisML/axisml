package artifact

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the Artifact lifecycle (design §3.5).
const (
	StatusUploading = "Uploading"
	StatusReady     = "Ready"
	StatusFailed    = "Failed"
	StatusDeleting  = "Deleting"
	StatusDeleted   = "Deleted"
)

// Artifact is the GORM-backed `artifacts` row.
type Artifact struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	RepoID      uuid.UUID      `gorm:"type:uuid;not null;column:repo_id"`
	Version     string         `gorm:"size:128;not null"`
	DisplayName string         `gorm:"type:text;not null;default:''"`
	Description string         `gorm:"type:text;not null;default:''"`
	Labels      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Annotations datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	OwnerUser   string         `gorm:"type:text;not null;default:''"`
	Spec        datatypes.JSON `gorm:"type:jsonb;not null"`
	Status      string         `gorm:"size:16;not null"`
	Message     string         `gorm:"type:text;not null;default:''"`
	Digest      string         `gorm:"type:text;not null;default:''"`
	ReadyAt     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// TableName binds the model.
func (Artifact) TableName() string { return "artifacts" }
