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

	body := map[string]any{
		"name":     "predictor",
		"poolName": "services-e2e-pool",
		"unitName": "small",
		"quota":    "axisml-default",
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
	assert.Equal(t, "axisml-default", cr.Spec.Scheduling.Quota)
	assert.Equal(t, "axisml-default", cr.Labels[mlservicev1alpha1.LabelQuota])
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

func TestMLServiceValidation(t *testing.T) {
	rr := doJSON(t, context.Background(), http.MethodPost,
		"/api/v1/namespaces/x-ns/mlservices",
		map[string]any{"name": "no-quota"}, nil)
	requireClientError(t, rr)
}
