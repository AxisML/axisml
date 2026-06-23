package server

import "time"

// ServicePort is one named container port exposed by a service.
type ServicePort struct {
	Name string `json:"name" binding:"required,max=15"`
	Port int    `json:"port" binding:"required,min=1,max=65535"`
}

// MLServiceRoute exposes an MLService through the gateway (enable toggle +
// path). Data-plane auth / rate-limit are out of scope until the dedicated
// API-key management feature lands.
type MLServiceRoute struct {
	Enabled bool   `json:"enabled,omitempty"`
	Path    string `json:"path,omitempty"`
}

// MLService is a long-running online inference service.
type MLService struct {
	ID                UUID           `json:"id"`
	Namespace         string         `json:"namespace"`
	TenantName        string         `json:"tenantName"`
	TenantDisplayName string         `json:"tenantDisplayName,omitempty"`
	ComputeNamespace  string         `json:"computeNamespace,omitempty"`
	Name              string         `json:"name"`
	DisplayName       string         `json:"displayName,omitempty"`
	Description       string         `json:"description,omitempty"`
	Owner             string         `json:"owner"`
	OwnerID           UUID           `json:"ownerId,omitempty"`
	Backend           Backend        `json:"backend,omitempty"`
	ModelName         string         `json:"modelName,omitempty"`
	ModelVersion      string         `json:"modelVersion,omitempty"`
	Image             string         `json:"image,omitempty"`
	Command           []string       `json:"command,omitempty"`
	Args              []string       `json:"args,omitempty"`
	Env               []EnvVar       `json:"env,omitempty"`
	Ports             []ServicePort  `json:"ports,omitempty"`
	PoolName          string         `json:"poolName,omitempty" binding:"dns1123,max=40"`
	UnitName          string         `json:"unitName,omitempty" binding:"dns1123,max=40"`
	Quota             string         `json:"quota,omitempty"`
	Resources         ResourceMap    `json:"resources,omitempty"`
	Replicas          int            `json:"replicas,omitempty" binding:"min=0"`
	ReadyReplicas     int            `json:"readyReplicas,omitempty" binding:"min=0"`
	Route             MLServiceRoute `json:"route,omitempty"`
	AccessURL         string         `json:"accessUrl,omitempty"`
	Phase             MLServicePhase `json:"phase,omitempty"`
	Message           string         `json:"message,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// MLServiceList is a page of MLService.
type MLServiceList struct {
	Items         []MLService `json:"items"`
	Count         int         `json:"count" binding:"min=0"`
	ContinueToken string      `json:"continueToken,omitempty"`
	Partial       bool        `json:"partial,omitempty"`
}

// MLServiceCreateRequest is the body of POST /mlservices. Backend defaults
// server-side from the chosen model/image when omitted.
type MLServiceCreateRequest struct {
	Name         string         `json:"name" binding:"required,dns1123,min=1,max=40"`
	DisplayName  string         `json:"displayName,omitempty" binding:"max=100"`
	Description  string         `json:"description,omitempty" binding:"max=1000"`
	Backend      Backend        `json:"backend,omitempty"`
	ModelName    string         `json:"modelName" binding:"required"`
	ModelVersion string         `json:"modelVersion" binding:"required"`
	Image        string         `json:"image" binding:"required"`
	Command      []string       `json:"command,omitempty"`
	Args         []string       `json:"args,omitempty"`
	Env          []EnvVar       `json:"env,omitempty"`
	Ports        []ServicePort  `json:"ports" binding:"required,min=1"`
	PoolName     string         `json:"poolName" binding:"required,dns1123,max=40"`
	UnitName     string         `json:"unitName" binding:"required,dns1123,max=40"`
	Quota        string         `json:"quota,omitempty"`
	Replicas     int            `json:"replicas" binding:"required,min=0"`
	Route        MLServiceRoute `json:"route,omitempty"`
}

// MLServicePatchRequest is the body of PATCH /mlservices/{name}.
type MLServicePatchRequest struct {
	DisplayName string `json:"displayName,omitempty" binding:"max=100"`
	Description string `json:"description,omitempty" binding:"max=1000"`
}

// MLServiceScaleRequest is the body of POST /mlservices/{name}/scale.
type MLServiceScaleRequest struct {
	Replicas int `json:"replicas" binding:"required,min=0"`
}

// MetricPoint is one (timestamp, value) sample.
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricSeries is a labelled time series of MetricPoint.
type MetricSeries struct {
	Metric     string        `json:"metric"`
	Range      string        `json:"range"`
	Step       string        `json:"step,omitempty"`
	Percentile string        `json:"percentile,omitempty"`
	Unit       string        `json:"unit,omitempty"`
	Series     []MetricPoint `json:"series"`
}
