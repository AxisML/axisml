package server

import "time"

// MetricPoint is one sample in a MetricSeries.
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp" desc:"Sample time (RFC3339)."`
	Value     float64   `json:"value" desc:"Sampled value at the timestamp."`
}

// MetricSeries is a single resource/serving metric's time series for a
// workload, sampled from the metrics backend (Prometheus) over a range.
type MetricSeries struct {
	Metric string        `json:"metric" desc:"Metric name that was queried."`
	Range  string        `json:"range" desc:"Query range window (e.g. 1h, 24h)."`
	Step   string        `json:"step,omitempty" desc:"Sampling step between points (e.g. 30s)."`
	Unit   string        `json:"unit,omitempty" desc:"Value unit (cores, bytes, percent, req/s, ms)."`
	Series []MetricPoint `json:"series" desc:"Sampled points, oldest first."`
}
