//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
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

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	poolmod "github.com/axisml/axisml/components/compute/internal/resourcepool"
	unitmod "github.com/axisml/axisml/components/compute/internal/resourceunit"
)

// TestServiceCreateRoundTrip exercises the namespace-keyed service pipeline:
// POST /services → DB row → reconciler tick → MLService CR in envtest. Then
// Get, List, Scale, Delete.
//
// Mirror of TestJobCreateRoundTrip but covering the MLService side of the
// service module — without it, services/{handler,reconciler,render,service}
// have no L1 coverage from the compute side at all. (compute-operator has
// its own envtest covering the CR controllers, but that doesn't exercise
// compute's outbox loop or HTTP layer.)
func TestServiceCreateRoundTrip(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding (testEngine) not initialised")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Seed a ResourcePool + ResourceUnit so the service has an ID to
	// reference. Distinct names from the jobs test to avoid collisions
	// when both run in the same package.
	pools := poolmod.NewService(gormDB)
	pool, err := pools.EnsureDefault(ctx, "services-e2e-pool")
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

	const ns = "services-e2e-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}))

	// POST /api/v1/namespaces/services-e2e-ns/services.
	body := map[string]any{
		"name":           "predictor",
		"resourceUnitId": unitView.ID,
		"quota":          "axisml-default",
		"backend":        map[string]string{"name": "native", "engine": "deployment"},
		"modelRef":       map[string]any{"name": "demo", "version": "v1"},
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
	rr := postJSON(t, "/api/v1/namespaces/"+ns+"/services", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Reconciler tick should create the MLService CR.
	var cr mlservicev1alpha1.MLService
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "predictor"}, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLService CR did not appear")
	assert.Equal(t, "axisml-default", cr.Spec.Scheduling.Quota)
	assert.Equal(t, "axisml-default", cr.Labels[mlservicev1alpha1.LabelQuota])
	assert.NotEmpty(t, cr.Labels[mlservicev1alpha1.LabelServiceID],
		"compute must stamp service-id label — operator validation rejects empty")
	require.Len(t, cr.Spec.Roles, 1)
	assert.Equal(t, int32(1), cr.Spec.Roles[0].Replicas)

	// GET reflects the row.
	rr = doRequestJSON(t, http.MethodGet, "/api/v1/namespaces/"+ns+"/services/predictor", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, ns, got["namespace"])
	assert.Equal(t, "predictor", got["name"])

	// LIST returns the service.
	rr = doRequestJSON(t, http.MethodGet, "/api/v1/namespaces/"+ns+"/services", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var list struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	assert.GreaterOrEqual(t, list.Total, int64(1))

	// Duplicate Create -> 409.
	rr = postJSON(t, "/api/v1/namespaces/"+ns+"/services", body)
	require.Equal(t, http.StatusConflict, rr.Code)

	// Scale to 3 — DB row updates immediately; reconciler propagates to CR.
	rr = postJSON(t, "/api/v1/namespaces/"+ns+"/services/predictor/scale",
		map[string]any{"replicas": 3})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	require.Eventually(t, func() bool {
		fresh := &mlservicev1alpha1.MLService{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "predictor"}, fresh); err != nil {
			return false
		}
		return len(fresh.Spec.Roles) > 0 && fresh.Spec.Roles[0].Replicas == 3
	}, 10*time.Second, 200*time.Millisecond, "CR.spec.roles[0].replicas not yet 3")

	// DELETE moves the row to Deleting; reconciler removes the CR.
	rr = doRequestJSON(t, http.MethodDelete, "/api/v1/namespaces/"+ns+"/services/predictor", "")
	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Eventually(t, func() bool {
		return c.Get(ctx, types.NamespacedName{Namespace: ns, Name: "predictor"}, &mlservicev1alpha1.MLService{}) != nil
	}, 10*time.Second, 200*time.Millisecond, "MLService CR was not reaped")
}

// TestServiceValidation ensures the handler rejects malformed payloads
// before they reach the reconciler. Mirrors TestJobValidation.
func TestServiceValidation(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	rr := postJSON(t, "/api/v1/namespaces/x-ns/services", map[string]any{
		"name": "no-quota",
	})
	assert.NotEqual(t, http.StatusOK, rr.Code)
	assert.NotEqual(t, http.StatusCreated, rr.Code)
}
