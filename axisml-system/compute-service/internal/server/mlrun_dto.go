package server

import (
	"time"

	"github.com/google/uuid"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
)

// MLRunCreateRequest is the API request body. Caller selects pool/unit by NAME
// (the ResourcePool CRD lives in K8s; compute reads it via Informer cache).
// `Quota` is the ElasticQuota CR name (cluster-unique string) stamped onto
// Pod labels — compute treats it as opaque.
type MLRunCreateRequest struct {
	Name          string                       `json:"name" binding:"required,axisml_name" desc:"MLRun name, unique within the namespace."`
	DisplayName   string                       `json:"displayName" desc:"Human-readable run label."`
	Description   string                       `json:"description" desc:"Free-text run description."`
	Labels        map[string]string            `json:"labels,omitempty" desc:"User-defined labels stored on the row and stamped onto the CR."`
	Annotations   map[string]string            `json:"annotations,omitempty" desc:"User-defined annotations stored on the row and stamped onto the CR."`
	PoolName      string                       `json:"poolName" binding:"required" desc:"Resource pool name resolved against the ResourcePool CRD via the Informer cache."`
	UnitName      string                       `json:"unitName" binding:"required" desc:"Resource unit (shape) name within the selected pool."`
	Quota         string                       `json:"quota" binding:"required" desc:"ElasticQuota CR name (opaque) stamped onto Pod labels for axisml-scheduler admission."`
	PriorityClass string                       `json:"priorityClass,omitempty" desc:"Optional Kubernetes PriorityClass name for the run's pods."`
	Backend       *mlrunv1alpha1.BackendSpec   `json:"backend" desc:"Compute backend/engine that runs the workload; defaults to (native, job) when omitted."`
	Roles         []mlrunv1alpha1.RoleSpec     `json:"roles" binding:"required,min=1" desc:"Run topology roles (at least one)."`
	RunPolicy     *mlrunv1alpha1.RunPolicySpec `json:"runPolicy" desc:"Run-level execution limits (deadline, TTL, backoff)."`
}

// MLRun is the HTTP response payload. Mirrors the design yaml: nested
// spec / status sub-trees, phase at the top level, owner / labels /
// annotations carried separately.
type MLRun struct {
	ID          uuid.UUID               `json:"id" desc:"Stable run identifier (PG row UUID)."`
	Namespace   string                  `json:"namespace" desc:"Namespace (= tenant identifier) the run belongs to."`
	Name        string                  `json:"name" desc:"MLRun name, unique within the namespace."`
	DisplayName string                  `json:"displayName,omitempty" desc:"Human-readable run label."`
	Description string                  `json:"description,omitempty" desc:"Free-text run description."`
	Owner       string                  `json:"owner,omitempty" desc:"Username of the run owner."`
	Labels      map[string]string       `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations map[string]string       `json:"annotations,omitempty" desc:"User-defined annotations."`
	Phase       string                  `json:"phase" desc:"Current run lifecycle phase: Creating, Pending, Running, Succeeded, Failed, Canceling, Cancelled, Deleting, Deleted."`
	Spec        mlrunv1alpha1.MLRunSpec `json:"spec" desc:"Resolved MLRun spec sub-tree (backend, scheduling, roles, run policy)."`
	Status      MLRunStatus             `json:"status" desc:"Operator-reported status sub-tree."`
	CreatedAt   time.Time               `json:"createdAt" desc:"Time the run was created."`
	UpdatedAt   time.Time               `json:"updatedAt" desc:"Time the run was last updated."`
	DeletedAt   *time.Time              `json:"deletedAt,omitempty" desc:"Soft-deletion timestamp, set once the run is deleted."`
}

// MLRunPatchRequest is the body for PATCH /api/v1/namespaces/{ns}/mlruns/{mlrun}.
// Per design §4.3, only the four "PG-only display" fields are mutable
// after create — the rest of the spec is frozen.
type MLRunPatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty" desc:"Updated human-readable run label."`
	Description *string           `json:"description,omitempty" desc:"Updated free-text run description."`
	Labels      map[string]string `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations map[string]string `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// MLRunStatus mirrors the CR status sub-tree compute persists for MLRuns.
type MLRunStatus struct {
	Message    string     `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	StartedAt  *time.Time `json:"startedAt,omitempty" desc:"Time the run started executing."`
	FinishedAt *time.Time `json:"finishedAt,omitempty" desc:"Time the run reached a terminal phase."`
}

// MLRunPhase is the lightweight response for the phase probes — GET
// /api/v1/namespaces/{ns}/mlruns/{mlrun}/phase (single) and the batch GET
// /api/v1/namespaces/{ns}/mlruns/phases. It returns only the run's lifecycle
// phase and status detail, skipping the heavy spec sub-tree the full MLRun
// payload carries. `name` identifies the run in batch responses.
type MLRunPhase struct {
	Name       string     `json:"name" desc:"MLRun name, unique within the namespace."`
	Phase      string     `json:"phase" desc:"Current run lifecycle phase: Creating, Pending, Running, Succeeded, Failed, Canceling, Cancelled, Deleting, Deleted."`
	Message    string     `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	StartedAt  *time.Time `json:"startedAt,omitempty" desc:"Time the run started executing."`
	FinishedAt *time.Time `json:"finishedAt,omitempty" desc:"Time the run reached a terminal phase."`
}
