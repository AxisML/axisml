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

// Visibility enumerates per-artifact visibility (design §3).
const (
	VisibilityTenant = "tenant" // default; visible only within the row's tenant scope
	VisibilityPublic = "public" // global; only allowed in the default tenant scope
)

// PublicVisibilityNamespace is the only logical tenant scope where
// visibility='public' is accepted; design §3 + database.md §3.1.
const PublicVisibilityNamespace = "default"

// Artifact is the GORM-backed `artifacts` row. Keyed on
// (namespace, kind, name, version); namespace is a bare string with no
// existence check, kind matches a registered ArtifactHandler.
type Artifact struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Namespace   string         `gorm:"size:253;not null"`
	Kind        string         `gorm:"size:32;not null"`
	Name        string         `gorm:"size:128;not null"`
	Version     string         `gorm:"size:128;not null"`
	Visibility  string         `gorm:"size:16;not null;default:'tenant'"`
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
