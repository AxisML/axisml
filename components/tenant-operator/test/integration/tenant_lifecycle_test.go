//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/tenant-operator/internal/reconcile"

	"github.com/axisml/axisml/test/testutil"
)

// TestTenant_SuspendUnsuspend toggles spec.suspended and verifies the
// dispatcher routes through the suspend short-circuit (phase=Suspended,
// underlying resources untouched), then back to Active when un-suspended.
// Per design §5, suspension does NOT delete Namespace/ElasticQuota — those
// stay so existing workloads keep running while submission is gated.
func TestTenant_SuspendUnsuspend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName = "team-susp"
		tenantNs   = "team-susp-ns"
	)
	t.Cleanup(func() { cleanupTenant(t, c, tenantName, tenantNs) })

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-susp"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
			Quotas: []axisml.QuotaSpec{{
				Pool: "default", Name: "default",
				Min: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
				Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	// Reach Active first.
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

	// Suspend.
	var fresh axisml.Tenant
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &fresh))
	patch := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Suspended = true
	require.NoError(t, c.Patch(ctx, &fresh, patch))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.TenantPhaseSuspended {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		hasCond := false
		for _, cond := range got.Status.Conditions {
			if cond.Type == axisml.ConditionSuspended && cond.Status == metav1.ConditionTrue {
				hasCond = true
				break
			}
		}
		if !hasCond {
			return fmt.Errorf("Suspended condition missing")
		}
		return nil
	})

	// Underlying resources stay intact while suspended.
	eqName := reconcile.ElasticQuotaName(tenantName, "default", "default")
	var eq schedv1alpha1.ElasticQuota
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: eqName}, &eq),
		"ElasticQuota must NOT be deleted while suspended")

	// Un-suspend.
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &fresh))
	patch = client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Suspended = false
	require.NoError(t, c.Patch(ctx, &fresh, patch))

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
}

// TestTenant_MissingSourceSecret verifies that referring to a non-existent
// source Secret leaves the tenant in a transient (non-Active, non-Failed)
// phase with a descriptive status.message; then once the source appears
// the next reconcile converges to Active. Validates the design choice to
// surface "in-progress" via empty/previous phase rather than crashing
// the reconcile loop.
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
		// Best-effort source secret cleanup; ignore not-found.
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

	// Tenant must NOT reach Active while the source secret is missing.
	// The per-item status carries the "source secret not found" message;
	// the top-level Status.Message is the higher-level "init resources not
	// ready" because the reconcile keeps making progress and aggregates
	// per-item failures into the per-item status.
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

	// Now create the source secret. The reconciler doesn't watch source
	// resources (they're read via the uncached APIReader), so creating
	// the secret alone won't requeue the tenant — bump a benign annotation
	// to nudge controller-runtime, which is what Compute would do in
	// production by re-PATCHing the Tenant CR after the upstream condition
	// changes.
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

	// And the per-tenant pull secret materialised.
	pullName := reconcile.PerTenantResourceName(tenantName, pullSecName)
	var got corev1.Secret
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: pullName}, &got))
}

// TestTenant_QuotaUpdate verifies that mutating an existing Tenant's
// quota.max value propagates to the underlying ElasticQuota — exercising
// the idempotent reconcile path on a CR mutation, not just create.
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

	// Bump max from 2 to 8.
	var fresh axisml.Tenant
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &fresh))
	patch := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Quotas[0].Max[corev1.ResourceCPU] = resource.MustParse("8")
	require.NoError(t, c.Patch(ctx, &fresh, patch))

	// ElasticQuota.spec.max.cpu must reflect the new value.
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

	// Tenant remains Active after the mutation.
	var after axisml.Tenant
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantName}, &after))
	require.Equal(t, axisml.TenantPhaseActive, after.Status.Phase)
}

// (cleanupTenant lives in tenant_reconciler_test.go and is shared.)
