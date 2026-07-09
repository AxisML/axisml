//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

func TestMLRun_CancelGuards(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-cancel-pool", "small-cancel")
	const ns = "jobs-cancel-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("cancel-me", "jobs-cancel-pool", "small-cancel"), nil)
	requireStatus(t, rr, http.StatusCreated)

	// Cancel before the operator has observed the CR returns 412 telling
	// the caller to DELETE instead of cancel; the row is still in Creating.
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns/cancel-me/cancel", nil, nil)
	requireStatus(t, rr, http.StatusPreconditionFailed)
	var p map[string]any
	require.NoError(t, decodeJSONBody(rr, &p))
	assert.Equal(t, string(apperrors.CodePrecondition), p["code"])
}

func TestMLRun_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const ns = "jobs-nf-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/ghost", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns/ghost/cancel", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)

	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/ghost/phase", nil, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

// TestMLRun_Phase checks the lightweight phase probe: a freshly-created run is
// in the default Creating phase before the operator observes it.
func TestMLRun_Phase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-phase-pool", "small-phase")
	const ns = "jobs-phase-ns"
	mustCreateNamespace(t, ctx, ns)

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("phase-run", "jobs-phase-pool", "small-phase"), nil)
	requireStatus(t, rr, http.StatusCreated)

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/phase-run/phase", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "Creating", got["phase"])
	assert.Equal(t, "phase-run", got["name"])
	// The phase probe is a lean projection — the heavy spec sub-tree must not
	// ride along.
	assert.NotContains(t, got, "spec")
}

// TestMLRun_BatchPhase covers the batch probe: ?names selects an explicit set
// (unresolved names omitted), an explicit empty ?names returns [], the names
// cap is enforced, ?labelSelector filters, and the no-filter form returns the
// whole namespace.
func TestMLRun_BatchPhase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-batch-pool", "small-batch")
	const ns = "jobs-batch-ns"
	mustCreateNamespace(t, ctx, ns)

	// batch-a/b carry the group=ml label; batch-c does not.
	for _, n := range []string{"batch-a", "batch-b"} {
		body := buildMLRunCreateBody(n, "jobs-batch-pool", "small-batch")
		body["labels"] = map[string]string{"group": "ml"}
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, nil)
		requireStatus(t, rr, http.StatusCreated)
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("batch-c", "jobs-batch-pool", "small-batch"), nil)
	requireStatus(t, rr, http.StatusCreated)

	type phaseList struct {
		Items []map[string]any `json:"items"`
		Count int              `json:"count"`
		Total int64            `json:"total"`
	}
	phaseByName := func(items []map[string]any) map[string]string {
		out := map[string]string{}
		for _, it := range items {
			name, _ := it["name"].(string)
			phase, _ := it["phase"].(string)
			out[name] = phase
			assert.NotContains(t, it, "spec")
		}
		return out
	}

	// ?names selects an explicit set; the unresolved "ghost" is omitted.
	var byNames phaseList
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns/phases?names=batch-a,ghost,batch-b", nil, &byNames)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, 2, byNames.Count)
	assert.Equal(t, map[string]string{"batch-a": "Creating", "batch-b": "Creating"}, phaseByName(byNames.Items))

	// An explicit empty ?names returns an empty set, NOT the whole namespace.
	var empty phaseList
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/phases?names=", nil, &empty)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, 0, empty.Count)
	assert.Empty(t, empty.Items)

	// More names than the cap → 400 validation.
	tooMany := make([]string, 201)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("n%d", i)
	}
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns/phases?names="+strings.Join(tooMany, ","), nil, nil)
	requireStatus(t, rr, http.StatusBadRequest)

	// ?labelSelector filters to the labelled subset (batch-a, batch-b).
	var bySelector phaseList
	rr = doJSON(t, ctx, http.MethodGet,
		"/api/v1/namespaces/"+ns+"/mlruns/phases?labelSelector="+url.QueryEscape("group=ml"), nil, &bySelector)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, int64(2), bySelector.Total)
	assert.Equal(t, map[string]string{"batch-a": "Creating", "batch-b": "Creating"}, phaseByName(bySelector.Items))

	// No filter → whole namespace (all three).
	var all phaseList
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/phases", nil, &all)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, int64(3), all.Total)

	// The static /phases route must not shadow the single-get param route.
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/batch-a", nil, nil)
	requireStatus(t, rr, http.StatusOK)
}

func TestMLRun_ListPagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-list-pool", "small-list")
	const ns = "jobs-list-ns"
	mustCreateNamespace(t, ctx, ns)

	for _, n := range []string{"a", "b", "c"} {
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
			buildMLRunCreateBody("job-"+n, "jobs-list-pool", "small-list"), nil)
		requireStatus(t, rr, http.StatusCreated)
	}

	var page1 struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns?limit=2", nil, &page1)
	requireStatus(t, rr, http.StatusOK)
	assert.Len(t, page1.Items, 2)
	assert.Equal(t, int64(3), page1.Total)

	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns?limit=0", nil, nil)
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

func buildMLRunCreateBody(name, poolName, unitName string) map[string]any {
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
