//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	mlservicev1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltpv1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// compute-service + compute-operator: traffic-config (MLTrafficPolicy).
// Drive the HTTP API; assert on the HTTP response, the materialized
// MLTrafficPolicy CR, and the derived weighted Gateway-API HTTPRoute.

// createReadyService creates an nginx MLService and waits for it to reach
// Ready (real Deployment + pod), so it is a valid traffic-policy member.
func createReadyService(t *testing.T, ctx context.Context, ns, quota, name string) {
	t.Helper()
	r, err := h.createMLService(ctx, ns, nginxMLServiceReq(name, h.cfg.DefaultPool, h.cfg.DefaultUnit, quota, nil))
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create member service %s: %d: %s", name, r.status, string(r.body))
	t.Cleanup(func() { _, _ = h.deleteMLService(context.Background(), ns, name) })

	eventually(t, h.cfg.PodReadyTimeout, func() error {
		var svc mlservicev1.MLService
		if err := h.get(ctx, ns, name, &svc); err != nil {
			return err
		}
		if svc.Status.Phase != mlservicev1.PhaseReady {
			return assertErr("member %s phase=%q want Ready", name, svc.Status.Phase)
		}
		return nil
	})
}

// httpRouteBackendWeights reads the derived HTTPRoute's backendRefs as
// serviceName -> weight via the unstructured accessor (gateway-api types are
// not pulled into the e2e module).
func httpRouteBackendWeights(ctx context.Context, ns, name string) (map[string]int64, error) {
	hr := httpRouteObj()
	if err := h.get(ctx, ns, name, hr); err != nil {
		return nil, err
	}
	rules, found, err := unstructured.NestedSlice(hr.Object, "spec", "rules")
	if err != nil || !found || len(rules) == 0 {
		return nil, assertErr("HTTPRoute %s has no spec.rules yet", name)
	}
	rule0, ok := rules[0].(map[string]any)
	if !ok {
		return nil, assertErr("HTTPRoute %s rule[0] not an object", name)
	}
	refs, found, err := unstructured.NestedSlice(rule0, "backendRefs")
	if err != nil || !found {
		return nil, assertErr("HTTPRoute %s rule[0] has no backendRefs", name)
	}
	out := map[string]int64{}
	for _, r := range refs {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		svcName, _, _ := unstructured.NestedString(m, "name")
		w, _, _ := unstructured.NestedInt64(m, "weight")
		out[svcName] = w
	}
	return out, nil
}

func TestTrafficPolicy_CanaryThroughGateway(t *testing.T) {
	ctx := context.Background()
	ns := sharedNS(t)
	quota := sharedQuota(t, ctx)

	stable := uniqueName("e2e-tp-stable")
	canary := uniqueName("e2e-tp-canary")
	policy := uniqueName("e2e-tp")

	createReadyService(t, ctx, ns, quota, stable)
	createReadyService(t, ctx, ns, quota, canary)

	// --- create -------------------------------------------------------------
	r, err := h.createTrafficPolicy(ctx, ns, canaryTrafficReq(policy, stable, canary))
	require.NoError(t, err)
	require.True(t, r.is2xx(), "create traffic policy: %d: %s", r.status, string(r.body))
	t.Cleanup(func() { _, _ = h.deleteTrafficPolicy(context.Background(), ns, policy) })

	// HTTP response carries the derived backend tuple + auto-filled endpoint path.
	var view csTrafficPolicyView
	require.NoError(t, r.decode(&view))
	assert.Equal(t, string(mltpv1.TrafficModeCanary), view.Mode)
	assert.Equal(t, mltpv1.BackendKindNative, view.Spec.Backend.Name)
	assert.Equal(t, "/services/"+ns+"/"+policy+"/", view.Spec.Endpoint.Path)

	// MLTrafficPolicy CR materializes with the derived backend tuple + weights.
	var cr mltpv1.MLTrafficPolicy
	eventually(t, h.cfg.CRProvisionTimeout, func() error { return h.get(ctx, ns, policy, &cr) })
	assert.Equal(t, mltpv1.BackendKindNative, cr.Spec.Backend.Name)
	assert.Equal(t, mltpv1.EngineHTTPRoute, cr.Spec.Backend.Engine)
	assert.Equal(t, mltpv1.TrafficModeCanary, cr.Spec.Mode)
	assert.NotEmpty(t, cr.Labels[mltpv1.LabelTrafficPolicyID])
	require.Len(t, cr.Spec.Backends, 2)

	// The operator derives a single weighted HTTPRoute over both members.
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		w, err := httpRouteBackendWeights(ctx, ns, policy)
		if err != nil {
			return err
		}
		if w[stable] != 90 || w[canary] != 10 {
			return assertErr("HTTPRoute weights=%v want %s:90 %s:10", w, stable, canary)
		}
		return nil
	})

	// --- split: shift to 50/50 ---------------------------------------------
	s, err := h.splitTrafficPolicy(ctx, ns, policy, csTrafficSplitReq{Backends: []csTrafficWeightUpdate{
		{ServiceName: stable, Weight: 50},
		{ServiceName: canary, Weight: 50},
	}})
	require.NoError(t, err)
	require.True(t, s.is2xx(), "split: %d: %s", s.status, string(s.body))
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		w, err := httpRouteBackendWeights(ctx, ns, policy)
		if err != nil {
			return err
		}
		if w[stable] != 50 || w[canary] != 50 {
			return assertErr("HTTPRoute weights=%v want 50/50", w)
		}
		return nil
	})

	// --- promote: canary becomes stable @100 (roles swap) ------------------
	p, err := h.promoteTrafficPolicy(ctx, ns, policy)
	require.NoError(t, err)
	require.True(t, p.is2xx(), "promote: %d: %s", p.status, string(p.body))
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		w, err := httpRouteBackendWeights(ctx, ns, policy)
		if err != nil {
			return err
		}
		if w[canary] != 100 || w[stable] != 0 {
			return assertErr("HTTPRoute weights=%v want canary:100 stable:0", w)
		}
		return nil
	})
	require.NoError(t, h.get(ctx, ns, policy, &cr))
	assert.Equal(t, mltpv1.RoleStable, roleForService(cr.Spec.Backends, canary), "promoted canary should be the new stable")

	// --- delete: policy CR removed, member services retained ---------------
	d, err := h.deleteTrafficPolicy(ctx, ns, policy)
	require.NoError(t, err)
	require.True(t, d.is2xx(), "delete: %d", d.status)
	eventually(t, h.cfg.CRProvisionTimeout, func() error {
		var gone mltpv1.MLTrafficPolicy
		if err := h.get(ctx, ns, policy, &gone); isNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}
		return assertErr("MLTrafficPolicy %s still present", policy)
	})
	// Members are not deleted by the policy.
	for _, svc := range []string{stable, canary} {
		g, err := h.getMLService(ctx, ns, svc)
		require.NoError(t, err)
		assert.True(t, g.is2xx(), "member service %s should survive policy delete: %d", svc, g.status)
	}
}

func roleForService(backends []mltpv1.BackendMember, name string) mltpv1.BackendRole {
	for _, b := range backends {
		if b.ServiceName == name {
			return b.Role
		}
	}
	return ""
}
