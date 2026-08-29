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

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
)

// TestKubeProxy_ListMLRunPods drives GET /mlruns/{job}/pods: seed a Pod
// carrying the compute.axisml.io/run-id label and assert it shows up in the
// projected response.
func TestKubeProxy_ListMLRunPods(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	seedResourcePool(t, ctx, "kp-pool", "small")
	const ns = "kp-ns"
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, "kp-pool", resourceList("100", "1Ti"))

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("kp-job", "kp-pool", "small"), nil)
	requireStatus(t, rr, http.StatusCreated)

	// Pull the job's id so the seeded Pod label matches what the runtime
	// will query for.
	var view map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/kp-job", nil, &view)
	requireStatus(t, rr, http.StatusOK)
	jobID := view["id"].(string)

	// The runtime resolves instances by reading the MLRun CR's run-id label, so
	// wait for the reconciler to materialise the CR before seeding a Pod (Pods
	// only ever exist after the workload does).
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "kp-job"}, &mlrunv1alpha1.MLRun{}) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLRun CR did not appear")

	// Seed a Pod carrying compute.axisml.io/run-id.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "fake-pod",
			Labels:    map[string]string{mlrunv1alpha1.LabelRunID: jobID},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox:1.36"}},
		},
	}
	require.NoError(t, c.Create(ctx, pod))

	var out struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns/kp-job/pods", nil, &out)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, 1, out.Total, "kubeproxy must return the seeded pod")
	assert.Equal(t, "fake-pod", out.Items[0]["name"])
}

// TestKubeProxy_ListMLRunEvents lists events tied to an MLRun.
func TestKubeProxy_ListMLRunEvents(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const ns = "kp-evt-ns"
	mustCreateNamespace(t, ctx, ns)
	seedResourcePool(t, ctx, "kp-evt-pool", "small")

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("evt-job", "kp-evt-pool", "small"), nil)
	requireStatus(t, rr, http.StatusCreated)

	// We're not asserting events exist (envtest emits none by default); we
	// only assert the endpoint returns 200 with a well-formed envelope.
	var out struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns/evt-job/events", nil, &out)
	requireStatus(t, rr, http.StatusOK)
	assert.NotNil(t, out.Items)
}
