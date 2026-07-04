package server

import "time"

// Backend selects the compute backend + engine for a run/service.
type Backend struct {
	Name   BackendName    `json:"name" desc:"Compute backend that runs the workload (native, kubeflow-trainer, kserve, custom)."`
	Engine string         `json:"engine" desc:"Engine within the backend (e.g. job, pytorchjob, deployment)."`
	Config map[string]any `json:"config,omitempty" desc:"Backend-specific free-form config (e.g. the target GVK for the custom backend)."`
}

// RoleTemplate is the pod template for one run role. ports/volumes/volumeMounts
// are pass-through from compute (shape matches the K8s PodSpec equivalents).
type RoleTemplate struct {
	Image        string           `json:"image,omitempty" desc:"Container image reference for this role's pods."`
	Command      []string         `json:"command,omitempty" desc:"Container entrypoint override."`
	Args         []string         `json:"args,omitempty" desc:"Container args override."`
	Env          []EnvVar         `json:"env,omitempty" desc:"Environment variables injected into the role's pods."`
	Ports        []map[string]any `json:"ports,omitempty" desc:"Container ports (pass-through to the K8s container ports shape)."`
	Resources    ResourceMap      `json:"resources,omitempty" desc:"Per-replica resource requests/limits."`
	Volumes      []map[string]any `json:"volumes,omitempty" desc:"Pod volumes (pass-through to the K8s PodSpec volumes shape)."`
	VolumeMounts []map[string]any `json:"volumeMounts,omitempty" desc:"Container volume mounts (pass-through to the K8s shape)."`
}

// MLRunRole is one role of a run's topology.
type MLRunRole struct {
	Name          string        `json:"name" desc:"Role name within the run topology (e.g. master, worker)."`
	Replicas      int           `json:"replicas,omitempty" binding:"min=1" desc:"Number of pods for this role."`
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty" desc:"Pod restart policy for the role."`
	Template      RoleTemplate  `json:"template" desc:"Pod template for the role."`
}

// MLRunRoleStatus is the live status of one run role.
type MLRunRoleStatus struct {
	Name              string       `json:"name,omitempty" desc:"Role name."`
	Replicas          int          `json:"replicas,omitempty" desc:"Desired replica count for the role."`
	ActiveReplicas    int          `json:"activeReplicas,omitempty" desc:"Pods currently running."`
	ReadyReplicas     int          `json:"readyReplicas,omitempty" desc:"Pods that have passed readiness."`
	SucceededReplicas int          `json:"succeededReplicas,omitempty" desc:"Pods that completed successfully."`
	FailedReplicas    int          `json:"failedReplicas,omitempty" desc:"Pods that terminated in failure."`
	Template          RoleTemplate `json:"template,omitempty" desc:"Resolved pod template for the role."`
	RestartPolicy     string       `json:"restartPolicy,omitempty" desc:"Effective restart policy for the role."`
}

// RunPolicy carries run-level execution limits. The bounded fields use
// omitempty so an omitted (zero) RunPolicy doesn't trip the min validators.
type RunPolicy struct {
	ActiveDeadlineSeconds   int `json:"activeDeadlineSeconds,omitempty" binding:"omitempty,min=1" desc:"Hard wall-clock limit (seconds) before the run is terminated."`
	TTLSecondsAfterFinished int `json:"ttlSecondsAfterFinished,omitempty" binding:"omitempty,min=0" desc:"Seconds to retain a finished run before garbage collection."`
	BackoffLimit            int `json:"backoffLimit,omitempty" binding:"omitempty,min=0" desc:"Number of retries before the run is marked failed."`
	ProgressDeadlineSeconds int `json:"progressDeadlineSeconds,omitempty" binding:"omitempty,min=1" desc:"Seconds without progress before the run is considered stalled."`
}

