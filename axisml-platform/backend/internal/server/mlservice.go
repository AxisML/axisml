package server

import "time"

// ServicePort is one named container port exposed by a service.
type ServicePort struct {
	Name string `json:"name" binding:"required,max=15" desc:"Port name (unique within the service)."`
	Port int    `json:"port" binding:"required,min=1,max=65535" desc:"Container port number the service listens on."`
}

// MLServiceRoute exposes an MLService through the gateway (enable toggle +
// path). Data-plane auth / rate-limit are out of scope until the dedicated
// API-key management feature lands.
type MLServiceRoute struct {
	Enabled bool   `json:"enabled,omitempty" desc:"Whether the service is exposed through the gateway."`
	Path    string `json:"path,omitempty" desc:"Gateway route path the service is served under."`
}

// MLService is a long-running online inference service.
type MLService struct {
	ID                UUID           `json:"id" desc:"Stable service identifier."`
	Namespace         string         `json:"namespace" desc:"Platform tenant namespace the service belongs to."`
	TenantName        string         `json:"tenantName" desc:"Tenant identifier owning the service."`
	TenantDisplayName string         `json:"tenantDisplayName,omitempty" desc:"Human-readable tenant name."`
	ComputeNamespace  string         `json:"computeNamespace,omitempty" desc:"Underlying compute (Kubernetes) namespace running the service."`
	Name              string         `json:"name" desc:"Service name (unique within the tenant)."`
	DisplayName       string         `json:"displayName,omitempty" desc:"Human-readable service label."`
	Description       string         `json:"description,omitempty" desc:"Free-text service description."`
	Owner             string         `json:"owner" desc:"Username of the service owner."`
	OwnerID           UUID           `json:"ownerId,omitempty" desc:"User ID of the service owner."`
	Backend           Backend        `json:"backend,omitempty" desc:"Compute backend/engine serving the model."`
	ModelName         string         `json:"modelName,omitempty" desc:"Model artifact definition name being served."`
	ModelVersion      string         `json:"modelVersion,omitempty" desc:"Model artifact version being served."`
	Image             string         `json:"image,omitempty" desc:"Serving container image reference."`
	Command           []string       `json:"command,omitempty" desc:"Container entrypoint override."`
	Args              []string       `json:"args,omitempty" desc:"Container args override."`
	Env               []EnvVar       `json:"env,omitempty" desc:"Environment variables injected into the serving pods."`
	Ports             []ServicePort  `json:"ports,omitempty" desc:"Container ports exposed by the service."`
	PoolName          string         `json:"poolName,omitempty" binding:"dns1123,max=40" desc:"Resource pool the service is scheduled onto."`
	UnitName          string         `json:"unitName,omitempty" binding:"dns1123,max=40" desc:"Resource unit (shape) within the pool."`
	Quota             string         `json:"quota,omitempty" desc:"ElasticQuota the service draws from."`
	Resources         ResourceMap    `json:"resources,omitempty" desc:"Per-replica resource requests/limits."`
	Replicas          int            `json:"replicas,omitempty" binding:"min=0" desc:"Desired replica count."`
	ReadyReplicas     int            `json:"readyReplicas,omitempty" binding:"min=0" desc:"Replicas that have passed readiness."`
	Route             MLServiceRoute `json:"route,omitempty" desc:"Gateway exposure settings for the service."`
	AccessURL         string         `json:"accessUrl,omitempty" desc:"Resolved URL clients use to reach the service."`
	Phase             MLServicePhase `json:"phase,omitempty" desc:"Current service lifecycle phase."`
	Message           string         `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	CreatedAt         time.Time      `json:"createdAt" desc:"Time the service was created."`
	UpdatedAt         time.Time      `json:"updatedAt" desc:"Time the service was last updated."`
}

// MLServiceList is a page of MLService.
type MLServiceList struct {
	Items         []MLService `json:"items" desc:"Services in this page."`
	Count         int         `json:"count" binding:"min=0" desc:"Number of services in this page."`
	ContinueToken string      `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool        `json:"partial,omitempty" desc:"True if the list was truncated by an upstream limit."`
}

// MLServiceCreateRequest is the body of POST /mlservices. Backend defaults
// server-side from the chosen model/image when omitted.
type MLServiceCreateRequest struct {
	Name         string         `json:"name" binding:"required,dns1123,min=1,max=40" desc:"Service name (unique within the tenant)."`
	DisplayName  string         `json:"displayName,omitempty" binding:"max=100" desc:"Human-readable service label."`
	Description  string         `json:"description,omitempty" binding:"max=1000" desc:"Free-text service description."`
	Backend      Backend        `json:"backend,omitempty" desc:"Compute backend/engine (defaults from the model/image when omitted)."`
	ModelName    string         `json:"modelName" binding:"required" desc:"Model artifact definition name to serve."`
	ModelVersion string         `json:"modelVersion" binding:"required" desc:"Model artifact version to serve."`
	Image        string         `json:"image" binding:"required" desc:"Serving container image reference."`
	Command      []string       `json:"command,omitempty" desc:"Container entrypoint override."`
	Args         []string       `json:"args,omitempty" desc:"Container args override."`
	Env          []EnvVar       `json:"env,omitempty" desc:"Environment variables injected into the serving pods."`
	Ports        []ServicePort  `json:"ports" binding:"required,min=1" desc:"Container ports exposed by the service (at least one)."`
	PoolName     string         `json:"poolName" binding:"required,dns1123,max=40" desc:"Resource pool to schedule the service onto."`
	UnitName     string         `json:"unitName" binding:"required,dns1123,max=40" desc:"Resource unit (shape) within the pool."`
	Quota        string         `json:"quota,omitempty" desc:"ElasticQuota the service draws from."`
	Replicas     int            `json:"replicas" binding:"required,min=0" desc:"Desired replica count."`
	Route        MLServiceRoute `json:"route,omitempty" desc:"Gateway exposure settings for the service."`
}

// MLServicePatchRequest is the body of PATCH /mlservices/{name}.
type MLServicePatchRequest struct {
	DisplayName string `json:"displayName,omitempty" binding:"max=100" desc:"Updated human-readable service label."`
	Description string `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text service description."`
}

// MLServiceScaleRequest is the body of POST /mlservices/{name}/scale.
type MLServiceScaleRequest struct {
	Replicas int `json:"replicas" binding:"required,min=0" desc:"Target replica count to scale the service to."`
}

// MetricPoint is one (timestamp, value) sample.
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp" desc:"Sample timestamp."`
	Value     float64   `json:"value" desc:"Sample value."`
}

// MetricSeries is a labelled time series of MetricPoint.
type MetricSeries struct {
	Metric     string        `json:"metric" desc:"Metric name (e.g. qps, latency)."`
	Range      string        `json:"range" desc:"Query time range (e.g. 1h)."`
	Step       string        `json:"step,omitempty" desc:"Sampling step between points (e.g. 1m)."`
	Percentile string        `json:"percentile,omitempty" desc:"Percentile the series represents (e.g. p95)."`
	Unit       string        `json:"unit,omitempty" desc:"Value unit (e.g. ms, req/s)."`
	Series     []MetricPoint `json:"series" desc:"Ordered (timestamp, value) samples."`
}
