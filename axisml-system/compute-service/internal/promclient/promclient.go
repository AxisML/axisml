// Package promclient implements the compute-service MetricsProvider over the
// Prometheus HTTP query API. It executes range queries only; PromQL construction
// (the per-metric, per-workload label selection) lives in the calling handler,
// keeping this package a thin, deployment-neutral executor.
package promclient

import (
	"context"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// Client is a Prometheus-backed extensions.MetricsProvider.
type Client struct {
	api promv1.API
}

// New builds a metrics provider for the given Prometheus base URL. An empty URL
// yields a disabled provider (Enabled reports false) with no error, so a
// deployment without a metrics backend still starts cleanly.
func New(url string) (*Client, error) {
	if url == "" {
		return &Client{}, nil
	}
	c, err := promapi.NewClient(promapi.Config{Address: url})
	if err != nil {
		return nil, err
	}
	return &Client{api: promv1.NewAPI(c)}, nil
}

// Enabled reports whether a Prometheus endpoint is configured.
func (c *Client) Enabled() bool { return c.api != nil }

// QueryRange runs a PromQL range query over [now-rng, now] at step and returns
// the first matched series' samples, oldest first. The handler's queries
// aggregate to a single series, so any additional matrix rows are ignored.
func (c *Client) QueryRange(ctx context.Context, query string, rng, step time.Duration) ([]extensions.MetricSample, error) {
	if c.api == nil {
		return nil, nil
	}
	end := time.Now()
	val, _, err := c.api.QueryRange(ctx, query, promv1.Range{Start: end.Add(-rng), End: end, Step: step})
	if err != nil {
		return nil, err
	}
	m, ok := val.(model.Matrix)
	if !ok || len(m) == 0 {
		return []extensions.MetricSample{}, nil
	}
	out := make([]extensions.MetricSample, 0, len(m[0].Values))
	for _, p := range m[0].Values {
		out = append(out, extensions.MetricSample{Timestamp: p.Timestamp.Time(), Value: float64(p.Value)})
	}
	return out, nil
}
