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

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

func TestMLRun_CancelGuards(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePoolWithSelector(t, ctx, "jobs-cancel-pool", "small-cancel", map[string]string{"queue.axisml.io/test": "blocked"})
	const ns = "jobs-cancel-ns"
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, "jobs-cancel-pool", resourceList("100", "1Ti"))

	var created map[string]any
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("cancel-me", "jobs-cancel-pool", "small-cancel"), &created)
	requireStatus(t, rr, http.StatusCreated)
	assert.Equal(t, "Queued", created["phase"])

	// A queued Run has no runtime object and can be cancelled directly.
	var cancelled map[string]any
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns/cancel-me/cancel", nil, &cancelled)
	requireStatus(t, rr, http.StatusAccepted)
	assert.Equal(t, "Cancelled", cancelled["phase"])

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("delete-me", "jobs-cancel-pool", "small-cancel"), nil)
	requireStatus(t, rr, http.StatusCreated)
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+ns+"/mlruns/delete-me", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	err = c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "delete-me"}, &mlrunv1alpha1.MLRun{})
	assert.True(t, apierrors.IsNotFound(err), "deleting a queued Run must not create a runtime object: %v", err)
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

// TestMLRun_Phase checks the lightweight phase probe for a Run whose pool has
// no matching node: it remains Queued and the projection omits the heavy spec.
func TestMLRun_Phase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePoolWithSelector(t, ctx, "jobs-phase-pool", "small-phase", map[string]string{"queue.axisml.io/test": "blocked"})
	const ns = "jobs-phase-ns"
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, "jobs-phase-pool", resourceList("100", "1Ti"))

	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("phase-run", "jobs-phase-pool", "small-phase"), nil)
	requireStatus(t, rr, http.StatusCreated)

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/phase-run/phase", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "Queued", got["phase"])
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

	seedResourcePoolWithSelector(t, ctx, "jobs-batch-pool", "small-batch", map[string]string{"queue.axisml.io/test": "blocked"})
	const ns = "jobs-batch-ns"
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, "jobs-batch-pool", resourceList("100", "1Ti"))

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
	assert.Equal(t, map[string]string{"batch-a": "Queued", "batch-b": "Queued"}, phaseByName(byNames.Items))

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
	assert.Equal(t, map[string]string{"batch-a": "Queued", "batch-b": "Queued"}, phaseByName(bySelector.Items))

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
	mustSetTenantQuota(t, ctx, ns, "jobs-list-pool", resourceList("100", "1Ti"))

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

func TestMLRun_PriorityValidationAndImmutability(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePoolWithSelector(t, ctx, "jobs-priority-contract-pool", "small", map[string]string{"queue.axisml.io/test": "blocked"})
	const ns = "jobs-priority-contract-ns"
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, "jobs-priority-contract-pool", resourceList("100", "1Ti"))

	invalid := buildMLRunCreateBody("invalid-priority", "jobs-priority-contract-pool", "small")
	invalid["annotations"] = map[string]string{mlrunv1alpha1.AnnotationPriority: "2147483648"}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", invalid, nil)
	requireStatus(t, rr, http.StatusBadRequest)

	body := buildMLRunCreateBody("priority-run", "jobs-priority-contract-pool", "small")
	body["annotations"] = map[string]string{mlrunv1alpha1.AnnotationPriority: "10"}
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/namespaces/"+ns+"/mlruns/priority-run",
		map[string]any{"annotations": map[string]string{mlrunv1alpha1.AnnotationPriority: "10", "note": "kept"}}, nil)
	requireStatus(t, rr, http.StatusOK)

	var problem map[string]any
	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/namespaces/"+ns+"/mlruns/priority-run",
		map[string]any{"annotations": map[string]string{mlrunv1alpha1.AnnotationPriority: "11"}}, &problem)
	requireStatus(t, rr, http.StatusConflict)
	assert.Equal(t, string(apperrors.CodeImmutableField), problem["code"])

	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/namespaces/"+ns+"/mlruns/priority-run",
		map[string]any{"annotations": map[string]string{"note": "priority removed"}}, &problem)
	requireStatus(t, rr, http.StatusConflict)

	withoutPriority := buildMLRunCreateBody("default-priority-run", "jobs-priority-contract-pool", "small")
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", withoutPriority, nil)
	requireStatus(t, rr, http.StatusCreated)
	rr = doJSON(t, ctx, http.MethodPatch, "/api/v1/namespaces/"+ns+"/mlruns/default-priority-run",
		map[string]any{"annotations": map[string]string{mlrunv1alpha1.AnnotationPriority: "0"}}, &problem)
	requireStatus(t, rr, http.StatusConflict)
}

