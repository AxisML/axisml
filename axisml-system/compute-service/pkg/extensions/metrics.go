package extensions

import (
	"context"
	"time"
)

// MetricSample is one (timestamp, value) sample of a metric series.
type MetricSample struct {
	Timestamp time.Time
	Value     float64
}

// MetricsProvider queries a metrics backend (Prometheus) for time series. It is
// optional: a composition root with no metrics backend injects a provider whose
// Enabled reports false, and the per-workload metrics routes then report
// metrics-unavailable rather than fabricating data.
type MetricsProvider interface {
	// Enabled reports whether a metrics backend is configured.
	Enabled() bool
	// QueryRange runs a PromQL range query over [now-rng, now] at the given step
	// and returns the sampled series, oldest first (empty when nothing matches).
	QueryRange(ctx context.Context, query string, rng, step time.Duration) ([]MetricSample, error)
}
