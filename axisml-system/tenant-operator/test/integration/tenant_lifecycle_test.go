//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	schedv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/scheduling/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/axisml-system/tenant-operator/internal/reconcile"

	"github.com/axisml/axisml/test/testutil"
)

// TestTenant_SubmitCRHappyPath is the primary acceptance test for the new
// tenant-operator design: submit a Tenant CR, wait for the operator to
// converge to Active, then verify the underlying Namespace, ElasticQuota,
// and per-tenant ImagePullSecret all landed as designed (tenant-operator §4).
func TestTenant_SubmitCRHappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName = "team-happy"
		tenantNs   = "team-happy-ns"
		pullName   = "registry"
	)
	t.Cleanup(func() {
		cleanupTenant(t, c, tenantName, tenantNs)
		_ = c.Delete(context.Background(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: "happy-source"},
		})
	})

	// Seed the source ImagePullSecret (read by tenant-operator via APIReader).
	require.NoError(t, c.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: "happy-source"},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)},
	}))

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-happy"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{
				Name:   tenantNs,
				Labels: map[string]string{"team": "happy"},
			},
			Quotas: []axisml.QuotaSpec{{
				Pool: "default", Name: "default",
				Min: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
				Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")},
			}},
			InitResources: axisml.InitResources{
				ImagePullSecrets: []axisml.ImagePullSecretSpec{{
					Name:            pullName,
					SourceSecretRef: axisml.SourceSecretRef{Namespace: SourceNamespace, Name: "happy-source"},
				}},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.TenantPhaseActive {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})

	// Namespace exists and carries managed-by label.
	var ns corev1.Namespace
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantNs}, &ns))
	require.Equal(t, axisml.ManagedByValue, ns.Labels[axisml.LabelManagedBy])
	require.Equal(t, "happy", ns.Labels["team"])

	// ElasticQuota exists with the requested max.
	eqName := reconcile.ElasticQuotaName(tenantName, "default", "default")
	var eq schedv1alpha1.ElasticQuota
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: eqName}, &eq))
	require.Equal(t, "4", eq.Spec.Max.Cpu().String())

	// Per-tenant ImagePullSecret materialised.
	perTenant := reconcile.PerTenantResourceName(tenantName, pullName)
	var pulled corev1.Secret
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: perTenant}, &pulled))
}

// TestTenant_MissingSourceSecret verifies that referring to a non-existent
// source Secret leaves the tenant in a transient (non-Active, non-Failed)
// phase with a descriptive status.message; once the source appears the next
// reconcile converges to Active.
func TestTenant_MissingSourceSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName  = "team-missing"
		tenantNs    = "team-missing-ns"
		sourceName  = "missing-on-purpose"
		pullSecName = "registry"
	)
	t.Cleanup(func() {
		cleanupTenant(t, c, tenantName, tenantNs)
		_ = c.Delete(context.Background(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: sourceName},
		})
	})

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-missing"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
			InitResources: axisml.InitResources{
				ImagePullSecrets: []axisml.ImagePullSecretSpec{{
					Name:            pullSecName,
					SourceSecretRef: axisml.SourceSecretRef{Namespace: SourceNamespace, Name: sourceName},
				}},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase == axisml.TenantPhaseActive {
			return fmt.Errorf("tenant unexpectedly Active despite missing source secret")
		}
		if len(got.Status.InitResources.ImagePullSecrets) == 0 {
			return fmt.Errorf("InitResources.ImagePullSecrets status not yet populated")
		}
		item := got.Status.InitResources.ImagePullSecrets[0]
		if item.Ready {
			return fmt.Errorf("imagePullSecret %q unexpectedly Ready", item.Name)
		}
		if !strings.Contains(strings.ToLower(item.Message), "not found") {
			return fmt.Errorf("per-item message=%q does not mention not-found", item.Message)
		}
		return nil
	})

	// Create the source secret, then nudge the Tenant to trigger reconcile.
	require.NoError(t, c.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: sourceName},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)},
	}))

	var nudged axisml.Tenant
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &nudged))
	patch := client.MergeFrom(nudged.DeepCopy())
	if nudged.Annotations == nil {
		nudged.Annotations = map[string]string{}
	}
	nudged.Annotations["axisml.io/test-nudge"] = "1"
	require.NoError(t, c.Patch(ctx, &nudged, patch))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.TenantPhaseActive {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})

	pullName := reconcile.PerTenantResourceName(tenantName, pullSecName)
	var got corev1.Secret
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: pullName}, &got))
}

// TestTenant_QuotaUpdate verifies that mutating an existing Tenant's
// quota.max value propagates to the underlying ElasticQuota.
func TestTenant_QuotaUpdate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName = "team-quota"
		tenantNs   = "team-quota-ns"
	)
	t.Cleanup(func() { cleanupTenant(t, c, tenantName, tenantNs) })

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-quota"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
			Quotas: []axisml.QuotaSpec{{
				Pool: "default", Name: "default",
				Min: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
				Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	eqName := reconcile.ElasticQuotaName(tenantName, "default", "default")
	var eq schedv1alpha1.ElasticQuota
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: tenantNs, Name: eqName}, &eq, testWaitTimeout)
	require.Equal(t, "2", eq.Spec.Max.Cpu().String(), "initial max.cpu")

	var fresh axisml.Tenant
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &fresh))
	patch := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Quotas[0].Max[corev1.ResourceCPU] = resource.MustParse("8")
	require.NoError(t, c.Patch(ctx, &fresh, patch))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got schedv1alpha1.ElasticQuota
		if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: eqName}, &got); err != nil {
			return err
		}
		if got.Spec.Max.Cpu().String() != "8" {
			return fmt.Errorf("ElasticQuota.spec.max.cpu=%s, want 8", got.Spec.Max.Cpu().String())
		}
		return nil
	})

	var after axisml.Tenant
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &after))
	require.Equal(t, axisml.TenantPhaseActive, after.Status.Phase)
}
