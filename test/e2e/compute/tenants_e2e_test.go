//go:build e2e

package compute_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/e2e"
	"github.com/axisml/axisml/test/testutil"
)

// TestComputeAPI_TenantQuotaLifecycle: tenant create → Namespace + phase=Active,
// quota create → ElasticQuota, suspend → Suspended, unsuspend → Active.
// Cleanup deletes the namespace explicitly (tenant-operator does not, per design §6.1).
func TestComputeAPI_TenantQuotaLifecycle(t *testing.T) {
	_, c := e2e.SetupOrSkip(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	api := e2e.PortForwardCompute(t)

	const (
		tenantName = "e2e-tenant-quota"
		tenantNS   = "e2e-tenant-quota-ns"
		poolName   = "default" // bootstrap-created
		quotaName  = "e2e-q1"
	)

	t.Cleanup(func() {
		// Best-effort cleanup; ignore errors so the next run starts clean.
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelClean()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete, fmt.Sprintf("/api/v1/tenants/%s/quotas/%s", tenantName, quotaName), nil, nil).Body.Close()
		_ = api.DoJSON(t, cleanCtx, http.MethodDelete, fmt.Sprintf("/api/v1/tenants/%s", tenantName), nil, nil).Body.Close()
		// Tenant-operator does NOT delete Namespaces (design §6.1) — do it ourselves.
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNS}}, 2*time.Minute)
	})

	// 1. Create tenant.
	resp := api.DoJSON(t, ctx, http.MethodPost, "/api/v1/tenants", map[string]any{
		"name":      tenantName,
		"namespace": map[string]any{"name": tenantNS},
	}, nil)
	body := e2e.ReadBody(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create tenant: %s", e2e.PrettyResp(resp, body))

	// Tenant CR appears, then Namespace, then phase=Active.
	cr := &tenantv1alpha1.Tenant{}
	testutil.EventuallyExists(t, ctx, c, client.ObjectKey{Name: tenantName}, cr, 60*time.Second)
	var ns corev1.Namespace
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Name: tenantNS}, &ns, 60*time.Second)
	testutil.Eventually(t, 90*time.Second, time.Second, func() error {
		var got tenantv1alpha1.Tenant
		if err := c.Get(ctx, client.ObjectKey{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != tenantv1alpha1.TenantPhaseActive {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})

	// 2. Create quota.
	resp = api.DoJSON(t, ctx, http.MethodPost, fmt.Sprintf("/api/v1/tenants/%s/quotas", tenantName), map[string]any{
		"pool": poolName,
		"name": quotaName,
		"max":  map[string]any{"cpu": "1"},
	}, nil)
	body = e2e.ReadBody(resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "create quota: %s", e2e.PrettyResp(resp, body))

	// ElasticQuota lands in the tenant namespace.
	eqName := e2e.ElasticQuotaName(tenantName, poolName, quotaName)
	var eq schedv1alpha1.ElasticQuota
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: tenantNS, Name: eqName}, &eq, 60*time.Second)
	require.Equal(t, "1", eq.Spec.Max.Cpu().String())

	// 3. Suspend.
	resp = api.DoJSON(t, ctx, http.MethodPost, fmt.Sprintf("/api/v1/tenants/%s/suspend", tenantName), nil, nil)
	body = e2e.ReadBody(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, "suspend: %s", e2e.PrettyResp(resp, body))

	testutil.Eventually(t, 60*time.Second, time.Second, func() error {
		var got tenantv1alpha1.Tenant
		if err := c.Get(ctx, client.ObjectKey{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != tenantv1alpha1.TenantPhaseSuspended {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})

	// 4. Unsuspend.
	resp = api.DoJSON(t, ctx, http.MethodPost, fmt.Sprintf("/api/v1/tenants/%s/unsuspend", tenantName), nil, nil)
	body = e2e.ReadBody(resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, "unsuspend: %s", e2e.PrettyResp(resp, body))

	testutil.Eventually(t, 60*time.Second, time.Second, func() error {
		var got tenantv1alpha1.Tenant
		if err := c.Get(ctx, client.ObjectKey{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != tenantv1alpha1.TenantPhaseActive {
			return fmt.Errorf("phase=%q", got.Status.Phase)
		}
		return nil
	})
}
