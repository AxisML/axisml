// Package store holds the GORM persistence structs. Types here are bare
// (no Row/Model suffix) and named by entity; they are a pure leaf layer
// that imports no domain package.
package store

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

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
