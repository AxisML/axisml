//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/test/testutil"
)

// TestMLTrafficPolicy_WeightTransitions verifies the operator re-renders the
// derived HTTPRoute backend weights as the policy spec changes through the
// canary state machine: initial 90/10 -> split 50/50 -> promote 100/0. Ported
// from the e2e suite (which exercised the same transitions end-to-end through a
// real gateway); the operator's render-on-spec-change is hermetic, so envtest is
// the proper home. Initial-weight rendering is covered by
// TestMLTrafficPolicy_NativeHTTPRoute_Weighted.
func TestMLTrafficPolicy_WeightTransitions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns         = "envt-mltp-trans"
		policyName = "rollout-traffic"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	require.NoError(t, c.Create(ctx, memberService(ns, "api-v1", 8080)))
	require.NoError(t, c.Create(ctx, memberService(ns, "api-v2", 8080)))

	policy := &mltp.MLTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      policyName,
			Labels: map[string]string{
				mltp.LabelTrafficPolicyID: "uuid-rollout",
				mltp.LabelTenant:          "acme",
			},
		},
		Spec: mltp.MLTrafficPolicySpec{
			Backend:  mltp.Backend{Name: mltp.BackendKindNative, Engine: mltp.EngineHTTPRoute},
			Mode:     mltp.TrafficModeCanary,
			Endpoint: mltp.Endpoint{Path: "/services/acme/api/", Hostname: "api.example.com"},
			Backends: []mltp.BackendMember{
				{ServiceName: "api-v1", Role: mltp.RoleStable, Weight: 90},
				{ServiceName: "api-v2", Role: mltp.RoleCanary, Weight: 10},
			},
		},
	}
	require.NoError(t, c.Create(ctx, policy))

	var route gwapiv1.HTTPRoute
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: policyName}, &route, testWaitTimeout)
	requireRouteWeights(t, ctx, c, ns, policyName, map[string]int32{"api-v1": 90, "api-v2": 10})

	// Split: shift weight toward the canary (50/50).
	setPolicyWeights(t, ctx, c, ns, policyName, 50, 50)
	requireRouteWeights(t, ctx, c, ns, policyName, map[string]int32{"api-v1": 50, "api-v2": 50})

	// Promote: canary takes all traffic (100/0).
	setPolicyWeights(t, ctx, c, ns, policyName, 0, 100)
	requireRouteWeights(t, ctx, c, ns, policyName, map[string]int32{"api-v1": 0, "api-v2": 100})
}

// setPolicyWeights updates the stable/canary backend weights on the policy spec.
func setPolicyWeights(t *testing.T, ctx context.Context, c client.Client, ns, name string, stable, canary int32) {
	t.Helper()
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var p mltp.MLTrafficPolicy
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &p); err != nil {
			return err
		}
		for i := range p.Spec.Backends {
			switch p.Spec.Backends[i].Role {
			case mltp.RoleStable:
				p.Spec.Backends[i].Weight = stable
			case mltp.RoleCanary:
				p.Spec.Backends[i].Weight = canary
			}
		}
		return c.Update(ctx, &p)
	})
}

// requireRouteWeights polls the derived HTTPRoute until its backendRefs carry the
// expected per-service weights.
func requireRouteWeights(t *testing.T, ctx context.Context, c client.Client, ns, name string, want map[string]int32) {
	t.Helper()
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var route gwapiv1.HTTPRoute
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &route); err != nil {
			return err
		}
		if len(route.Spec.Rules) != 1 {
			return fmt.Errorf("expected 1 rule, have %d", len(route.Spec.Rules))
		}
		got := map[string]int32{}
		for _, br := range route.Spec.Rules[0].BackendRefs {
			if br.Weight != nil {
				got[string(br.Name)] = *br.Weight
			}
		}
		for svc, w := range want {
			if got[svc] != w {
				return fmt.Errorf("weight[%s]=%d want %d (have %v)", svc, got[svc], w, got)
			}
		}
		return nil
	})
}
