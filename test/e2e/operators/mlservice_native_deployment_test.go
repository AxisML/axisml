//go:build e2e

package operators_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tenantv1 "github.com/axisml-io/axisml/components/operators/tenant-operator/api/v1alpha1"
	mlsvcv1 "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
	"github.com/axisml-io/axisml/test/e2e"
	"github.com/axisml-io/axisml/test/testutil"
)

// TestMLService_NativeDeployment exercises the (native, deployment) handler:
// create Tenant + MLService running nginx, wait for the underlying
// Deployment to become Available and MLService.Status.Phase to reach Ready.
func TestMLService_NativeDeployment(t *testing.T) {
	ctx, cancel := setup(t)
	defer cancel()

	const (
		tenantName = "e2e-mlsvc-tenant"
		tenantNs   = "e2e-mlsvc-tenant-ns"
		svcName    = "hello-nginx"
	)

	tenant := &tenantv1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{tenantv1.LabelTenantID: "uuid-e2e-mlsvc"},
		},
		Spec: tenantv1.TenantSpec{
			Namespace: tenantv1.NamespaceSpec{Name: tenantNs},
			Quotas: []tenantv1.QuotaSpec{
				{
					Pool: "default",
					Name: "default",
					Min:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")},
					Max:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))
	t.Cleanup(func() {
		// Fresh context — the test's parent ctx may already be cancelled by t.Fatal.
		cleanCtx, cancelClean := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancelClean()
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&mlsvcv1.MLService{ObjectMeta: metav1.ObjectMeta{Namespace: tenantNs, Name: svcName}}, 2*time.Minute)
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&tenantv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: tenantName}}, time.Minute)
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNs}}, 3*time.Minute)
	})

	// Wait for the tenant namespace to exist, then submit the MLService
	// without waiting for Tenant.Status.Phase == Active or the ElasticQuota.
	// The mlservice-operator reconciler retries on missing-quota errors, so
	// any race resolves on the next reconcile rather than failing the test.
	var ns corev1.Namespace
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Name: tenantNs}, &ns, 90*time.Second)

	quotaName := e2e.ElasticQuotaName(tenantName, "default", "default")
	svc := &mlsvcv1.MLService{
		ObjectMeta: metav1.ObjectMeta{Namespace: tenantNs, Name: svcName},
		Spec: mlsvcv1.MLServiceSpec{
			Backend:    mlsvcv1.Backend{Name: "native", Engine: "deployment"},
			Scheduling: mlsvcv1.Scheduling{Quota: quotaName},
			ModelRef:   mlsvcv1.ModelRef{Name: "placeholder", Version: "v0"},
			Roles: []mlsvcv1.RoleSpec{{
				Name:     mlsvcv1.DefaultRoleName,
				Replicas: 1,
				Template: mlsvcv1.PodTemplate{
					Image: "nginx:1.27",
					Ports: []mlsvcv1.PodPort{{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP}},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, svc))

	testutil.Eventually(t, 4*time.Minute, 2*time.Second, func() error {
		var got mlsvcv1.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: svcName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != mlsvcv1.PhaseReady {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})
}
