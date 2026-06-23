package server

import "time"

// TrafficPolicyEndpoint is the stable external entry of a traffic policy. An empty
// Path auto-generates /services/<tenant>/<name>/. Immutable after creation.
type TrafficPolicyEndpoint struct {
	Path     string `json:"path,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// TrafficPolicyBackend is one member online service of a traffic policy. ActualPct
// and Ready are live, read-only fields sourced from compute.
type TrafficPolicyBackend struct {
	ServiceName string                   `json:"serviceName" binding:"required,dns1123,max=40"`
	Role        TrafficPolicyBackendRole `json:"role,omitempty"`
	Weight      int                      `json:"weight" binding:"min=0,max=100"`
	ActualPct   int                      `json:"actualPct,omitempty"`
	Ready       bool                     `json:"ready,omitempty"`
}

// TrafficPolicy fans one stable external entry's traffic across member online
// services by weight (weighted split / canary ramp / blue-green cutover).
// endpoint and mode are immutable after creation; only backend weights (and
// canary percent) change.
type TrafficPolicy struct {
	ID                UUID                   `json:"id"`
	Namespace         string                 `json:"namespace"`
	TenantName        string                 `json:"tenantName"`
	TenantDisplayName string                 `json:"tenantDisplayName,omitempty"`
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"displayName,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Owner             string                 `json:"owner"`
	OwnerID           UUID                   `json:"ownerId,omitempty"`
	Mode              TrafficPolicyMode      `json:"mode"`
	Endpoint          TrafficPolicyEndpoint  `json:"endpoint,omitempty"`
	AccessURL         string                 `json:"accessUrl,omitempty"`
	Backends          []TrafficPolicyBackend `json:"backends"`
	CanaryPercent     int                    `json:"canaryPercent,omitempty"`
	Phase             TrafficPolicyPhase     `json:"phase,omitempty"`
	Message           string                 `json:"message,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// TrafficPolicyList is a page of TrafficPolicy.
type TrafficPolicyList struct {
	Items         []TrafficPolicy `json:"items"`
	Count         int             `json:"count" binding:"min=0"`
	ContinueToken string          `json:"continueToken,omitempty"`
	Partial       bool            `json:"partial,omitempty"`
}

// TrafficPolicyBackendSpec is one backend in a create / split request.
type TrafficPolicyBackendSpec struct {
	ServiceName string                   `json:"serviceName" binding:"required,dns1123,max=40"`
	Role        TrafficPolicyBackendRole `json:"role,omitempty"`
	Weight      int                      `json:"weight,omitempty" binding:"min=0,max=100"`
}

// TrafficPolicyCreateRequest is the body of POST /traffic. For canary, supply
// one stable + one canary backend and an initial CanaryPercent; for weighted,
// supply N backends whose weights sum to 100.
type TrafficPolicyCreateRequest struct {
	Name          string                     `json:"name" binding:"required,dns1123,min=1,max=40"`
	DisplayName   string                     `json:"displayName,omitempty" binding:"max=100"`
	Description   string                     `json:"description,omitempty" binding:"max=1000"`
	Mode          TrafficPolicyMode          `json:"mode" binding:"required"`
	Endpoint      TrafficPolicyEndpoint      `json:"endpoint,omitempty"`
	Backends      []TrafficPolicyBackendSpec `json:"backends" binding:"required,min=1"`
	CanaryPercent int                        `json:"canaryPercent,omitempty" binding:"min=0,max=100"`
}

// TrafficPolicyPatchRequest edits display metadata only; weights go via /split.
type TrafficPolicyPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
}

// TrafficPolicySplitRequest adjusts the distribution. Weighted policies set
// backends[*].weight (Σ=100); canary policies set canaryPercent (stable=100−p).
type TrafficPolicySplitRequest struct {
	Backends      []TrafficPolicyBackendSpec `json:"backends,omitempty"`
	CanaryPercent *int                       `json:"canaryPercent,omitempty" binding:"omitempty,min=0,max=100"`
}
