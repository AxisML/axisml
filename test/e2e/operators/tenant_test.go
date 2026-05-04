//go:build e2e

package operators_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tenantv1 "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/test/e2e"
	"github.com/axisml/axisml/test/testutil"
)

// TestTenant_HappyPath creates a Tenant CR and asserts the deployed
// tenant-operator reconciles it into a Namespace + ElasticQuota. The deeper
// per-resource expansions (Secret/ConfigMap/SA/Role/RoleBinding) are already
// covered by tenant-operator/test/envtest L1 — this test focuses on the
// real-cluster path: image-loaded operator + helm-installed CRDs +
// koordinator's ElasticQuota CRD.
func TestTenant_HappyPath(t *testing.T) {
	ctx, cancel := setup(t)
	defer cancel()

	const (
		tenantName = "e2e-tenant-a"
		tenantNs   = "e2e-tenant-a-ns"
	)

	tenant := &tenantv1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{tenantv1.LabelTenantID: "uuid-e2e-a"},
		},
		Spec: tenantv1.TenantSpec{
			Namespace: tenantv1.NamespaceSpec{Name: tenantNs},
			Quotas: []tenantv1.QuotaSpec{
				{
					Pool: "default",
					Name: "default",
					Min:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
					Max:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))
	t.Cleanup(func() {
		// Fresh context — the test's parent ctx may already be cancelled by t.Fatal.
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelClean()
		// Tenant reconciler does NOT delete the namespace (design §6.1), so
		// we delete both explicitly to keep the cluster hermetic between runs.
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&tenantv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: tenantName}}, time.Minute)
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNs}}, 2*time.Minute)
	})

	var ns corev1.Namespace
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Name: tenantNs}, &ns, 60*time.Second)
	require.Equal(t, tenantv1.ManagedByValue, ns.Labels[tenantv1.LabelManagedBy])

	eqName := e2e.ElasticQuotaName(tenantName, "default", "default")
	var eq schedv1alpha1.ElasticQuota
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: eqName}, &eq, 60*time.Second)
	require.Equal(t, "2", eq.Spec.Max.Cpu().String())

	testutil.Eventually(t, 90*time.Second, time.Second, func() error {
		var got tenantv1.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != tenantv1.TenantPhaseActive {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})
}
