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

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
)

// TestMLRunCreateRoundTrip exercises the namespace-keyed job pipeline:
// seed ResourcePool CR → POST /mlruns → DB row → reconciler tick →
// MLRun CR in envtest. Then GET, List, conflict, Delete.
func TestMLRunCreateRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Seed a ResourcePool CR with one unit (per the new design, pool/unit
	// live in the K8s CRD, not PG).
	seedResourcePool(t, ctx, "jobs-e2e-pool", "small")

	const ns = "jobs-e2e-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}))

	body := buildMLRunCreateBody("my-job", "jobs-e2e-pool", "small")
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	// Reconciler tick should create the MLRun CR. Poll for it.
	var cr mlrunv1alpha1.MLRun
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "my-job"}, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLRun CR did not appear")
	assert.Equal(t, "axisml-default", cr.Spec.Scheduling.Quota)
	assert.Equal(t, "axisml-default", cr.Labels[mlrunv1alpha1.LabelQuota])

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/my-job", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, ns, got["namespace"])
	assert.Equal(t, "my-job", got["name"])

	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	assert.GreaterOrEqual(t, list.Total, int64(1))

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, nil)
	requireStatus(t, rr, http.StatusConflict)

	// DELETE moves the row to Deleting; reconciler then deletes the CR.
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+ns+"/mlruns/my-job", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	require.Eventually(t, func() bool {
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "my-job"}, &mlrunv1alpha1.MLRun{})
		return err != nil
	}, 10*time.Second, 200*time.Millisecond, "MLRun CR was not reaped")
}

func TestMLRunValidation(t *testing.T) {
	rr := doJSON(t, context.Background(), http.MethodPost,
		"/api/v1/namespaces/x-ns/mlruns",
		map[string]any{"name": "no-quota"}, nil)
	requireClientError(t, rr)
}
