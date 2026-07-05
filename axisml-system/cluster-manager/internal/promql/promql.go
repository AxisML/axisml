// Package promql builds (tenant, pool) PromQL and runs it against Prometheus for
// the cluster-manager per-pool metrics endpoint.
//
// A pool has no pod-level label on cAdvisor/DCGM series, so resource metrics are
// scoped by the tenant namespace and filtered to the pool's nodes: pod metrics
// are joined to their node via kube_pod_info, then kept only for nodes matching
// the pool's nodeSelector via kube_node_labels. This requires the cluster's
// kube-state-metrics to expose those node labels (metricLabelsAllowlist) and the
// DCGM ServiceMonitor to be enabled; where they are not, the query simply
// matches nothing and an empty (honest) series is returned. GPU utilisation is
// scoped by namespace only (DCGM's node dimension differs from cAdvisor's).
package promql

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

var (
	// ErrUnavailable is returned when no Prometheus backend is configured.
	ErrUnavailable = errors.New("cluster metrics are unavailable")
	// ErrBadRequest wraps an unsupported metric or a malformed range/step.
	ErrBadRequest = errors.New("invalid metrics request")
)

// Querier builds and runs (tenant, pool) metric queries.
type Querier struct {
	api promv1.API
}

// New builds a Querier for the Prometheus base URL. An empty URL yields a
// disabled Querier (Enabled reports false) with no error.
func New(url string) (*Querier, error) {
	if url == "" {
		return &Querier{}, nil
	}
	c, err := promapi.NewClient(promapi.Config{Address: url})
	if err != nil {
		return nil, err
	}
	return &Querier{api: promv1.NewAPI(c)}, nil
}

// Enabled reports whether a Prometheus endpoint is configured.
func (q *Querier) Enabled() bool { return q.api != nil }

type metricSpec struct {
	unit  string
	build func(ns, nodeMatch string) string
}

var specs = map[string]metricSpec{
	"cpu_util": {"cores", func(ns, nm string) string {
		base := fmt.Sprintf(`rate(container_cpu_usage_seconds_total{namespace=%q,container!="",container!="POD"}[5m])`, ns)
		return sumNodeScoped(base, ns, nm)
	}},
	"mem_util": {"GiB", func(ns, nm string) string {
		base := fmt.Sprintf(`container_memory_working_set_bytes{namespace=%q,container!="",container!="POD"}`, ns)
		return sumNodeScoped(base, ns, nm) + " / 1073741824"
	}},
	"gpu_util": {"percent", func(ns, _ string) string {
		return fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{exported_namespace=%q})`, ns)
	}},
}

// Series builds and runs the metric query for a (tenant namespace, pool) pair
// and projects the result into a PoolMetricSeries. nodeSelector is the pool's node
// selector (empty → namespace-wide, no per-pool node filter).
func (q *Querier) Series(ctx context.Context, namespace string, nodeSelector map[string]string, metric, rangeStr, stepStr string) (srv.PoolMetricSeries, error) {
	if !q.Enabled() {
		return srv.PoolMetricSeries{}, ErrUnavailable
	}
	spec, ok := specs[metric]
	if !ok {
		return srv.PoolMetricSeries{}, fmt.Errorf("%w: unsupported metric %q", ErrBadRequest, metric)
	}
	rng, err := parseWindow(rangeStr)
	if err != nil || rng <= 0 {
		return srv.PoolMetricSeries{}, fmt.Errorf("%w: invalid range %q", ErrBadRequest, rangeStr)
	}
	step, err := resolveStep(stepStr, rng)
	if err != nil || step <= 0 {
		return srv.PoolMetricSeries{}, fmt.Errorf("%w: invalid step %q", ErrBadRequest, stepStr)
	}

	end := time.Now()
	val, _, err := q.api.QueryRange(ctx, spec.build(namespace, nodeMatcher(nodeSelector)),
		promv1.Range{Start: end.Add(-rng), End: end, Step: step})
	if err != nil {
		return srv.PoolMetricSeries{}, err
	}
	out := srv.PoolMetricSeries{Metric: metric, Range: rangeStr, Step: stepStr, Unit: spec.unit, Series: []srv.PoolMetricPoint{}}
	if m, ok := val.(model.Matrix); ok && len(m) > 0 {
		for _, p := range m[0].Values {
			out.Series = append(out.Series, srv.PoolMetricPoint{Timestamp: p.Timestamp.Time(), Value: float64(p.Value)})
		}
	}
	return out, nil
}

// sumNodeScoped wraps a per-pod expression: with a nodeMatch it joins each pod
// to its node via kube_pod_info and keeps only nodes matching the pool's
// selector (kube_node_labels); without one it sums over the whole namespace.
func sumNodeScoped(base, ns, nodeMatch string) string {
	if nodeMatch == "" {
		return "sum(" + base + ")"
	}
	return fmt.Sprintf(
		`sum((%s * on(namespace,pod) group_left(node) kube_pod_info{namespace=%q}) * on(node) group_left() kube_node_labels{%s})`,
		base, ns, nodeMatch,
	)
}

// nodeMatcher renders a pool nodeSelector as a kube_node_labels matcher body,
// e.g. {node.axisml.io/pool: gpu} → `label_node_axisml_io_pool="gpu"`.
func nodeMatcher(sel map[string]string) string {
	if len(sel) == 0 {
		return ""
	}
	keys := make([]string, 0, len(sel))
	for k := range sel {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", ksmLabelName(k), sel[k]))
	}
	return strings.Join(parts, ",")
}

var nonLabelChar = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// ksmLabelName mirrors kube-state-metrics' label-name mangling.
func ksmLabelName(key string) string {
	return "label_" + nonLabelChar.ReplaceAllString(key, "_")
}

// parseWindow extends time.ParseDuration with day (d) and week (w) suffixes.
func parseWindow(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if n, ok := strings.CutSuffix(s, "d"); ok {
		days, err := strconv.Atoi(n)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if n, ok := strings.CutSuffix(s, "w"); ok {
		weeks, err := strconv.Atoi(n)
		if err != nil {
			return 0, err
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// resolveStep parses an explicit step, or derives ~50 points across the range
// (clamped to [15s, 10m]).
func resolveStep(s string, rng time.Duration) (time.Duration, error) {
	if strings.TrimSpace(s) != "" {
		return parseWindow(s)
	}
	step := rng / 50
	if step < 15*time.Second {
		step = 15 * time.Second
	}
	if step > 10*time.Minute {
		step = 10 * time.Minute
	}
	return step, nil
}
