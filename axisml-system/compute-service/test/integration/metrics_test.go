//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// stubMetrics is an always-enabled MetricsProvider returning one canned sample,
// so the metrics routes exercise the full resolve → pod-list → query → project
// path without a real Prometheus.
type stubMetrics struct{}

func (stubMetrics) Enabled() bool { return true }

func (stubMetrics) QueryRange(_ context.Context, _ string, _, _ time.Duration) ([]extensions.MetricSample, error) {
	return []extensions.MetricSample{{Timestamp: time.Unix(1700000000, 0).UTC(), Value: 2.5}}, nil
}

var testMetrics = stubMetrics{}

// TestMLRunMetrics drives GET /mlruns/{name}/metrics: a workload with a live pod
// yields a sampled series; an unsupported metric is rejected.
func TestMLRunMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "metrics-pool", "small")

	const ns = "metrics-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))

	body := buildMLRunCreateBody("metrics-job", "metrics-pool", "small")
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	// Wait for the reconciled CR so its run-id label is readable, then seed a pod
	// carrying that label — the runtime resolves it into the PromQL pod selector.
	var cr mlrunv1alpha1.MLRun
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "metrics-job"}, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLRun CR did not appear")
	runID := cr.Labels[mlrunv1alpha1.LabelRunID]
	require.NotEmpty(t, runID)

	require.NoError(t, c.Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "metrics-job-worker-0", Namespace: ns,
			Labels: map[string]string{mlrunv1alpha1.LabelRunID: runID},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker", Image: "busybox"}}},
	}))

	var series struct {
		Metric string `json:"metric"`
		Unit   string `json:"unit"`
		Series []struct {
			Value float64 `json:"value"`
		} `json:"series"`
	}
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/metrics-job/metrics?metric=cpu_util&range=1h", nil, &series)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "cpu_util", series.Metric)
	assert.Equal(t, "cores", series.Unit)
	require.Len(t, series.Series, 1)
	assert.Equal(t, 2.5, series.Series[0].Value)

	// Serving metrics have no backing source → rejected as a client error.
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/metrics-job/metrics?metric=request_rate&range=1h", nil, nil)
	requireClientError(t, rr)
}
