package server

import "time"

// WorkspaceLifecycle is a workspace's lifecycle policy.
type WorkspaceLifecycle struct {
	IdleTimeoutSeconds int `json:"idleTimeoutSeconds,omitempty" binding:"min=0"`
}

// WorkspaceVolume is one data volume mounted into a workspace. An empty Name
// requests a new volume (sized by Size / StorageClass); a set Name mounts an
// existing volume. Used is live consumption (read-only).
type WorkspaceVolume struct {
	Name         string `json:"name,omitempty"`
	Size         string `json:"size,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
	MountPath    string `json:"mountPath" binding:"required"`
	Used         string `json:"used,omitempty"`
}

// WorkspaceEndpoint carries a workspace's reachable URLs.
type WorkspaceEndpoint struct {
	AccessURL   string `json:"accessUrl,omitempty"`
	InternalDNS string `json:"internalDns,omitempty"`
}

// Workspace is a long-running interactive dev container.
type Workspace struct {
	ID                UUID                  `json:"id"`
	Namespace         string                `json:"namespace"`
	TenantName        string                `json:"tenantName"`
	TenantDisplayName string                `json:"tenantDisplayName,omitempty"`
	ComputeNamespace  string                `json:"computeNamespace,omitempty"`
	Name              string                `json:"name"`
	DisplayName       string                `json:"displayName,omitempty"`
	Description       string                `json:"description,omitempty"`
	Owner             string                `json:"owner"`
	OwnerID           UUID                  `json:"ownerId,omitempty"`
	Image             string                `json:"image"`
	Command           []string              `json:"command,omitempty"`
	Args              []string              `json:"args,omitempty"`
	Env               []EnvVar              `json:"env,omitempty"`
	ContainerPort     int                   `json:"containerPort" binding:"min=1,max=65535"`
	PoolName          string                `json:"poolName,omitempty" binding:"dns1123,max=40"`
	UnitName          string                `json:"unitName,omitempty" binding:"dns1123,max=40"`
	Quota             string                `json:"quota,omitempty"`
	Resources         ResourceMap           `json:"resources,omitempty"`
	Volumes           []WorkspaceVolume     `json:"volumes,omitempty"`
	Lifecycle         WorkspaceLifecycle    `json:"lifecycle,omitempty"`
	Replicas          int                   `json:"replicas,omitempty" binding:"min=0"`
	ReadyReplicas     int                   `json:"readyReplicas,omitempty" binding:"min=0"`
	DesiredState      WorkspaceDesiredState `json:"desiredState,omitempty"`
	Phase             WorkspacePhase        `json:"phase,omitempty"`
	Message           string                `json:"message,omitempty"`
	Endpoint          WorkspaceEndpoint     `json:"endpoint,omitempty"`
	LastStartedAt     *time.Time            `json:"lastStartedAt,omitempty"`
	LastStoppedAt     *time.Time            `json:"lastStoppedAt,omitempty"`
	CreatedAt         time.Time             `json:"createdAt"`
	UpdatedAt         time.Time             `json:"updatedAt"`
}

// WorkspaceList is a page of Workspace.
type WorkspaceList struct {
	Items         []Workspace `json:"items"`
	Count         int         `json:"count" binding:"min=0"`
	ContinueToken string      `json:"continueToken,omitempty"`
	Partial       bool        `json:"partial,omitempty"`
}

// WorkspaceCreateRequest is the body of POST /workspaces. ContainerPort
// defaults server-side from the image when omitted.
type WorkspaceCreateRequest struct {
	Name          string             `json:"name" binding:"required,dns1123,min=1,max=40"`
	DisplayName   string             `json:"displayName,omitempty" binding:"max=100"`
	Description   string             `json:"description,omitempty" binding:"max=1000"`
	Image         string             `json:"image" binding:"required"`
	ContainerPort int                `json:"containerPort,omitempty" binding:"min=1,max=65535"`
	Command       []string           `json:"command,omitempty"`
	Args          []string           `json:"args,omitempty"`
	Env           []EnvVar           `json:"env,omitempty"`
	PoolName      string             `json:"poolName" binding:"required,dns1123,max=40"`
	UnitName      string             `json:"unitName" binding:"required,dns1123,max=40"`
	Quota         string             `json:"quota,omitempty"`
	Volumes       []WorkspaceVolume  `json:"volumes,omitempty"`
	Lifecycle     WorkspaceLifecycle `json:"lifecycle,omitempty"`
}

// WorkspacePatchRequest is the body of PATCH /workspaces/{name}.
type WorkspacePatchRequest struct {
	DisplayName string             `json:"displayName,omitempty" binding:"max=100"`
	Description string             `json:"description,omitempty" binding:"max=1000"`
	Lifecycle   WorkspaceLifecycle `json:"lifecycle,omitempty"`
}

// WorkspaceDeleteRequest is the optional body of DELETE /workspaces/{name}.
type WorkspaceDeleteRequest struct {
	DeletePVC bool `json:"deletePvc,omitempty"`
}
