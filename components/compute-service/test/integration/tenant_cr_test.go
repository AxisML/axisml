//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// TestTenant_CR_RoundTrip is the load-bearing end-to-end test for the
// Tenant reconciler: POST creates the PG row, the reconciler picks up
// phase=Creating, the Tenant CR materialises in envtest with the
// expected spec + identity label, then DELETE flows through to CR
// removal (status=Deleting → CR Delete → onDelete → status=Deleted).
func TestTenant_CR_RoundTrip(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped (docker likely unavailable)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	const tenantName = "tenant-cr-roundtrip"
	body := map[string]any{
		"name":      tenantName,
		"namespace": map[string]any{"name": tenantName + "-ns"},
		"quotas": []map[string]any{{
			"pool": "default", "name": "default",
			"min": map[string]string{"cpu": "1"},
			"max": map[string]string{"cpu": "4"},
		}},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", body, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+tenantName, nil, nil)
	})

	// Reconciler should produce the Tenant CR within a few ticks.
	var cr tenantv1alpha1.Tenant
	require.Eventually(t, func() bool {
		err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &cr)
		return err == nil
	}, 10*time.Second, 200*time.Millisecond, "Tenant CR did not appear")

	require.Equal(t, tenantName+"-ns", cr.Spec.Namespace.Name)
	require.Len(t, cr.Spec.Quotas, 1)
	require.NotEmpty(t, cr.Labels[tenantv1alpha1.LabelTenantID])

	// DELETE flows through to CR removal.
	rr = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+tenantName, nil, nil)
	requireStatus(t, rr, http.StatusNoContent)

	require.Eventually(t, func() bool {
		err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &tenantv1alpha1.Tenant{})
		return apierrors.IsNotFound(err)
	}, 10*time.Second, 200*time.Millisecond, "Tenant CR not reaped")
}

// TestTenant_CR_GenerationBumpsOnPatch verifies that PATCH on a tenant
// spec field (quotas[].max) bumps generation; the reconciler then
// observes generation != observed_generation and patches the CR.
func TestTenant_CR_GenerationBumpsOnPatch(t *testing.T) {
	if testEngine == nil {
		t.Skip("test engine not bootstrapped")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	const tenantName = "tenant-cr-gen"
	body := map[string]any{
		"name":      tenantName,
		"namespace": map[string]any{"name": tenantName + "-ns"},
		"quotas": []map[string]any{{
			"pool": "default", "name": "default",
			"min": map[string]string{"cpu": "1"},
			"max": map[string]string{"cpu": "2"},
		}},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/namespaces", body, nil)
	requireStatus(t, rr, http.StatusCreated)
	t.Cleanup(func() {
		_ = doJSON(t, ctx, http.MethodDelete, "/api/v1/namespaces/"+tenantName, nil, nil)
	})

	// Wait for the CR to appear with max.cpu=2.
	require.Eventually(t, func() bool {
		var cr tenantv1alpha1.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &cr); err != nil {
			return false
		}
		if len(cr.Spec.Quotas) == 0 {
			return false
		}
		return cr.Spec.Quotas[0].Max.Cpu().String() == "2"
	}, 10*time.Second, 200*time.Millisecond, "initial CR not yet visible")

	// Bump max via the quota sub-route (an identity-preserving PATCH).
	rr = doJSON(t, ctx, http.MethodPatch,
		"/api/v1/namespaces/"+tenantName+"/quotas/default/default",
		map[string]any{"max": map[string]string{"cpu": "8"}}, nil)
	requireStatus(t, rr, http.StatusOK)

	// The reconciler should propagate the new max to the CR.
	require.Eventually(t, func() bool {
		var cr tenantv1alpha1.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &cr); err != nil {
			return false
		}
		if len(cr.Spec.Quotas) == 0 {
			return false
		}
		return cr.Spec.Quotas[0].Max.Cpu().String() == "8"
	}, 10*time.Second, 200*time.Millisecond, "CR did not reflect bumped quota.max")
}
