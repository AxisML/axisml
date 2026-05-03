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

	mljobv1 "axisml.io/operators/mljob/api/v1alpha1"
	tenantv1 "github.com/axisml-io/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml-io/axisml/test/e2e"
	"github.com/axisml-io/axisml/test/testutil"
)

// TestMLJob_NativeJob runs a full submit→schedule→succeed loop on the real
// minikube cluster: create Tenant + MLJob{native, job} executing
// `busybox` echo, then assert MLJob.Status.Phase == Succeeded once the
// underlying batch/v1 Job's Pod completes for real.
func TestMLJob_NativeJob(t *testing.T) {
	ctx, cancel := setup(t)
	defer cancel()

	const (
		tenantName = "e2e-mljob-tenant"
		tenantNs   = "e2e-mljob-tenant-ns"
		jobName    = "hello"
	)

	tenant := &tenantv1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{tenantv1.LabelTenantID: "uuid-e2e-mljob"},
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
			&mljobv1.MLJob{ObjectMeta: metav1.ObjectMeta{Namespace: tenantNs, Name: jobName}}, 2*time.Minute)
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&tenantv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: tenantName}}, time.Minute)
		e2e.DeleteAndWaitGone(t, cleanCtx, c,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNs}}, 3*time.Minute)
	})

	// Wait for the tenant namespace to exist, then submit the MLJob without
	// waiting for Tenant.Status.Phase == Active or for the ElasticQuota to be
	// created. The mljob-operator reconciler retries on missing-quota errors,
	// so any race resolves on the next reconcile rather than failing the test.
	// Trading a few seconds of "Pending" status for ~30s of unnecessary
	// pre-flight waits.
	var ns corev1.Namespace
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Name: tenantNs}, &ns, 90*time.Second)

	quotaName := e2e.ElasticQuotaName(tenantName, "default", "default")
	mljob := &mljobv1.MLJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: tenantNs, Name: jobName},
		Spec: mljobv1.MLJobSpec{
			Backend:    mljobv1.BackendSpec{Name: "native", Engine: "job"},
			Scheduling: mljobv1.SchedulingSpec{Quota: quotaName},
			Roles: []mljobv1.RoleSpec{{
				Name:          "trainer",
				Replicas:      1,
				RestartPolicy: corev1.RestartPolicyNever,
				Template: mljobv1.PodTemplateSubset{
					Image:   "busybox:latest",
					Command: []string{"sh", "-c", "echo hello"},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mljob))

	testutil.Eventually(t, 4*time.Minute, 2*time.Second, func() error {
		var got mljobv1.MLJob
		if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: jobName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != mljobv1.PhaseSucceeded {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})
}
