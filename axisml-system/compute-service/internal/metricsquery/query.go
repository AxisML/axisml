// Package metricsquery builds workload PromQL and executes it against a
// Prometheus MetricsProvider, projecting the result into the MetricSeries DTO.
//
// cAdvisor / DCGM series do not carry AxisML workload labels, so callers first
// resolve the workload's live pod names (via the ComputeRuntime) and pass them
// here; the query selects by namespace + pod name. Only the resource metrics
// with a real backing source in the standard deployment (kube-prometheus-stack
// cAdvisor + the DCGM exporter) are supported; serving metrics (request_rate /
// latency / error_rate) have no scraped source yet and are rejected as
// unsupported rather than fabricated.
package metricsquery

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// metricSpec renders the PromQL for a supported metric over a namespace + pod
// regex and names the value unit.
type metricSpec struct {
	unit  string
	build func(ns, podRe string) string
}

var specs = map[string]metricSpec{
	"cpu_util": {"cores", func(ns, re string) string {
		return fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace=%q,pod=~%q,container!="",container!="POD"}[5m]))`, ns, re)
	}},
	"mem_util": {"bytes", func(ns, re string) string {
		return fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace=%q,pod=~%q,container!="",container!="POD"})`, ns, re)
	}},
	"gpu_util": {"percent", func(ns, re string) string {
		return fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{exported_namespace=%q,exported_pod=~%q})`, ns, re)
	}},
}

// Querier builds and runs workload metric queries.
type Querier struct {
	provider extensions.MetricsProvider
}

// NewQuerier constructs a Querier. provider may be nil (Enabled reports false).
func NewQuerier(p extensions.MetricsProvider) *Querier { return &Querier{provider: p} }

// Enabled reports whether a metrics backend is available.
func (q *Querier) Enabled() bool { return q.provider != nil && q.provider.Enabled() }

// Series resolves metric over the given pods and returns a MetricSeries. An empty
// pod set yields an empty (honest) series — the workload simply has no live pods
// to sample.
func (q *Querier) Series(ctx context.Context, namespace string, podNames []string, metric, rangeStr, stepStr string) (server.MetricSeries, error) {
	if !q.Enabled() {
		return server.MetricSeries{}, apperrors.New(apperrors.CodeUnavailable, "workload metrics are unavailable")
	}
	spec, ok := specs[metric]
	if !ok {
		return server.MetricSeries{}, apperrors.Newf(apperrors.CodeValidation, "metric %q is not available in this deployment", metric)
	}
	rng, err := parseWindow(rangeStr)
	if err != nil || rng <= 0 {
		return server.MetricSeries{}, apperrors.Newf(apperrors.CodeValidation, "invalid range %q", rangeStr)
	}
	step, err := resolveStep(stepStr, rng)
	if err != nil || step <= 0 {
		return server.MetricSeries{}, apperrors.Newf(apperrors.CodeValidation, "invalid step %q", stepStr)
	}

	out := server.MetricSeries{Metric: metric, Range: rangeStr, Step: stepStr, Unit: spec.unit, Series: []server.MetricPoint{}}
	names := sanitize(podNames)
	if len(names) == 0 {
		return out, nil
	}
	samples, err := q.provider.QueryRange(ctx, spec.build(namespace, strings.Join(names, "|")), rng, step)
	if err != nil {
		return server.MetricSeries{}, apperrors.Wrap(apperrors.CodeUnavailable, "query metrics", err)
	}
	for _, s := range samples {
		out.Series = append(out.Series, server.MetricPoint{Timestamp: s.Timestamp, Value: s.Value})
	}
	return out, nil
}

// PodNames extracts non-empty pod names from a PodList.
func PodNames(list *corev1.PodList) []string {
	if list == nil {
		return nil
	}
	out := make([]string, 0, len(list.Items))
	for i := range list.Items {
		if n := list.Items[i].Name; n != "" {
			out = append(out, n)
		}
	}
	return out
}

// sanitize regex-escapes each pod name so it is matched literally inside the
// `pod=~"a|b"` alternation.
func sanitize(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n != "" {
			out = append(out, regexp.QuoteMeta(n))
		}
	}
	return out
}

// parseWindow parses a Prometheus-style window, extending time.ParseDuration
// with day (d) and week (w) suffixes used by the metrics API (e.g. 24h, 7d).
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
// (clamped to [15s, 10m]) when unset.
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
