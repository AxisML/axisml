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

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// TestMLServiceCreateRoundTrip exercises the namespace-keyed service pipeline:
// seed ResourcePool CR → POST /mlservices → DB row → reconciler tick →
// MLService CR in envtest. Then Get, List, Scale, Delete.
func TestMLServiceCreateRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedResourcePool(t, ctx, "services-e2e-pool", "small")

	const ns = "services-e2e-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}))
	mustSetTenantQuota(t, ctx, ns, "services-e2e-pool", resourceList("100", "1Ti"))

	body := map[string]any{
		"name":     "predictor",
		"poolName": "services-e2e-pool",
		"unitName": "small",
		"backend":  map[string]string{"name": "native", "engine": "deployment"},
		"roles": []map[string]any{
			{
				"name":     mlservicev1alpha1.DefaultRoleName,
				"replicas": 1,
				"template": map[string]any{
					"image": "nginx:1.27",
					"ports": []map[string]any{
						{"name": "http", "containerPort": 8080, "protocol": string(corev1.ProtocolTCP)},
					},
				},
			},
		},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	var cr mlservicev1alpha1.MLService
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "predictor"}, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLService CR did not appear")
	assert.Equal(t, "axisml-services-e2e-ns-services-e2e-pool", cr.Spec.Scheduling.Quota)
	assert.Equal(t, "axisml-services-e2e-ns-services-e2e-pool", cr.Labels[mlservicev1alpha1.LabelQuota])
	assert.NotEmpty(t, cr.Labels[mlservicev1alpha1.LabelServiceID])
	require.Len(t, cr.Spec.Roles, 1)
	assert.Equal(t, int32(1), cr.Spec.Roles[0].Replicas)

	var got map[string]any
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/predictor", nil, &got)
	requireStatus(t, rr, http.StatusOK)
	assert.Equal(t, ns, got["namespace"])
	assert.Equal(t, "predictor", got["name"])

	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	rr = doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices", nil, &list)
	requireStatus(t, rr, http.StatusOK)
	assert.GreaterOrEqual(t, list.Total, int64(1))

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, nil)
	requireStatus(t, rr, http.StatusConflict)

	// Scale to 3 — 202 Accepted per design (DB row mutated; reconciler
	// propagates async to the CR).
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices/predictor/scale",
		map[string]any{"replicas": 3}, nil)
	requireStatus(t, rr, http.StatusAccepted)

	require.Eventually(t, func() bool {
		fresh := &mlservicev1alpha1.MLService{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "predictor"}, fresh); err != nil {
			return false
		}
		return len(fresh.Spec.Roles) > 0 && fresh.Spec.Roles[0].Replicas == 3
	}, 10*time.Second, 200*time.Millisecond, "CR.spec.roles[0].replicas not yet 3")

	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+ns+"/mlservices/predictor", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "predictor"}, &mlservicev1alpha1.MLService{}) != nil
	}, 10*time.Second, 200*time.Millisecond, "MLService CR was not reaped")
}

// TestMLServiceIncrementalAdmission verifies that one replica is enough to
// leave Queued, while the runtime receives only the admitted count. Expanding
// tenant quota later admits and dispatches the remaining desired replicas.
func TestMLServiceIncrementalAdmission(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	const (
		ns   = "services-incremental-ns"
		pool = "services-incremental-pool"
	)
	seedResourcePool(t, ctx, pool, "small")
	mustCreateNamespace(t, ctx, ns)
	mustSetTenantQuota(t, ctx, ns, pool, corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("100m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	})

	body := map[string]any{
		"name": "incremental", "poolName": pool, "unitName": "small",
		"backend": map[string]string{"name": "native", "engine": "deployment"},
		"roles": []map[string]any{{
			"name": mlservicev1alpha1.DefaultRoleName, "replicas": 3,
			"template": map[string]any{
				"image": "nginx:1.27",
				"ports": []map[string]any{{"name": "http", "containerPort": 8080, "protocol": string(corev1.ProtocolTCP)}},
			},
		}},
	}
	var created map[string]any
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/mlservices", body, &created)
	requireStatus(t, rr, http.StatusCreated)
	require.Equal(t, "Queued", created["phase"])

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var cr mlservicev1alpha1.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "incremental"}, &cr); err != nil {
			return false
		}
		return len(cr.Spec.Roles) == 1 && cr.Spec.Roles[0].Replicas == 1
	}, 15*time.Second, 200*time.Millisecond, "runtime must receive only the first admitted replica")

	require.Eventually(t, func() bool {
		var got map[string]any
		rr := doJSON(t, ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/mlservices/incremental", nil, &got)
		if rr.Code != http.StatusOK {
			return false
		}
		status, _ := got["status"].(map[string]any)
		return status["admittedReplicas"] == float64(1) && status["admissionReason"] == "QuotaExceeded"
	}, 15*time.Second, 200*time.Millisecond, "remaining replicas must report a stable quota admission reason")

	mustSetTenantQuota(t, ctx, ns, pool, corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("300m"),
		corev1.ResourceMemory: resource.MustParse("384Mi"),
	})
	require.Eventually(t, func() bool {
		var cr mlservicev1alpha1.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "incremental"}, &cr); err != nil {
			return false
		}
		return len(cr.Spec.Roles) == 1 && cr.Spec.Roles[0].Replicas == 3
	}, 15*time.Second, 200*time.Millisecond, "remaining replicas were not admitted after quota expansion")
}

func TestMLServiceValidation(t *testing.T) {
	rr := doJSON(t, context.Background(), http.MethodPost,
		"/api/v1/namespaces/x-ns/mlservices",
		map[string]any{"name": "no-quota"}, nil)
	requireClientError(t, rr)
}
