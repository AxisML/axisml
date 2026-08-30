// Package store holds the GORM-backed persistence structs for compute-service.
// Each entity is a bare row type keyed on (namespace, name); namespace is a
// bare string partition key (= tenants.name). These types carry only gorm
// tags + TableName helpers — no API shape and no business logic — so the
// persistence layer stays free of the HTTP/domain vocabulary.
package store

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// MLRun is the GORM-backed `mlruns` row. Pool / unit names live on labels
// (resource.axisml.io/pool / -unit) for provenance, not in dedicated columns.
// The CR status sub-tree {message, startedAt, finishedAt} is
// persisted in `status jsonb`; the top-level `phase` column carries the
// high-frequency filter value (design §3.2 / database.md §3.2).
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
	Priority    int32          `gorm:"not null;default:0"`
	Phase       string         `gorm:"size:16;not null;default:'Queued'"`
	StatusJSON  datatypes.JSON `gorm:"type:jsonb;not null;default:'{}';column:status"`
	ScheduledAt *time.Time     `gorm:"column:scheduled_at"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

func (MLRun) TableName() string { return "mlruns" }

// MLService is the GORM-backed `mlservices` row. Desired replicas live in Spec;
// AdmittedReplicas is the durable capacity/quota reservation and
// DispatchedReplicas is the last replica vector accepted by the runtime. Phase
// mirrors CR status only after the pre-runtime Queued/Creating states. The rest
// of the status sub-tree lives in status jsonb. Pool/unit names live on labels
// (resource.axisml.io/pool / -unit) for provenance, not in dedicated columns.
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
	AdmittedReplicas   datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';column:admitted_replicas"`
	DispatchedReplicas datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';column:dispatched_replicas"`
	Generation         int64          `gorm:"not null;default:1"`
	ObservedGeneration int64          `gorm:"not null;default:0;column:observed_generation"`
	Phase              string         `gorm:"size:16;not null;default:'Queued'"`
	StatusJSON         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}';column:status"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

func (MLService) TableName() string { return "mlservices" }

// TrafficPolicy is the GORM-backed `traffic_policies` row. Phase column =
// high-frequency CR status.phase mirror; the rest of the status sub-tree
// {message, endpoint, backends[]} lives in `status jsonb`.
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
