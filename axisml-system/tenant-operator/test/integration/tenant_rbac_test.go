//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/axisml-system/tenant-operator/internal/reconcile"
	"github.com/axisml/axisml/test/testutil"
)

// TestTenant_RBACClusterRoleBinding exercises §6.6's "ClusterRole binding"
// shape: when spec.serviceAccounts[].rbac.roleRef.kind == "ClusterRole", the
// reconciler must NOT create a per-tenant Role and the RoleBinding must point
// at the named ClusterRole.
func TestTenant_RBACClusterRoleBinding(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName  = "team-cr"
		tenantNs    = "team-cr-ns"
		clusterRole = "axisml-platform-default"
	)
	t.Cleanup(func() {
		cleanupTenant(t, c, tenantName, tenantNs)
		_ = c.Delete(context.Background(), &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: clusterRole},
		})
	})

	// Pre-seed the ClusterRole the binding will reference. The reconciler
	// itself only creates the binding; the ClusterRole is platform-supplied.
	require.NoError(t, c.Create(ctx, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: clusterRole},
		Rules: []rbacv1.PolicyRule{
			{Verbs: []string{"get"}, APIGroups: []string{""}, Resources: []string{"pods"}},
		},
	}))

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-cr"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
			InitResources: axisml.InitResources{
				ServiceAccounts: []axisml.ServiceAccountSpec{{
					Name: "runner",
					RBAC: &axisml.RBACSpec{
						RoleRef: &axisml.RBACRoleRef{Kind: "ClusterRole", Name: clusterRole},
					},
				}},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	saName := reconcile.PerTenantResourceName(tenantName, "runner")

	// ServiceAccount appears.
	var sa corev1.ServiceAccount
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: saName}, &sa, testWaitTimeout)

	// RoleBinding points at the cluster role with kind=ClusterRole.
	var rb rbacv1.RoleBinding
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		if err := c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: saName}, &rb); err != nil {
			return err
		}
		if rb.RoleRef.Kind != "ClusterRole" {
			return fmt.Errorf("RoleRef.Kind=%q want ClusterRole", rb.RoleRef.Kind)
		}
		if rb.RoleRef.Name != clusterRole {
			return fmt.Errorf("RoleRef.Name=%q want %q", rb.RoleRef.Name, clusterRole)
		}
		return nil
	})

	// And NO per-tenant Role was created — it would shadow the ClusterRole.
	var role rbacv1.Role
	err := c.Get(ctx, types.NamespacedName{Namespace: tenantNs, Name: saName}, &role)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected no per-tenant Role for ClusterRole binding; got err=%v role=%+v", err, role)
	}
}

// TestTenant_ConfigMapDataDriftCorrected verifies the reconciler's drift-
// correction behaviour for ConfigMaps: changing the source ConfigMap data
// must propagate to the per-tenant copy on the next reconcile.
func TestTenant_ConfigMapDataDriftCorrected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName = "team-drift"
		tenantNs   = "team-drift-ns"
		srcCM      = "src-drift"
	)
	t.Cleanup(func() {
		cleanupTenant(t, c, tenantName, tenantNs)
		_ = c.Delete(context.Background(), &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: srcCM},
		})
	})

	require.NoError(t, c.Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: srcCM},
		Data:       map[string]string{"k": "v1"},
	}))

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-drift"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
			InitResources: axisml.InitResources{
				ConfigMaps: []axisml.ConfigMapSpec{{
					Name:               "envs",
					SourceConfigMapRef: axisml.SourceConfigMapRef{Namespace: SourceNamespace, Name: srcCM},
				}},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	cmName := reconcile.PerTenantResourceName(tenantName, "envs")
	target := types.NamespacedName{Namespace: tenantNs, Name: cmName}

	// Initial reconcile copies "v1".
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got corev1.ConfigMap
		if err := c.Get(ctx, target, &got); err != nil {
			return err
		}
		if got.Data["k"] != "v1" {
			return fmt.Errorf("data=%q want v1", got.Data["k"])
		}
		return nil
	})

	// Mutate the source — controller must detect drift and patch the copy.
	var src corev1.ConfigMap
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: SourceNamespace, Name: srcCM}, &src))
	src.Data["k"] = "v2"
	require.NoError(t, c.Update(ctx, &src))

	// Trigger the reconcile by bumping a no-op annotation on the Tenant
	// (the cached client doesn't see source CMs, so a real-cluster
	// resync would catch this; bumping forces it deterministically here).
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var t1 axisml.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &t1); err != nil {
			return err
		}
		// spec.annotations no longer exists in the new design; nudge
		// metadata.annotations instead to trigger a reconcile.
		if t1.Annotations == nil {
			t1.Annotations = map[string]string{}
		}
		t1.Annotations["bump"] = time.Now().Format(time.RFC3339Nano)
		return c.Update(ctx, &t1)
	})

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got corev1.ConfigMap
		if err := c.Get(ctx, target, &got); err != nil {
			return err
		}
		if got.Data["k"] != "v2" {
			return fmt.Errorf("drift not corrected: data=%q want v2", got.Data["k"])
		}
		return nil
	})
}
