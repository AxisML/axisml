package server

import "time"

// TrafficPolicyEndpoint is the stable external entry of a traffic policy. An empty
// Path auto-generates /services/<tenant>/<name>/. Immutable after creation.
type TrafficPolicyEndpoint struct {
	Path     string `json:"path,omitempty" desc:"External URL path; empty auto-generates /services/<tenant>/<name>/. Immutable after creation."`
	Hostname string `json:"hostname,omitempty" desc:"Optional external hostname for the entry. Immutable after creation."`
}

// TrafficPolicyBackend is one member online service of a traffic policy. ActualPct
// and Ready are live, read-only fields sourced from compute.
type TrafficPolicyBackend struct {
	ServiceName string                   `json:"serviceName" binding:"required,dns1123,max=40" desc:"Name of the member online service receiving a share of traffic."`
	Role        TrafficPolicyBackendRole `json:"role,omitempty" desc:"Backend role within the policy (stable, canary, member)."`
	Weight      int                      `json:"weight" binding:"min=0,max=100" desc:"Configured traffic weight (0-100); weights across backends sum to 100."`
	ActualPct   int                      `json:"actualPct,omitempty" desc:"Live percentage of traffic actually routed to this backend (read-only)."`
	Ready       bool                     `json:"ready,omitempty" desc:"Whether the backend service is ready to serve (read-only)."`
}

// TrafficPolicy fans one stable external entry's traffic across member online
// services by weight (weighted split / canary ramp / blue-green cutover).
// endpoint and mode are immutable after creation; only backend weights (and
// canary percent) change.
type TrafficPolicy struct {
	ID                UUID                   `json:"id" desc:"Stable traffic policy identifier."`
	Namespace         string                 `json:"namespace" desc:"Platform tenant namespace the policy belongs to."`
	TenantName        string                 `json:"tenantName" desc:"Tenant identifier owning the policy."`
	TenantDisplayName string                 `json:"tenantDisplayName,omitempty" desc:"Human-readable tenant name."`
	Name              string                 `json:"name" desc:"Traffic policy name (unique within the tenant)."`
	DisplayName       string                 `json:"displayName,omitempty" desc:"Human-readable policy label."`
	Description       string                 `json:"description,omitempty" desc:"Free-text policy description."`
	Owner             string                 `json:"owner" desc:"Username of the policy owner."`
	OwnerID           UUID                   `json:"ownerId,omitempty" desc:"User ID of the policy owner."`
	Mode              TrafficPolicyMode      `json:"mode" desc:"Routing mode (weighted, canary). Immutable after creation."`
	Endpoint          TrafficPolicyEndpoint  `json:"endpoint,omitempty" desc:"Stable external entry the traffic is fanned out from."`
	AccessURL         string                 `json:"accessUrl,omitempty" desc:"Resolved external URL clients call (read-only)."`
	Backends          []TrafficPolicyBackend `json:"backends" desc:"Member online services and their weights."`
	CanaryPercent     int                    `json:"canaryPercent,omitempty" desc:"For canary mode, percent of traffic on the canary backend (stable = 100−p)."`
	Phase             TrafficPolicyPhase     `json:"phase,omitempty" desc:"Current policy lifecycle phase."`
	Message           string                 `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	CreatedAt         time.Time              `json:"createdAt" desc:"Time the policy was created."`
	UpdatedAt         time.Time              `json:"updatedAt" desc:"Time the policy was last updated."`
}

// TrafficPolicyList is a page of TrafficPolicy.
type TrafficPolicyList struct {
	Items         []TrafficPolicy `json:"items" desc:"Traffic policies in this page."`
	Count         int             `json:"count" binding:"min=0" desc:"Number of policies in this page."`
	ContinueToken string          `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool            `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// TrafficPolicyBackendSpec is one backend in a create / split request.
type TrafficPolicyBackendSpec struct {
	ServiceName string                   `json:"serviceName" binding:"required,dns1123,max=40" desc:"Name of the member online service to route traffic to."`
	Role        TrafficPolicyBackendRole `json:"role,omitempty" desc:"Backend role within the policy (stable, canary, member)."`
	Weight      int                      `json:"weight,omitempty" binding:"min=0,max=100" desc:"Traffic weight (0-100); weights across backends sum to 100 for weighted mode."`
}

// TrafficPolicyCreateRequest is the body of POST /traffic. For canary, supply
// one stable + one canary backend and an initial CanaryPercent; for weighted,
// supply N backends whose weights sum to 100.
type TrafficPolicyCreateRequest struct {
	Name          string                     `json:"name" binding:"required,dns1123,min=1,max=40" desc:"Traffic policy name (unique within the tenant)."`
	DisplayName   string                     `json:"displayName,omitempty" binding:"max=100" desc:"Human-readable policy label."`
	Description   string                     `json:"description,omitempty" binding:"max=1000" desc:"Free-text policy description."`
	Mode          TrafficPolicyMode          `json:"mode" binding:"required" desc:"Routing mode (weighted, canary). Immutable after creation."`
	Endpoint      TrafficPolicyEndpoint      `json:"endpoint,omitempty" desc:"Stable external entry; empty path auto-generates one."`
	Backends      []TrafficPolicyBackendSpec `json:"backends" binding:"required,min=1" desc:"Member online services and their weights (≥1)."`
	CanaryPercent int                        `json:"canaryPercent,omitempty" binding:"min=0,max=100" desc:"Initial canary traffic percent for canary mode."`
}

// TrafficPolicyPatchRequest edits display metadata only; weights go via /split.
type TrafficPolicyPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"max=100" desc:"Updated human-readable policy label."`
	Description string    `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text policy description."`
	Labels      StringMap `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations StringMap `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// TrafficPolicySplitRequest adjusts the distribution. Weighted policies set
// backends[*].weight (Σ=100); canary policies set canaryPercent (stable=100−p).
type TrafficPolicySplitRequest struct {
	Backends      []TrafficPolicyBackendSpec `json:"backends,omitempty" desc:"Updated backend weights for weighted mode (weights sum to 100)."`
	CanaryPercent *int                       `json:"canaryPercent,omitempty" binding:"omitempty,min=0,max=100" desc:"Updated canary traffic percent for canary mode (stable = 100−p)."`
}
