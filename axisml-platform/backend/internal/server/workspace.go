package server

import "time"

// WorkspaceLifecycle is a workspace's lifecycle policy.
type WorkspaceLifecycle struct {
	IdleTimeoutSeconds int `json:"idleTimeoutSeconds,omitempty" binding:"min=0" desc:"Idle duration (seconds) after which the workspace is auto-stopped; 0 disables auto-stop."`
}

// WorkspaceVolume is one data volume mounted into a workspace. An empty Name
// requests a new volume (sized by Size / StorageClass); a set Name mounts an
// existing volume. Used is live consumption (read-only).
type WorkspaceVolume struct {
	Name         string `json:"name,omitempty" desc:"Volume name; empty requests a new volume, set mounts an existing one."`
	Size         string `json:"size,omitempty" desc:"Requested capacity for a new volume (e.g. 50Gi)."`
	StorageClass string `json:"storageClass,omitempty" desc:"StorageClass backing a new volume."`
	MountPath    string `json:"mountPath" binding:"required" desc:"Path the volume is mounted at inside the container."`
	Used         string `json:"used,omitempty" desc:"Live consumed capacity of the volume (read-only)."`
}

// WorkspaceEndpoint carries a workspace's reachable URLs.
type WorkspaceEndpoint struct {
	AccessURL   string `json:"accessUrl,omitempty" desc:"External URL for reaching the workspace UI."`
	InternalDNS string `json:"internalDns,omitempty" desc:"In-cluster DNS name for the workspace service."`
}

// Workspace is a long-running interactive dev container.
type Workspace struct {
	ID                UUID                  `json:"id" desc:"Stable workspace identifier."`
	Namespace         string                `json:"namespace" desc:"Platform tenant namespace the workspace belongs to."`
	TenantName        string                `json:"tenantName" desc:"Tenant identifier owning the workspace."`
	TenantDisplayName string                `json:"tenantDisplayName,omitempty" desc:"Human-readable tenant name."`
	ComputeNamespace  string                `json:"computeNamespace,omitempty" desc:"Underlying compute (Kubernetes) namespace hosting the workspace."`
	Name              string                `json:"name" desc:"Workspace name (unique within the tenant)."`
	DisplayName       string                `json:"displayName,omitempty" desc:"Human-readable workspace label."`
	Description       string                `json:"description,omitempty" desc:"Free-text workspace description."`
	Owner             string                `json:"owner" desc:"Username of the workspace owner."`
	OwnerID           UUID                  `json:"ownerId,omitempty" desc:"User ID of the workspace owner."`
	Image             string                `json:"image" desc:"Container image for the dev environment (e.g. jupyter, code-server)."`
	Command           []string              `json:"command,omitempty" desc:"Container entrypoint override."`
	Args              []string              `json:"args,omitempty" desc:"Container args override."`
	Env               []EnvVar              `json:"env,omitempty" desc:"Environment variables injected into the container."`
	ContainerPort     int                   `json:"containerPort" binding:"min=1,max=65535" desc:"Port the dev server listens on inside the container."`
	PoolName          string                `json:"poolName,omitempty" binding:"dns1123,max=40" desc:"Resource pool the workspace is scheduled onto."`
	UnitName          string                `json:"unitName,omitempty" binding:"dns1123,max=40" desc:"Resource unit (shape) within the pool."`
	Quota             string                `json:"quota,omitempty" desc:"ElasticQuota the workspace draws from."`
	Resources         ResourceMap           `json:"resources,omitempty" desc:"Resources reserved by the workspace."`
	Volumes           []WorkspaceVolume     `json:"volumes,omitempty" desc:"Data volumes mounted into the workspace."`
	Lifecycle         WorkspaceLifecycle    `json:"lifecycle,omitempty" desc:"Lifecycle policy (e.g. idle auto-stop)."`
	Replicas          int                   `json:"replicas,omitempty" binding:"min=0" desc:"Desired pod count (0 when stopped)."`
	ReadyReplicas     int                   `json:"readyReplicas,omitempty" binding:"min=0" desc:"Pods that have passed readiness."`
	DesiredState      WorkspaceDesiredState `json:"desiredState,omitempty" desc:"User-requested run state (Running, Stopped)."`
	Phase             WorkspacePhase        `json:"phase,omitempty" desc:"Current workspace lifecycle phase."`
	Message           string                `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	Endpoint          WorkspaceEndpoint     `json:"endpoint,omitempty" desc:"Reachable URLs for the workspace."`
	LastStartedAt     *time.Time            `json:"lastStartedAt,omitempty" desc:"Time the workspace was last started."`
	LastStoppedAt     *time.Time            `json:"lastStoppedAt,omitempty" desc:"Time the workspace was last stopped."`
	CreatedAt         time.Time             `json:"createdAt" desc:"Time the workspace was created."`
	UpdatedAt         time.Time             `json:"updatedAt" desc:"Time the workspace was last updated."`
}

// WorkspaceList is a page of Workspace.
type WorkspaceList struct {
	Items         []Workspace `json:"items" desc:"Workspaces in this page."`
	Count         int         `json:"count" binding:"min=0" desc:"Number of workspaces in this page."`
	ContinueToken string      `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool        `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// WorkspaceCreateRequest is the body of POST /workspaces. ContainerPort
// defaults server-side from the image when omitted.
type WorkspaceCreateRequest struct {
	Name          string             `json:"name" binding:"required,dns1123,min=1,max=40" desc:"Workspace name (unique within the tenant)."`
	DisplayName   string             `json:"displayName,omitempty" binding:"max=100" desc:"Human-readable workspace label."`
	Description   string             `json:"description,omitempty" binding:"max=1000" desc:"Free-text workspace description."`
	Image         string             `json:"image" binding:"required" desc:"Container image for the dev environment."`
	ContainerPort int                `json:"containerPort,omitempty" binding:"min=1,max=65535" desc:"Port the dev server listens on; defaults from the image when omitted."`
	Command       []string           `json:"command,omitempty" desc:"Container entrypoint override."`
	Args          []string           `json:"args,omitempty" desc:"Container args override."`
	Env           []EnvVar           `json:"env,omitempty" desc:"Environment variables injected into the container."`
	PoolName      string             `json:"poolName" binding:"required,dns1123,max=40" desc:"Resource pool to schedule the workspace onto."`
	UnitName      string             `json:"unitName" binding:"required,dns1123,max=40" desc:"Resource unit (shape) within the pool."`
	Quota         string             `json:"quota,omitempty" desc:"ElasticQuota the workspace draws from."`
	Volumes       []WorkspaceVolume  `json:"volumes,omitempty" desc:"Data volumes to mount into the workspace."`
	Lifecycle     WorkspaceLifecycle `json:"lifecycle,omitempty" desc:"Lifecycle policy (e.g. idle auto-stop)."`
}

// WorkspacePatchRequest is the body of PATCH /workspaces/{name}.
type WorkspacePatchRequest struct {
	DisplayName string             `json:"displayName,omitempty" binding:"max=100" desc:"Updated human-readable workspace label."`
	Description string             `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text workspace description."`
	Lifecycle   WorkspaceLifecycle `json:"lifecycle,omitempty" desc:"Replacement lifecycle policy."`
}

// WorkspaceDeleteRequest is the optional body of DELETE /workspaces/{name}.
type WorkspaceDeleteRequest struct {
	DeletePVC bool `json:"deletePvc,omitempty" desc:"When true, also delete the workspace's persistent volumes."`
}
