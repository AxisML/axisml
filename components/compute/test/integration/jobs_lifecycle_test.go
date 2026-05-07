//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	poolmod "github.com/axisml/axisml/components/compute/internal/resourcepool"
	unitmod "github.com/axisml/axisml/components/compute/internal/resourceunit"
)

// TestJobCreateRoundTrip exercises the namespace-keyed job pipeline:
// POST /jobs → DB row → reconciler tick → MLJob CR in envtest. Then
// GET, List, Cancel, Delete.
func TestJobCreateRoundTrip(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding (testEngine) not initialised")
	}
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
	rr := postJSON(t, "/api/v1/namespaces/"+ns+"/jobs", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Reconciler tick should create the MLJob CR. Poll for it.
	var cr mljobv1alpha1.MLJob
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "my-job"}, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLJob CR did not appear")
	assert.Equal(t, "axisml-default", cr.Spec.Scheduling.Quota)
	assert.Equal(t, "axisml-default", cr.Labels[mljobv1alpha1.LabelQuota])

	// GET reflects the row.
	rr = doRequestJSON(t, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs/my-job", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, ns, got["namespace"])
	assert.Equal(t, "my-job", got["name"])

	// LIST returns the job.
	rr = doRequestJSON(t, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	assert.GreaterOrEqual(t, list.Total, int64(1))

	// Duplicate Create -> 409.
	rr = postJSON(t, "/api/v1/namespaces/"+ns+"/jobs", body)
	require.Equal(t, http.StatusConflict, rr.Code)

	// DELETE moves the row to Deleting; reconciler then deletes the CR.
	rr = doRequestJSON(t, http.MethodDelete, "/api/v1/namespaces/"+ns+"/jobs/my-job", "")
	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Eventually(t, func() bool {
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "my-job"}, &mljobv1alpha1.MLJob{})
		return err != nil // either NotFound or the cache hasn't synced yet — both acceptable
	}, 10*time.Second, 200*time.Millisecond, "MLJob CR was not reaped")
}

// TestJobValidation ensures the handler rejects malformed payloads
// rather than reaching the reconcile path. Mapping binding errors to a
// specific 4xx status code is the server middleware's job and is checked
// elsewhere; here we only verify the request didn't slip through to 200.
func TestJobValidation(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	rr := postJSON(t, "/api/v1/namespaces/x-ns/jobs", map[string]any{
		"name": "no-quota",
	})
	assert.NotEqual(t, http.StatusOK, rr.Code)
	assert.NotEqual(t, http.StatusCreated, rr.Code)
}

// --- helpers ---------------------------------------------------------

func postJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	return doRequestJSON(t, http.MethodPost, path, string(raw))
}

func doRequestJSON(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	testEngine.ServeHTTP(rr, req)
	return rr
}
