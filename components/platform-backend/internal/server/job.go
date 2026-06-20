package server

import "time"

// Backend selects the compute backend + engine for a run/service.
type Backend struct {
	Name   BackendName    `json:"name"`
	Engine string         `json:"engine"`
	Config map[string]any `json:"config,omitempty"`
}

// RoleTemplate is the pod template for one run role. ports/volumes/volumeMounts
// are pass-through from compute (shape matches the K8s PodSpec equivalents).
type RoleTemplate struct {
	Image        string           `json:"image,omitempty"`
	Command      []string         `json:"command,omitempty"`
	Args         []string         `json:"args,omitempty"`
	Env          []EnvVar         `json:"env,omitempty"`
	Ports        []map[string]any `json:"ports,omitempty"`
	Resources    ResourceMap      `json:"resources,omitempty"`
	Volumes      []map[string]any `json:"volumes,omitempty"`
	VolumeMounts []map[string]any `json:"volumeMounts,omitempty"`
}

// MLRunRole is one role of a run's topology.
type MLRunRole struct {
	Name          string        `json:"name"`
	Replicas      int           `json:"replicas,omitempty" binding:"min=1"`
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`
	Template      RoleTemplate  `json:"template"`
}

// MLRunRoleStatus is the live status of one run role.
type MLRunRoleStatus struct {
	Name              string       `json:"name,omitempty"`
	Replicas          int          `json:"replicas,omitempty"`
	ActiveReplicas    int          `json:"activeReplicas,omitempty"`
	ReadyReplicas     int          `json:"readyReplicas,omitempty"`
	SucceededReplicas int          `json:"succeededReplicas,omitempty"`
	FailedReplicas    int          `json:"failedReplicas,omitempty"`
	Template          RoleTemplate `json:"template,omitempty"`
	RestartPolicy     string       `json:"restartPolicy,omitempty"`
}

// RunPolicy carries run-level execution limits.
type RunPolicy struct {
	ActiveDeadlineSeconds   int `json:"activeDeadlineSeconds,omitempty" binding:"min=1"`
	TTLSecondsAfterFinished int `json:"ttlSecondsAfterFinished,omitempty" binding:"min=0"`
	BackoffLimit            int `json:"backoffLimit,omitempty" binding:"min=0"`
	ProgressDeadlineSeconds int `json:"progressDeadlineSeconds,omitempty" binding:"min=1"`
}

// Run is a single Run (the compute MLRun produced by triggering a Job).
type Run struct {
	ID                UUID              `json:"id,omitempty"`
	Namespace         string            `json:"namespace"`
	TenantName        string            `json:"tenantName"`
	TenantDisplayName string            `json:"tenantDisplayName,omitempty"`
	ComputeNamespace  string            `json:"computeNamespace,omitempty"`
	Name              string            `json:"name"`
	JobName           string            `json:"jobName,omitempty"`
	RunNumber         int               `json:"runNumber,omitempty"`
	DisplayName       string            `json:"displayName,omitempty"`
	Description       string            `json:"description,omitempty"`
	Owner             string            `json:"owner"`
	OwnerID           UUID              `json:"ownerId,omitempty"`
	Backend           Backend           `json:"backend"`
	PoolName          string            `json:"poolName,omitempty" binding:"dns1123,max=40"`
	UnitName          string            `json:"unitName,omitempty" binding:"dns1123,max=40"`
	Quota             string            `json:"quota,omitempty"`
	Resources         ResourceMap       `json:"resources,omitempty"`
	Roles             []MLRunRoleStatus `json:"roles,omitempty"`
	RunPolicy         RunPolicy         `json:"runPolicy,omitempty"`
	Spec              MLRunSpec         `json:"spec,omitempty"`
	Phase             RunPhase          `json:"phase,omitempty"`
	Message           string            `json:"message,omitempty"`
	StartedAt         *time.Time        `json:"startedAt,omitempty"`
	FinishedAt        *time.Time        `json:"finishedAt,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

// MLRunSpec is a pass-through mirror of compute's MLRun spec, surfaced on
// Run so the UI can render a full YAML view. The authoritative shape lives
// in compute.
type MLRunSpec struct {
	Backend    Backend        `json:"backend,omitempty"`
	Scheduling map[string]any `json:"scheduling,omitempty"`
	Roles      []MLRunRole    `json:"roles,omitempty"`
	RunPolicy  RunPolicy      `json:"runPolicy,omitempty"`
}

// RunList is a page of Run.
type RunList struct {
	Items         []Run  `json:"items"`
	Count         int    `json:"count" binding:"min=0"`
	ContinueToken string `json:"continueToken,omitempty"`
	Partial       bool   `json:"partial,omitempty"`
}

// ArtifactRef references an artifact version consumed by a run.
type ArtifactRef struct {
	Kind    ArtifactKind `json:"kind"`
	Name    string       `json:"name"`
	Version string       `json:"version"`
}

// JobSpec is a reusable Job template (snapshotted into an MLRun per run).
type JobSpec struct {
	Backend   Backend       `json:"backend"`
	PoolName  string        `json:"poolName,omitempty" binding:"dns1123,max=40"`
	UnitName  string        `json:"unitName,omitempty" binding:"dns1123,max=40"`
	Quota     string        `json:"quota,omitempty"`
	Roles     []MLRunRole   `json:"roles" binding:"min=1"`
	RunPolicy RunPolicy     `json:"runPolicy,omitempty"`
	Artifacts []ArtifactRef `json:"artifacts,omitempty"`
}

// Job is a Platform-owned reusable Job template.
type Job struct {
	ID          UUID      `json:"id"`
	Namespace   string    `json:"namespace"`
	TenantName  string    `json:"tenantName"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName,omitempty"`
	Description string    `json:"description,omitempty"`
	Owner       string    `json:"owner"`
	OwnerID     UUID      `json:"ownerId,omitempty"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        JobSpec   `json:"spec"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// JobList is a page of Job.
type JobList struct {
	Items         []Job  `json:"items"`
	Count         int    `json:"count" binding:"min=0"`
	ContinueToken string `json:"continueToken,omitempty"`
	Partial       bool   `json:"partial,omitempty"`
}

// JobCreateRequest is the body of POST /jobs.
type JobCreateRequest struct {
	Name        string    `json:"name" binding:"required,dns1123,min=1,max=63"`
	DisplayName string    `json:"displayName,omitempty" binding:"max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        JobSpec   `json:"spec" binding:"required"`
}

// JobPatchRequest is the body of PATCH /jobs/{name}.
type JobPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        JobSpec   `json:"spec,omitempty"`
}

// RunTriggerRequest holds the narrow set of trigger-time overrides for a run.
type RunTriggerRequest struct {
	DisplayName string        `json:"displayName,omitempty" binding:"max=100"`
	Labels      StringMap     `json:"labels,omitempty"`
	Annotations StringMap     `json:"annotations,omitempty"`
	PoolName    string        `json:"poolName,omitempty" binding:"dns1123,max=40"`
	UnitName    string        `json:"unitName,omitempty" binding:"dns1123,max=40"`
	Quota       string        `json:"quota,omitempty"`
	Resources   ResourceMap   `json:"resources,omitempty"`
	Artifacts   []ArtifactRef `json:"artifacts,omitempty"`
	Roles       []struct {
		Name string   `json:"name"`
		Args []string `json:"args,omitempty"`
		Env  []EnvVar `json:"env,omitempty"`
	} `json:"roles,omitempty"`
}