// Run is a single Run (the compute MLRun produced by triggering a Job).
type Run struct {
	ID                UUID              `json:"id,omitempty" desc:"Stable run identifier."`
	Namespace         string            `json:"namespace" desc:"Platform tenant namespace the run belongs to."`
	TenantName        string            `json:"tenantName" desc:"Tenant identifier owning the run."`
	TenantDisplayName string            `json:"tenantDisplayName,omitempty" desc:"Human-readable tenant name."`
	ComputeNamespace  string            `json:"computeNamespace,omitempty" desc:"Underlying compute (Kubernetes) namespace executing the run."`
	Name              string            `json:"name" desc:"Run name (<job>-<n>)."`
	JobName           string            `json:"jobName,omitempty" desc:"Name of the Job definition that produced this run."`
	RunNumber         int               `json:"runNumber,omitempty" desc:"Monotonic per-job run sequence number."`
	DisplayName       string            `json:"displayName,omitempty" desc:"Human-readable run label."`
	Description       string            `json:"description,omitempty" desc:"Free-text run description."`
	Owner             string            `json:"owner" desc:"Username of the run owner."`
	OwnerID           UUID              `json:"ownerId,omitempty" desc:"User ID of the run owner."`
	Backend           Backend           `json:"backend" desc:"Compute backend/engine executing the run."`
	PoolName          string            `json:"poolName,omitempty" binding:"dns1123,max=40" desc:"Resource pool the run is scheduled onto."`
	UnitName          string            `json:"unitName,omitempty" binding:"dns1123,max=40" desc:"Resource unit (shape) within the pool."`
	Quota             string            `json:"quota,omitempty" desc:"ElasticQuota the run draws from."`
	Resources         ResourceMap       `json:"resources,omitempty" desc:"Aggregate resources reserved by the run."`
	Roles             []MLRunRoleStatus `json:"roles,omitempty" desc:"Live per-role status."`
	RunPolicy         RunPolicy         `json:"runPolicy,omitempty" desc:"Effective run-level execution limits."`
	Spec              MLRunSpec         `json:"spec,omitempty" desc:"Full resolved run spec (for the YAML view)."`
	Phase             RunPhase          `json:"phase,omitempty" desc:"Current run lifecycle phase."`
	Message           string            `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	ScheduledAt       *time.Time        `json:"scheduledAt,omitempty" desc:"Time the run was admitted by the scheduler (left Pending)."`
	StartedAt         *time.Time        `json:"startedAt,omitempty" desc:"Time the run started executing."`
	FinishedAt        *time.Time        `json:"finishedAt,omitempty" desc:"Time the run reached a terminal phase."`
	CreatedAt         time.Time         `json:"createdAt" desc:"Time the run was created."`
	UpdatedAt         time.Time         `json:"updatedAt" desc:"Time the run was last updated."`
}

// MLRunSpec is a pass-through mirror of compute's MLRun spec, surfaced on
// Run so the UI can render a full YAML view. The authoritative shape lives
// in compute.
type MLRunSpec struct {
	Backend    Backend        `json:"backend,omitempty" desc:"Compute backend/engine."`
	Scheduling map[string]any `json:"scheduling,omitempty" desc:"Scheduling directives (gang, priority, tolerations)."`
	Roles      []MLRunRole    `json:"roles,omitempty" desc:"Run topology roles."`
	RunPolicy  RunPolicy      `json:"runPolicy,omitempty" desc:"Run-level execution limits."`
}

// RunList is a page of Run.
type RunList struct {
	Items         []Run  `json:"items" desc:"Runs in this page."`
	Count         int    `json:"count" binding:"min=0" desc:"Number of runs in this page."`
	ContinueToken string `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool   `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// RunSummary is a per-definition roll-up of a Job's / Experiment's Runs. It
// rides on list responses (count + recent phases for the status strip) and on
// detail responses (latest-run phase), so list/detail pages need no extra call.
type RunSummary struct {
	Count       int        `json:"count" binding:"min=0" desc:"Total number of Runs of the definition."`
	Active      int        `json:"active,omitempty" desc:"Runs currently in a non-terminal phase."`
	Recent      []RunPhase `json:"recent,omitempty" desc:"Most-recent Run phases, oldest-to-newest, for the status strip."`
	LatestPhase RunPhase   `json:"latestPhase,omitempty" desc:"Phase of the most recent Run."`
	LatestRunAt *time.Time `json:"latestRunAt,omitempty" desc:"Creation time of the most recent Run."`
}

// ArtifactRef references an artifact version consumed by a run.
type ArtifactRef struct {
	Kind    ArtifactKind `json:"kind" desc:"Artifact kind (model or image)."`
	Name    string       `json:"name" desc:"Artifact definition name."`
	Version string       `json:"version" desc:"Artifact version."`
}

// JobSpec is a reusable Job template (snapshotted into an MLRun per run).
type JobSpec struct {
	Backend   Backend       `json:"backend" desc:"Compute backend/engine the runs use."`
	PoolName  string        `json:"poolName,omitempty" binding:"dns1123,max=40" desc:"Default resource pool for runs."`
	UnitName  string        `json:"unitName,omitempty" binding:"dns1123,max=40" desc:"Default resource unit (shape) within the pool."`
	Quota     string        `json:"quota,omitempty" desc:"Default ElasticQuota for runs."`
	Roles     []MLRunRole   `json:"roles" binding:"min=1" desc:"Run topology roles (at least one)."`
	RunPolicy RunPolicy     `json:"runPolicy,omitempty" desc:"Default run-level execution limits."`
	Artifacts []ArtifactRef `json:"artifacts,omitempty" desc:"Model/image artifact versions the runs consume."`
}

// Job is a Platform-owned reusable Job template.
type Job struct {
	ID          UUID        `json:"id" desc:"Stable job identifier."`
	Namespace   string      `json:"namespace" desc:"Platform tenant namespace the job belongs to."`
	TenantName  string      `json:"tenantName" desc:"Tenant identifier owning the job."`
	Name        string      `json:"name" desc:"Job definition name (unique within the tenant)."`
	DisplayName string      `json:"displayName,omitempty" desc:"Human-readable job label."`
	Description string      `json:"description,omitempty" desc:"Free-text job description."`
	Owner       string      `json:"owner" desc:"Username of the job owner."`
	OwnerID     UUID        `json:"ownerId,omitempty" desc:"User ID of the job owner."`
	Labels      StringMap   `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations StringMap   `json:"annotations,omitempty" desc:"User-defined annotations."`
	Spec        JobSpec     `json:"spec" desc:"Reusable run template."`
	RunSummary  *RunSummary `json:"runSummary,omitempty" desc:"Roll-up of the job's Runs (count + recent phases on lists, latest phase on detail)."`
	CreatedAt   time.Time   `json:"createdAt" desc:"Time the job was created."`
	UpdatedAt   time.Time   `json:"updatedAt" desc:"Time the job was last updated."`
}