func TestMLRun_QueueAdmitsHigherPriorityFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "jobs-priority-order-pool", "small")
	const ns = "jobs-priority-order-ns"
	mustCreateNamespace(t, ctx, ns)
	// Both runs first enter the durable queue. Raising the quota later makes
	// exactly one replica fit and lets the controller demonstrate its order.
	mustSetTenantQuota(t, ctx, ns, "jobs-priority-order-pool", resourceList("0", "0"))

	for name, priority := range map[string]string{"low": "-100", "high": "100"} {
		body := buildMLRunCreateBody(name, "jobs-priority-order-pool", "small")
		body["annotations"] = map[string]string{mlrunv1alpha1.AnnotationPriority: priority}
		var created map[string]any
		rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns", body, &created)
		requireStatus(t, rr, http.StatusCreated)
		assert.Equal(t, "Queued", created["phase"])
	}

	mustSetTenantQuota(t, ctx, ns, "jobs-priority-order-pool", resourceList("100m", "128Mi"))
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	var high mlrunv1alpha1.MLRun
	require.Eventually(t, func() bool {
		return c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "high"}, &high) == nil
	}, 10*time.Second, 100*time.Millisecond, "higher-priority MLRun was not submitted first")
	assert.Equal(t, "100", high.Annotations[mlrunv1alpha1.AnnotationPriority])

	err = c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "low"}, &mlrunv1alpha1.MLRun{})
	assert.True(t, apierrors.IsNotFound(err), "lower-priority MLRun must remain outside the runtime while quota is occupied: %v", err)
	var lowPhase map[string]any
	rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlruns/low/phase", nil, &lowPhase)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, "Queued", lowPhase["phase"])
}

func TestMLRun_ResourcePoolCapacityOverridesNodeSelectorInventory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		pool = "jobs-capacity-override-pool"
		unit = "small"
		ns   = "jobs-capacity-override-ns"
	)
	seedResourcePoolWithSelectorAndCapacity(t, ctx, pool, unit,
		map[string]string{"queue.axisml.io/test": "blocked"},
		resourceList("100m", "128Mi"))
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, pool, resourceList("100m", "128Mi"))

	var created map[string]any
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlruns",
		buildMLRunCreateBody("capacity-run", pool, unit), &created)
	requireStatus(t, rr, http.StatusCreated)
	assert.Equal(t, "Queued", created["phase"])

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return c.Get(ctx, client.ObjectKey{Namespace: ns, Name: "capacity-run"}, &mlrunv1alpha1.MLRun{}) == nil
	}, 10*time.Second, 100*time.Millisecond,
		"explicit ResourcePool capacity should override the unmatched node selector during admission")
}

// --- helpers --------------------------------------------------------------

// seedResourcePool creates (or reuses) a ResourcePool CR with one embedded
// unit so the poolcache Informer has something to resolve. Per the new
// design, compute reads pools from the K8s CRD — never from PG.
func seedResourcePool(t *testing.T, ctx context.Context, poolName, unitName string) {
	seedResourcePoolWithSelector(t, ctx, poolName, unitName, nil)
}

func seedResourcePoolWithSelector(t *testing.T, ctx context.Context, poolName, unitName string, selector map[string]string) {
	seedResourcePoolWithSelectorAndCapacity(t, ctx, poolName, unitName, selector, nil)
}

func seedResourcePoolWithSelectorAndCapacity(
	t *testing.T,
	ctx context.Context,
	poolName, unitName string,
	selector map[string]string,
	capacity corev1.ResourceList,
) {
	t.Helper()
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	pool := &axismlv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{Name: poolName},
		Spec: axismlv1alpha1.ResourcePoolSpec{
			NodeSelector: selector,
			Capacity:     capacity,
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

func resourceList(cpu, memory string) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:                    resource.MustParse(cpu),
		corev1.ResourceMemory:                 resource.MustParse(memory),
		corev1.ResourceName("nvidia.com/gpu"): resource.MustParse("100"),
	}
}

func mustSetTenantQuota(t *testing.T, ctx context.Context, tenant, pool string, max corev1.ResourceList) {
	t.Helper()
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	var current tenantv1alpha1.Tenant
	err = c.Get(ctx, client.ObjectKey{Name: tenant}, &current)
	if apierrors.IsNotFound(err) {
		current = tenantv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{Name: tenant},
			Spec: tenantv1alpha1.TenantSpec{
				Namespace: tenantv1alpha1.NamespaceSpec{Name: tenant},
				Quotas:    []tenantv1alpha1.QuotaSpec{{Pool: pool, Max: max}},
			},
		}
		require.NoError(t, c.Create(ctx, &current))
		return
	}
	require.NoError(t, err)
	for i := range current.Spec.Quotas {
		if current.Spec.Quotas[i].Pool == pool {
			current.Spec.Quotas[i].Max = max
			require.NoError(t, c.Update(ctx, &current))
			return
		}
	}
	current.Spec.Quotas = append(current.Spec.Quotas, tenantv1alpha1.QuotaSpec{Pool: pool, Max: max})
	require.NoError(t, c.Update(ctx, &current))
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
