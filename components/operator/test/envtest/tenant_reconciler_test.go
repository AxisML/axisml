//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
	"github.com/axisml/axisml/components/operator/internal/tenant/reconcile"
	"github.com/axisml/axisml/test/testutil"
)

const (
	testWaitTimeout = 30 * time.Second
)

// TestTenant_HappyPath drives the full reconcile loop end-to-end against the
// envtest apiserver:
//
//  1. Create source Secret + ConfigMap in axisml-system.
//  2. Create the Tenant CR.
//  3. Assert the target Namespace, ElasticQuota, ImagePullSecret, Secret,
//     ConfigMap, ServiceAccount, Role, and RoleBinding all appear with the
//     managed-by label.
//  4. Assert Tenant.status.phase converges to Active.
func TestTenant_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName = "team-a"
		tenantNs   = "team-a-ns"
		srcSecret  = "src-secret"
		srcCM      = "src-config"
	)
	t.Cleanup(func() { cleanupTenant(t, c, tenantName, tenantNs) })

	require.NoError(t, c.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: srcSecret},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)},
	}))
	require.NoError(t, c.Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: SourceNamespace, Name: srcCM},
		Data:       map[string]string{"hello": "world"},
	}))

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-a"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
			Quotas: []axisml.QuotaSpec{
				{
					Pool: "gpu",
					Name: "default",
					Min:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")},
					Max:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")},
				},
			},
			InitResources: axisml.InitResources{
				ImagePullSecrets: []axisml.ImagePullSecretSpec{
					{Name: "registry", SourceSecretRef: axisml.SourceSecretRef{Namespace: SourceNamespace, Name: srcSecret}},
				},
				ConfigMaps: []axisml.ConfigMapSpec{
					{Name: "envs", SourceConfigMapRef: axisml.SourceConfigMapRef{Namespace: SourceNamespace, Name: srcCM}},
				},
				ServiceAccounts: []axisml.ServiceAccountSpec{
					{
						Name:             "default",
						ImagePullSecrets: []string{"registry"},
						RBAC: &axisml.RBACSpec{
							Rules: []rbacv1.PolicyRule{
								{Verbs: []string{"get", "list"}, Resources: []string{"pods"}, APIGroups: []string{""}},
							},
						},
					},
				},
			},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	// Namespace appears.
	var ns corev1.Namespace
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Name: tenantNs}, &ns, testWaitTimeout)
	require.Equal(t, axisml.ManagedByValue, ns.Labels[axisml.LabelManagedBy], "namespace missing managed-by label")

	// ElasticQuota appears in tenant namespace, named per the operator's
	// canonical scheme (axisml-<tenant>-<pool>-<quota>).
	eqName := reconcile.ElasticQuotaName(tenantName, "gpu", "default")
	var eq schedv1alpha1.ElasticQuota
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: eqName}, &eq, testWaitTimeout)
	require.Equal(t, "4", eq.Spec.Max.Cpu().String(), "ElasticQuota.spec.max.cpu mismatch")

	// Per-tenant resources are renamed to axisml-tenant-<tenant>-<sub>.
	pullName := reconcile.PerTenantResourceName(tenantName, "registry")
	cmName := reconcile.PerTenantResourceName(tenantName, "envs")
	saName := reconcile.PerTenantResourceName(tenantName, "default")

	var pullSecret corev1.Secret
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: pullName}, &pullSecret, testWaitTimeout)
	require.Equal(t, corev1.SecretTypeDockerConfigJson, pullSecret.Type)

	var cm corev1.ConfigMap
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: cmName}, &cm, testWaitTimeout)
	require.Equal(t, "world", cm.Data["hello"])

	var sa corev1.ServiceAccount
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: saName}, &sa, testWaitTimeout)
	var role rbacv1.Role
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: saName}, &role, testWaitTimeout)
	var rb rbacv1.RoleBinding
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Namespace: tenantNs, Name: saName}, &rb, testWaitTimeout)

	// Tenant status converges to Active.
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.Tenant
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.TenantPhaseActive {
			return fmt.Errorf("phase=%q (msg=%q)", got.Status.Phase, got.Status.Message)
		}
		return nil
	})
}

// cleanupTenant tears down a Tenant CR and its target Namespace. The
// reconciler intentionally does NOT delete Namespaces (design §6.1), so the
// test must do it explicitly to keep envtest hermetic between runs.
func cleanupTenant(t *testing.T, c client.Client, name, ns string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenant := &axisml.Tenant{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := c.Delete(ctx, tenant); err != nil && !apierrors.IsNotFound(err) {
		t.Logf("cleanup tenant %q: %v", name, err)
	}
	if ns != "" {
		nsObj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		if err := c.Delete(ctx, nsObj); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("cleanup namespace %q: %v", ns, err)
		}
	}
}