// JobList is a page of Job.
type JobList struct {
	Items         []Job  `json:"items" desc:"Jobs in this page."`
	Count         int    `json:"count" binding:"min=0" desc:"Number of jobs in this page."`
	ContinueToken string `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool   `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// JobCreateRequest is the body of POST /jobs.
type JobCreateRequest struct {
	Name        string    `json:"name" binding:"required,dns1123,min=1,max=63" desc:"Job definition name (unique within the tenant)."`
	DisplayName string    `json:"displayName,omitempty" binding:"max=100" desc:"Human-readable job label."`
	Description string    `json:"description,omitempty" binding:"max=1000" desc:"Free-text job description."`
	Labels      StringMap `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations StringMap `json:"annotations,omitempty" desc:"User-defined annotations."`
	Spec        JobSpec   `json:"spec" binding:"required" desc:"Reusable run template."`
}

// JobPatchRequest is the body of PATCH /jobs/{name}.
type JobPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=100" desc:"Updated human-readable job label."`
	Description string    `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text job description."`
	Labels      StringMap `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations StringMap `json:"annotations,omitempty" desc:"Replacement annotation set."`
	Spec        JobSpec   `json:"spec,omitempty" desc:"Replacement run template."`
}

// RunTriggerRequest holds the narrow set of trigger-time overrides for a run.
type RunTriggerRequest struct {
	DisplayName string        `json:"displayName,omitempty" binding:"max=100" desc:"Display name for the triggered run."`
	Labels      StringMap     `json:"labels,omitempty" desc:"Labels for the triggered run."`
	Annotations StringMap     `json:"annotations,omitempty" desc:"Annotations for the triggered run."`
	PoolName    string        `json:"poolName,omitempty" binding:"dns1123,max=40" desc:"Override resource pool for this run."`
	UnitName    string        `json:"unitName,omitempty" binding:"dns1123,max=40" desc:"Override resource unit (shape) for this run."`
	Quota       string        `json:"quota,omitempty" desc:"Override ElasticQuota for this run."`
	Resources   ResourceMap   `json:"resources,omitempty" desc:"Override aggregate resources for this run."`
	Artifacts   []ArtifactRef `json:"artifacts,omitempty" desc:"Override artifact versions for this run."`
	Roles       []struct {
		Name string   `json:"name" desc:"Role name to override."`
		Args []string `json:"args,omitempty" desc:"Override container args for the role."`
		Env  []EnvVar `json:"env,omitempty" desc:"Override environment variables for the role."`
	} `json:"roles,omitempty" desc:"Per-role trigger-time overrides."`
}
