//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/components/compute-service/internal/store"
)

// seedReadyMLService inserts a services row already in phase=Ready so the
// traffic-policy member validation (kind=service + Ready + native family)
// passes without a running compute-operator.
func seedReadyMLService(t *testing.T, ctx context.Context, ns, name string) {
	t.Helper()
	row := &store.MLService{
		ID:          uuid.New(),
		Namespace:   ns,
		Name:        name,
		Kind:        "service",
		Phase:       "Ready",
		Spec:        datatypes.JSON(`{"backend":{"name":"native","engine":"deployment"}}`),
		Labels:      datatypes.JSON(`{}`),
		Annotations: datatypes.JSON(`{}`),
		StatusJSON:  datatypes.JSON(`{}`),
		Generation:  1,
		// observed == generation so the service reconciler treats the seed as
		// already in sync and doesn't try to emit an MLService CR for it.
		ObservedGeneration: 1,
	}
	require.NoError(t, gormDB.WithContext(ctx).Create(row).Error)
}

func TestTrafficPolicy_CanaryLifecycle(t *testing.T) {
	if testEngine == nil {
		t.Skip("handlers not bootstrapped (docker unavailable)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const (
		ns         = "tp-e2e"
		policyName = "chat-traffic"
	)
	seedReadyMLService(t, ctx, ns, "chat-v1")
	seedReadyMLService(t, ctx, ns, "chat-v2")

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	// Namespaced CRs need the K8s namespace to exist in envtest.
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))

	// --- create -------------------------------------------------------------
	body := map[string]any{
		"name":     policyName,
		"mode":     "canary",
		"endpoint": map[string]any{"hostname": "chat.example.com"},
		"backends": []map[string]any{
			{"serviceName": "chat-v1", "role": "stable", "weight": 90},
			{"serviceName": "chat-v2", "role": "canary", "weight": 10},
		},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/traffic-policies", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	// A member service cannot be deleted while the active policy references it.
	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/namespaces/"+ns+"/mlservices/chat-v1", nil, nil)
	requireStatus(t, rr, http.StatusConflict)

	key := types.NamespacedName{Namespace: ns, Name: policyName}
	var cr mltp.MLTrafficPolicy
	require.Eventually(t, func() bool {
		return c.Get(ctx, key, &cr) == nil
	}, 10*time.Second, 200*time.Millisecond, "MLTrafficPolicy CR did not appear")

	assert.Equal(t, mltp.BackendKindNative, cr.Spec.Backend.Name)
	assert.Equal(t, mltp.EngineHTTPRoute, cr.Spec.Backend.Engine)
	assert.Equal(t, mltp.TrafficModeCanary, cr.Spec.Mode)
	assert.Equal(t, "/services/tp-e2e/chat-traffic/", cr.Spec.Endpoint.Path) // auto-filled
	assert.NotEmpty(t, cr.Labels[mltp.LabelTrafficPolicyID])
	require.Len(t, cr.Spec.Backends, 2)
	assert.Equal(t, int32(90), weightOf(cr.Spec.Backends, "chat-v1"))
	assert.Equal(t, int32(10), weightOf(cr.Spec.Backends, "chat-v2"))

	// --- occupancy: a second policy over the same member is rejected --------
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/traffic-policies", map[string]any{
		"name": "chat-traffic-2",
		"mode": "canary",
		"backends": []map[string]any{
			{"serviceName": "chat-v1", "role": "stable", "weight": 90},
			{"serviceName": "chat-v2", "role": "canary", "weight": 10},
		},
	}, nil)
	requireClientError(t, rr)

	// --- split: shift to 50/50 ---------------------------------------------
	rr = doJSON(t, ctx, http.MethodPost,
		"/api/v1/namespaces/"+ns+"/traffic-policies/"+policyName+"/split", map[string]any{
			"backends": []map[string]any{
				{"serviceName": "chat-v1", "weight": 50},
				{"serviceName": "chat-v2", "weight": 50},
			},
		}, nil)
	requireStatus(t, rr, http.StatusAccepted)
	require.Eventually(t, func() bool {
		var got mltp.MLTrafficPolicy
		if c.Get(ctx, key, &got) != nil {
			return false
		}
		return weightOf(got.Spec.Backends, "chat-v2") == 50
	}, 10*time.Second, 200*time.Millisecond, "split weight did not propagate to CR")

	// --- promote: canary becomes stable @100, roles swap -------------------
	rr = doJSON(t, ctx, http.MethodPost,
		"/api/v1/namespaces/"+ns+"/traffic-policies/"+policyName+"/promote", nil, nil)
	requireStatus(t, rr, http.StatusAccepted)
	require.Eventually(t, func() bool {
		var got mltp.MLTrafficPolicy
		if c.Get(ctx, key, &got) != nil {
			return false
		}
		return weightOf(got.Spec.Backends, "chat-v2") == 100 &&
			roleOf(got.Spec.Backends, "chat-v2") == mltp.RoleStable &&
			roleOf(got.Spec.Backends, "chat-v1") == mltp.RoleCanary
	}, 10*time.Second, 200*time.Millisecond, "promote did not swap roles/weights on CR")

	// --- delete: CR removed, member services untouched ---------------------
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+ns+"/traffic-policies/"+policyName, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	require.Eventually(t, func() bool {
		return c.Get(ctx, key, &mltp.MLTrafficPolicy{}) != nil
	}, 10*time.Second, 200*time.Millisecond, "MLTrafficPolicy CR was not deleted")

	// Removing the policy releases its members for deletion.
	rr = doJSON(t, ctx, http.MethodDelete,
		"/api/v1/namespaces/"+ns+"/mlservices/chat-v1", nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
}

func weightOf(backends []mltp.BackendMember, name string) int32 {
	for _, b := range backends {
		if b.ServiceName == name {
			return b.Weight
		}
	}
	return -1
}

func roleOf(backends []mltp.BackendMember, name string) mltp.BackendRole {
	for _, b := range backends {
		if b.ServiceName == name {
			return b.Role
		}
	}
	return ""
}
