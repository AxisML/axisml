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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

func TestJob_CancelGuards(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-cancel-pool", "small-cancel")
	const ns = "jobs-cancel-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs",
		buildJobCreateBody("cancel-me", "jobs-cancel-pool", "small-cancel"), nil)
	requireStatus(t, rr, http.StatusCreated)

	// Cancel before the operator has observed the CR returns 412 telling
	// the caller to DELETE instead of cancel; the row is still in Creating.
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs/cancel-me/cancel", nil, nil)
	requireStatus(t, rr, http.StatusPreconditionFailed)
	var p map[string]any
	require.NoError(t, decodeJSONBody(rr, &p))
	assert.Equal(t, string(apperrors.CodePrecondition), p["code"])
}

func TestJob_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const ns = "jobs-nf-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs/ghost", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs/ghost/cancel", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestJob_ListPagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-list-pool", "small-list")
	const ns = "jobs-list-ns"
	mustCreateNamespace(t, ctx, ns)

	for _, n := range []string{"a", "b", "c"} {
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/jobs",
			buildJobCreateBody("job-"+n, "jobs-list-pool", "small-list"), nil)
		requireStatus(t, rr, http.StatusCreated)
	}

	var page1 struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs?limit=2", nil, &page1)
	requireStatus(t, rr, http.StatusOK)
	assert.Len(t, page1.Items, 2)
	assert.Equal(t, int64(3), page1.Total)

	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/jobs?limit=0", nil, nil)
	requireClientError(t, rr)
}

// --- helpers --------------------------------------------------------------

// seedResourcePool creates (or reuses) a ResourcePool CR with one embedded
// unit so the poolcache Informer has something to resolve. Per the new
// design, compute reads pools from the K8s CRD — never from PG.
func seedResourcePool(t *testing.T, ctx context.Context, poolName, unitName string) {
	t.Helper()
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	pool := &axismlv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{Name: poolName},
		Spec: axismlv1alpha1.ResourcePoolSpec{
			Units: []axismlv1alpha1.ResourceUnit{{
				Name: unitName,
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}},
		},
	}
	if err := c.Create(ctx, pool); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("seed ResourcePool: %v", err)
	}
}

func mustCreateNamespace(t *testing.T, ctx context.Context, ns string) {
	t.Helper()
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create namespace %q: %v", ns, err)
	}
}

func buildJobCreateBody(name, poolName, unitName string) map[string]any {
	return map[string]any{
		"name":     name,
		"poolName": poolName,
		"unitName": unitName,
		"quota":    "axisml-default",
		"backend":  map[string]string{"name": "native", "engine": "job"},
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
}
