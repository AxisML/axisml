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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
	poolmod "github.com/axisml/axisml/components/compute-service/internal/resourcepool"
	unitmod "github.com/axisml/axisml/components/compute-service/internal/resourceunit"
)

// TestJobCreateRoundTrip exercises the namespace-keyed job pipeline:
// POST /jobs → DB row → reconciler tick → MLJob CR in envtest. Then
// GET, List, Cancel, Delete.
func TestJobCreateRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Seed a ResourcePool + ResourceUnit so the service has an ID to
	// reference. Pool name is unique; unit lives under it.
	pools := poolmod.NewService(gormDB)
	pool, err := pools.EnsureDefault(ctx, "jobs-e2e-pool")
	require.NoError(t, err)
	units := unitmod.NewService(gormDB)
	unitView, err := units.Create(ctx, pool.ID, unitmod.CreateInput{
		Name: "small",
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	})
	require.NoError(t, err)

	// Create the K8s namespace the MLJob CR will land in.
	const ns = "jobs-e2e-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}))

	// POST /api/v1/namespaces/jobs-e2e-ns/jobs.
	body := map[string]any{
		"name":           "my-job",
		"resourceUnitId": unitView.ID,
		"quota":          "axisml-default",
		"backend":        map[string]string{"name": "native", "engine": "job"},
		"roles": []map[string]any{
			{
				"name":     "worker",
				"replicas": 1,
				"template": map[string]any{
					"containers": []map[string]any{
						{"name": "main", "image": "busybox:1.36"},
					},
				},
			},
		},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	// Reconciler tick should create the MLJob CR. Poll for it.
	var cr mljobv1alpha1.MLJob
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "my-job"}, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLJob CR did not appear")
	assert.Equal(t, "axisml-default", cr.Spec.Scheduling.Quota)
	assert.Equal(t, "axisml-default", cr.Labels[mljobv1alpha1.LabelQuota])

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs/my-job", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, ns, got["namespace"])
	assert.Equal(t, "my-job", got["name"])

	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	assert.GreaterOrEqual(t, list.Total, int64(1))

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs", body, nil)
	requireStatus(t, rr, http.StatusConflict)

	// DELETE moves the row to Deleting; reconciler then deletes the CR.
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+ns+"/jobs/my-job", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	require.Eventually(t, func() bool {
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "my-job"}, &mljobv1alpha1.MLJob{})
		return err != nil
	}, 10*time.Second, 200*time.Millisecond, "MLJob CR was not reaped")
}

func TestJobValidation(t *testing.T) {
	rr := doJSON(t, context.Background(), http.MethodPost,
		"/api/v1/namespaces/x-ns/jobs",
		map[string]any{"name": "no-quota"}, nil)
	requireClientError(t, rr)
}
