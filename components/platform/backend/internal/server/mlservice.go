package server

import "time"

// MLServiceRouteAuth configures auth on an MLService route.
type MLServiceRouteAuth struct {
	Type MLServiceRouteAuthType `json:"type"`
	JWT  struct {
		Issuer   string `json:"issuer,omitempty"`
		Audience string `json:"audience,omitempty"`
		JwksURI  string `json:"jwksUri,omitempty"`
	} `json:"jwt,omitempty"`
	APIKey struct {
		SecretRef struct {
			Name string `json:"name"`
			Key  string `json:"key,omitempty"`
		} `json:"secretRef,omitempty"`
	} `json:"apiKey,omitempty"`
}

// MLServiceRouteRateLimit configures per-route rate limiting.
type MLServiceRouteRateLimit struct {
	RequestsPerSecond int `json:"requestsPerSecond,omitempty" binding:"min=1"`
	Burst             int `json:"burst,omitempty" binding:"min=1"`
}

// MLServiceRoute is the gateway route exposing an MLService.
type MLServiceRoute struct {
	Enabled   bool                    `json:"enabled,omitempty"`
	Path      string                  `json:"path,omitempty"`
	Hostname  string                  `json:"hostname,omitempty"`
	Auth      MLServiceRouteAuth      `json:"auth,omitempty"`
	RateLimit MLServiceRouteRateLimit `json:"rateLimit,omitempty"`
	Timeout   string                  `json:"timeout,omitempty"`
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
	Image             string         `json:"image,omitempty"`
	Command           []string       `json:"command,omitempty"`
	Args              []string       `json:"args,omitempty"`
	Env               []EnvVar       `json:"env,omitempty"`
	ContainerPort     int            `json:"containerPort,omitempty" binding:"min=1,max=65535"`
	PoolName          string         `json:"poolName,omitempty" binding:"dns1123,max=40"`
	ResourcePoolName  string         `json:"resourcePoolName,omitempty"`
	UnitName          string         `json:"unitName,omitempty" binding:"dns1123,max=40"`
	ResourceUnitName  string         `json:"resourceUnitName,omitempty"`
	Quota             string         `json:"quota,omitempty"`
	Resources         ResourceMap    `json:"resources,omitempty"`
	Replicas          int            `json:"replicas,omitempty" binding:"min=0"`
	ReadyReplicas     int            `json:"readyReplicas,omitempty" binding:"min=0"`
	Endpoint          string         `json:"endpoint,omitempty"`
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

// MLServiceCreateRequest is the body of POST /mlservices.
type MLServiceCreateRequest struct {
	Name                    string         `json:"name" binding:"required,dns1123,min=1,max=40"`
	DisplayName             string         `json:"displayName,omitempty" binding:"max=100"`
	Description             string         `json:"description,omitempty" binding:"max=1000"`
	Backend                 Backend        `json:"backend" binding:"required"`
	Image                   string         `json:"image" binding:"required"`
	ContainerPort           int            `json:"containerPort" binding:"required,min=1,max=65535"`
	Command                 []string       `json:"command,omitempty"`
	Args                    []string       `json:"args,omitempty"`
	Env                     []EnvVar       `json:"env,omitempty"`
	PoolName                string         `json:"poolName" binding:"required,dns1123,max=40"`
	UnitName                string         `json:"unitName" binding:"required,dns1123,max=40"`
	Quota                   string         `json:"quota,omitempty"`
	Replicas                int            `json:"replicas" binding:"required,min=0"`
	ProgressDeadlineSeconds int            `json:"progressDeadlineSeconds,omitempty" binding:"min=1"`
	Route                   MLServiceRoute `json:"route,omitempty"`
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
