package server

import "time"

// ResourceMeter is one resource dimension's used-vs-total for a (tenant, pool).
type ResourceMeter struct {
	Resource string  `json:"resource" desc:"Resource dimension (cpu, memory, nvidia.com/gpu)."`
	Used     float64 `json:"used" desc:"Amount currently in use (from the ElasticQuota status)."`
	Total    float64 `json:"total" desc:"The tenant's quota ceiling in this pool."`
	Unit     string  `json:"unit,omitempty" desc:"Value unit (cores, GiB, cards)."`
}

// PoolUsage is a tenant's resource utilisation within one pool: the ElasticQuota
// used value against the tenant's folded quota ceiling, per resource dimension.
type PoolUsage struct {
	Pool   string          `json:"pool" desc:"Resource pool name."`
	Tenant string          `json:"tenant" desc:"Tenant identifier the usage is scoped to."`
	Meters []ResourceMeter `json:"meters" desc:"Per-resource used/total meters."`
}

// PoolMetricPoint is one sample in a PoolMetricSeries.
type PoolMetricPoint struct {
	Timestamp time.Time `json:"timestamp" desc:"Sample time (RFC3339)."`
	Value     float64   `json:"value" desc:"Sampled value at the timestamp."`
}

// PoolMetricSeries is a (tenant, pool) resource-utilisation time series sampled
// from Prometheus over a range.
type PoolMetricSeries struct {
	Metric string            `json:"metric" desc:"Metric name that was queried."`
	Range  string            `json:"range" desc:"Query range window (e.g. 1h, 24h)."`
	Step   string            `json:"step,omitempty" desc:"Sampling step between points (e.g. 30s)."`
	Unit   string            `json:"unit,omitempty" desc:"Value unit (cores, GiB, percent)."`
	Series []PoolMetricPoint `json:"series" desc:"Sampled points, oldest first."`
}
