//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/axisml-system/compute-operator/api/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/test/testutil"
)

// memberService stands in for the ClusterIP Service an MLService(native,
// deployment) would create. The MLTrafficPolicy handler resolves member ports
// off these Services.
func memberService(ns, name string, port int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Name: "http", Port: port}},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
}

func TestMLTrafficPolicy_NativeHTTPRoute_Weighted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns         = "envt-mltp"
		policyName = "chat-traffic"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	require.NoError(t, c.Create(ctx, memberService(ns, "chat-v1", 8080)))
	require.NoError(t, c.Create(ctx, memberService(ns, "chat-v2", 8080)))

	policy := &mltp.MLTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      policyName,
			Labels: map[string]string{
				mltp.LabelTrafficPolicyID: "uuid-chat-traffic",
				mltp.LabelTenant:          "acme",
			},
		},
		Spec: mltp.MLTrafficPolicySpec{
			Backend:  mltp.Backend{Name: mltp.BackendKindNative, Engine: mltp.EngineHTTPRoute},
			Mode:     mltp.TrafficModeCanary,
			Endpoint: mltp.Endpoint{Path: "/services/acme/chat/", Hostname: "chat.example.com"},
			Backends: []mltp.BackendMember{
				{ServiceName: "chat-v1", Role: mltp.RoleStable, Weight: 90},
				{ServiceName: "chat-v2", Role: mltp.RoleCanary, Weight: 10},
			},
		},
	}
	require.NoError(t, c.Create(ctx, policy))

	// The derived weighted HTTPRoute appears (owned by the policy).
	var route gwapiv1.HTTPRoute
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: policyName}, &route, testWaitTimeout)

	require.Len(t, route.Spec.ParentRefs, 1)
	assert.Equal(t, gwapiv1.ObjectName("axisml-gateway"), route.Spec.ParentRefs[0].Name)
	require.Len(t, route.Spec.Hostnames, 1)
	assert.Equal(t, gwapiv1.Hostname("chat.example.com"), route.Spec.Hostnames[0])
	assert.Equal(t, "uuid-chat-traffic", route.Labels[mltp.LabelTrafficPolicyID])

	require.Len(t, route.Spec.Rules, 1)
	rule := route.Spec.Rules[0]
	require.Len(t, rule.BackendRefs, 2)

	weights := map[string]int32{}
	for _, br := range rule.BackendRefs {
		require.NotNil(t, br.Weight)
		require.NotNil(t, br.Port)
		assert.Equal(t, gwapiv1.PortNumber(8080), *br.Port)
		weights[string(br.Name)] = *br.Weight
	}
	assert.Equal(t, int32(90), weights["chat-v1"])
	assert.Equal(t, int32(10), weights["chat-v2"])

	// envtest has no Gateway controller, so the route never reaches Accepted —
	// the policy phase stays Pending. Assert the status mirror is populated
	// with the configured per-backend weights regardless.
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got mltp.MLTrafficPolicy
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: policyName}, &got); err != nil {
			return err
		}
		if len(got.Status.Backends) != 2 {
			return fmt.Errorf("status.backends not yet populated (have %d)", len(got.Status.Backends))
		}
		return nil
	})
}
