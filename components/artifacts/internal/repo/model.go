package repo

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Status enumerates the ArtifactRepo lifecycle (design §4.2).
const (
	StatusActive   = "Active"
	StatusDeleting = "Deleting"
	StatusDeleted  = "Deleted"
)

// Kind enumerates the supported artifact kinds. Only `model` is implemented
// in MVP; the others are reserved per design §5 and rejected by the API
// layer until their handlers ship.
const (
	KindModel      = "model"
	KindDataset    = "dataset"
	KindImage      = "image"
	KindEvalReport = "eval_report"
)

// Repo is the GORM-backed artifact_repos row.
type Repo struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantName       *string        `gorm:"size:64;column:tenant_name"`
	Kind             string         `gorm:"size:32;not null"`
	Name             string         `gorm:"size:128;not null"`
	DisplayName      string         `gorm:"type:text;not null;default:''"`
	Description      string         `gorm:"type:text;not null;default:''"`
	Labels           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	Annotations      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	OwnerUser        string         `gorm:"type:text;not null;default:''"`
	Status           string         `gorm:"size:16;not null"`
	LatestArtifactID *uuid.UUID     `gorm:"type:uuid;column:latest_artifact_id"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// TableName binds the model.
func (Repo) TableName() string { return "artifact_repos" }

// Scope returns the OCI / S3 scope segment for this repo (design §4.3):
// `tenants/<name>` for tenant-private, `system` for the public space.
func (r *Repo) Scope() string {
	if r.TenantName == nil {
		return "system"
	}
	return "tenants/" + *r.TenantName
}
