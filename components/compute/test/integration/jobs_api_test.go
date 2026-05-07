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
	"sigs.k8s.io/controller-runtime/pkg/client"

	poolmod "github.com/axisml/axisml/components/compute/internal/resourcepool"
	unitmod "github.com/axisml/axisml/components/compute/internal/resourceunit"
)

// TestJob_CancelGuards verifies the Cancel API's pre-conditions: cancel
// against a freshly-created job (still in Creating because no operator is
// observing the CR in this test setup) returns 412 directing the caller
// to DELETE; cancel against a missing job returns 404 (covered in
// TestJob_NotFound). The full happy-path Cancel → Cancelled transition
// requires compute-operator updating the CR's status.Phase, which is
// out of scope for this test module's envtest.
func TestJob_CancelGuards(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pools := poolmod.NewService(gormDB)
	pool, err := pools.EnsureDefault(ctx, "jobs-cancel-pool")
	require.NoError(t, err)
	units := unitmod.NewService(gormDB)
	unitView, err := units.Create(ctx, pool.ID, unitmod.CreateInput{
		Name: "small-cancel",
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	})
	require.NoError(t, err)

	const ns = "jobs-cancel-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}))

	createBody := map[string]any{
		"name":           "cancel-me",
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
	rr := postJSON(t, "/api/v1/namespaces/"+ns+"/jobs", createBody)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Cancel a freshly-created job → 412 with "use DELETE" body.
	rr = postJSON(t, "/api/v1/namespaces/"+ns+"/jobs/cancel-me/cancel", nil)
	require.Equal(t, http.StatusPreconditionFailed, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "DELETE")
}

// TestJob_NotFound covers GET and cancel against a job name that doesn't
// exist. DELETE is intentionally idempotent (always 204 — see
// jobs_lifecycle_test.go for the happy-path delete + reconciler reap).
func TestJob_NotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	const ns = "jobs-nf-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	_ = c.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	})

	rr := doRequestJSON(t, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs/ghost", "")
	require.Equal(t, http.StatusNotFound, rr.Code)

	rr = postJSON(t, "/api/v1/namespaces/"+ns+"/jobs/ghost/cancel", nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestJob_ListPagination probes ?limit / ?offset against the list handler
// in a fresh namespace seeded with three jobs.
func TestJob_ListPagination(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pools := poolmod.NewService(gormDB)
	pool, err := pools.EnsureDefault(ctx, "jobs-list-pool")
	require.NoError(t, err)
	units := unitmod.NewService(gormDB)
	unitView, err := units.Create(ctx, pool.ID, unitmod.CreateInput{
		Name: "small-list",
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		},
	})
	require.NoError(t, err)

	const ns = "jobs-list-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	_ = c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	})

	for _, n := range []string{"a", "b", "c"} {
		rr := postJSON(t, "/api/v1/namespaces/"+ns+"/jobs", map[string]any{
			"name":           "job-" + n,
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
		})
		require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	}

	// limit=2 should return 2 items but total=3.
	rr := doRequestJSON(t, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/jobs?limit=2", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var page1 struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	require.NoError(t, decodeJSONBody(rr, &page1))
	assert.Len(t, page1.Items, 2)
	assert.Equal(t, int64(3), page1.Total)

	// limit=0 is a validation error.
	rr = doRequestJSON(t, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/jobs?limit=0", "")
	if rr.Code < 400 || rr.Code >= 500 {
		t.Fatalf("expected 4xx for limit=0, got %d", rr.Code)
	}
}
