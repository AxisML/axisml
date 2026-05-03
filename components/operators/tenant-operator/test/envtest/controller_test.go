//go:build integration

// Package envtest_test runs the tenant-operator reconciler against an
// existing Kubernetes cluster — typically the local minikube cluster
// created by `make cluster-up`.
//
// Prerequisites:
//   - Cluster reachable via the current kubeconfig context (e.g. `axisml`)
//   - Koordinator's ElasticQuota CRD installed; provided by
//     `make helm-install-infra`. The chart's Tenant CRD is applied by the
//     test itself from `deploy/helm/axisml-system/crds/`.
//
// The in-cluster tenant-operator Deployment must NOT be running — this
// test starts its own manager, and two reconcilers would race over the
// same Tenant CR. Scale it to 0 first if it is deployed:
//
//	kubectl scale -n axisml-system deploy/axisml-tenant-operator --replicas=0
//
// Run via `make test-integration` or `go test -tags integration ./test/envtest/...`.
package envtest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	axisml "github.com/axisml-io/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml-io/axisml/components/operators/tenant-operator/internal/controller"
	"github.com/axisml-io/axisml/components/operators/tenant-operator/internal/validate"
)

const (
	srcNamespace   = "axisml-system"
	tenantName     = "team-a"
	tenantNs       = "team-a-ns"
	tenantUUID     = "11111111-2222-3333-4444-555555555555"
	pollInterval   = 200 * time.Millisecond
	pollTimeout    = 60 * time.Second
	deleteTimeout  = 90 * time.Second
)

func TestTenantHappyPath(t *testing.T) {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(schedv1alpha1.AddToScheme(scheme))
	utilruntime.Must(axisml.AddToScheme(scheme))

	useExistingCluster := true
	repoRoot := repoRootDir(t)
	testEnv := &envtest.Environment{
		UseExistingCluster: &useExistingCluster,
		Scheme:             scheme,
		// Apply the Tenant CRD; ElasticQuota CRD is expected to already be
		// installed on the cluster via the Koordinator sub-chart.
		CRDDirectoryPaths: []string{
			filepath.Join(repoRoot, "deploy", "helm", "axisml-system", "crds"),
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("connect to existing cluster: %v (is `make cluster-up` running and is your kubeconfig context set?)", err)
	}
	t.Cleanup(func() { _ = testEnv.Stop() })

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	if err := (&controller.TenantReconciler{
		Client:       mgr.GetClient(),
		APIReader:    mgr.GetAPIReader(),
		Scheme:       mgr.GetScheme(),
		ValidateOpts: validate.Options{NamespaceDenylist: validate.DefaultNamespaceDenylist()},
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup controller: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Logf("manager stopped: %v", err)
		}
	}()

	c := mgr.GetClient()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatal("cache failed to sync")
	}

	// Drop any leftovers from a previous run before the manager starts
	// reconciling — and again on exit so the cluster is left clean. Use the
	// uncached API reader/writer (mgr.GetClient is fine post-cache-sync, but
	// uncached avoids races with the still-warming-up cache).
	cleanup := func(reason string) {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancelCleanup()
		cleanupTestArtifacts(t, c, cleanupCtx, reason)
	}
	cleanup("pre-test")
	t.Cleanup(func() { cleanup("post-test") })

	// Pre-create the controlled source namespace + source secret/configmap so
	// the init-resource subreconcilers have something to copy. axisml-system
	// usually already exists from `helm install`; mustCreate ignores
	// AlreadyExists so this is safe either way.
	mustCreate(t, ctx, c, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: srcNamespace}})
	mustCreate(t, ctx, c, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: srcNamespace, Name: "registry-source"},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)},
	})
	mustCreate(t, ctx, c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: srcNamespace, Name: "envs-source"},
		Data:       map[string]string{"FOO": "bar"},
	})

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: tenantUUID},
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
					{Name: "registry", SourceSecretRef: axisml.SourceSecretRef{Namespace: srcNamespace, Name: "registry-source"}},
				},
				ConfigMaps: []axisml.ConfigMapSpec{
					{Name: "envs", SourceConfigMapRef: axisml.SourceConfigMapRef{Namespace: srcNamespace, Name: "envs-source"}},
				},
				ServiceAccounts: []axisml.ServiceAccountSpec{
					{
						Name:             "default",
						ImagePullSecrets: []string{"registry"},
						RBAC: &axisml.RBACSpec{
							Rules: []rbacv1.PolicyRule{{
								APIGroups: []string{""},
								Resources: []string{"pods"},
								Verbs:     []string{"get", "list"},
							}},
						},
					},
				},
			},
		},
	}
	mustCreate(t, ctx, c, tenant)

	// Assert: Namespace exists, has managed-by label.
	mustEventually(t, "namespace ready", func() error {
		ns := &corev1.Namespace{}
		if err := c.Get(ctx, types.NamespacedName{Name: tenantNs}, ns); err != nil {
			return err
		}
		if ns.Labels[axisml.LabelManagedBy] != axisml.ManagedByValue {
			return errf("namespace missing managed-by label: %v", ns.Labels)
		}
		return nil
	})

	// Assert: ElasticQuota exists with right spec.
	mustEventually(t, "elasticquota created", func() error {
		eq := &schedv1alpha1.ElasticQuota{}
		if err := c.Get(ctx,
			types.NamespacedName{Namespace: tenantNs, Name: "axisml-team-a-gpu-default"},
			eq); err != nil {
			return err
		}
		want := tenant.Spec.Quotas[0]
		if !equalRL(eq.Spec.Min, want.Min) {
			return errf("eq.spec.min=%v want %v", eq.Spec.Min, want.Min)
		}
		if !equalRL(eq.Spec.Max, want.Max) {
			return errf("eq.spec.max=%v want %v", eq.Spec.Max, want.Max)
		}
		if eq.Labels[axisml.LabelTenantID] != tenantUUID {
			return errf("eq missing tenant-id label: %v", eq.Labels)
		}
		if !hasOwnerUID(eq.OwnerReferences, tenant) {
			return errf("eq missing tenant ownerRef")
		}
		return nil
	})

	// Assert: ImagePullSecret created.
	mustEventually(t, "imagepullsecret created", func() error {
		s := &corev1.Secret{}
		if err := c.Get(ctx,
			types.NamespacedName{Namespace: tenantNs, Name: "axisml-tenant-team-a-registry"},
			s); err != nil {
			return err
		}
		if s.Type != corev1.SecretTypeDockerConfigJson {
			return errf("secret.type=%s want %s", s.Type, corev1.SecretTypeDockerConfigJson)
		}
		if !hasOwnerUID(s.OwnerReferences, tenant) {
			return errf("secret missing tenant ownerRef")
		}
		return nil
	})

	// Assert: ConfigMap created.
	mustEventually(t, "configmap created", func() error {
		cm := &corev1.ConfigMap{}
		if err := c.Get(ctx,
			types.NamespacedName{Namespace: tenantNs, Name: "axisml-tenant-team-a-envs"},
			cm); err != nil {
			return err
		}
		if cm.Data["FOO"] != "bar" {
			return errf("configmap data missing FOO=bar: %v", cm.Data)
		}
		return nil
	})

	// Assert: ServiceAccount + Role + RoleBinding created and wired.
	mustEventually(t, "service account + rbac created", func() error {
		sa := &corev1.ServiceAccount{}
		if err := c.Get(ctx,
			types.NamespacedName{Namespace: tenantNs, Name: "axisml-tenant-team-a-default"},
			sa); err != nil {
			return err
		}
		if len(sa.ImagePullSecrets) != 1 || sa.ImagePullSecrets[0].Name != "axisml-tenant-team-a-registry" {
			return errf("sa.ImagePullSecrets unexpected: %v", sa.ImagePullSecrets)
		}

		role := &rbacv1.Role{}
		if err := c.Get(ctx,
			types.NamespacedName{Namespace: tenantNs, Name: "axisml-tenant-team-a-default"},
			role); err != nil {
			return err
		}
		if len(role.Rules) != 1 {
			return errf("role.Rules len=%d want 1", len(role.Rules))
		}

		rb := &rbacv1.RoleBinding{}
		if err := c.Get(ctx,
			types.NamespacedName{Namespace: tenantNs, Name: "axisml-tenant-team-a-default"},
			rb); err != nil {
			return err
		}
		if rb.RoleRef.Kind != "Role" || rb.RoleRef.Name != "axisml-tenant-team-a-default" {
			return errf("rb.RoleRef unexpected: %+v", rb.RoleRef)
		}
		if len(rb.Subjects) != 1 || rb.Subjects[0].Name != "axisml-tenant-team-a-default" {
			return errf("rb.Subjects unexpected: %+v", rb.Subjects)
		}
		return nil
	})

	// Assert: Tenant.status reaches Active.
	mustEventually(t, "tenant phase=Active", func() error {
		got := &axisml.Tenant{}
		if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.TenantPhaseActive {
			return errf("tenant.status.phase=%q want %q (msg=%q)", got.Status.Phase, axisml.TenantPhaseActive, got.Status.Message)
		}
		return nil
	})
}

// --- helpers ---

func repoRootDir(t *testing.T) string {
	t.Helper()
	// test/envtest/ -> tenant-operator/ -> operators/ -> components/ -> repo
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Join(wd, "..", "..", "..", "..", "..")
}

func mustCreate(t *testing.T, ctx context.Context, c client.Client, obj client.Object) {
	t.Helper()
	if err := c.Create(ctx, obj); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create %T %s/%s: %v", obj, obj.GetNamespace(), obj.GetName(), err)
	}
}

func mustEventually(t *testing.T, label string, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(pollTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		lastErr = fn()
		if lastErr == nil {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("%s: %v", label, lastErr)
}

// cleanupTestArtifacts removes the Tenant CR and per-tenant Namespace from
// the cluster so the next run starts clean. The Tenant has no finalizer and
// per-tenant resources rely on owner-ref GC; the Namespace is shared by
// design (no ownerRef) and must be deleted explicitly.
func cleanupTestArtifacts(t *testing.T, c client.Client, ctx context.Context, reason string) {
	t.Helper()

	tenant := &axisml.Tenant{}
	if err := c.Get(ctx, types.NamespacedName{Name: tenantName}, tenant); err == nil {
		if delErr := c.Delete(ctx, tenant); delErr != nil && !apierrors.IsNotFound(delErr) {
			t.Logf("%s: delete tenant: %v", reason, delErr)
		}
		waitGone(t, ctx, c, &axisml.Tenant{}, types.NamespacedName{Name: tenantName}, reason)
	} else if !apierrors.IsNotFound(err) {
		t.Logf("%s: get tenant: %v", reason, err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, types.NamespacedName{Name: tenantNs}, ns); err == nil {
		if ns.DeletionTimestamp == nil {
			if delErr := c.Delete(ctx, ns); delErr != nil && !apierrors.IsNotFound(delErr) {
				t.Logf("%s: delete namespace: %v", reason, delErr)
			}
		}
		waitGone(t, ctx, c, &corev1.Namespace{}, types.NamespacedName{Name: tenantNs}, reason)
	} else if !apierrors.IsNotFound(err) {
		t.Logf("%s: get namespace: %v", reason, err)
	}
}

func waitGone(t *testing.T, ctx context.Context, c client.Client, obj client.Object, key types.NamespacedName, reason string) {
	t.Helper()
	deadline := time.Now().Add(deleteTimeout)
	for time.Now().Before(deadline) {
		err := c.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return
		}
		if err != nil && ctx.Err() != nil {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Logf("%s: %T %s still present after %s", reason, obj, key, deleteTimeout)
}

func equalRL(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || va.Cmp(vb) != 0 {
			return false
		}
	}
	return true
}

func hasOwnerUID(refs []metav1.OwnerReference, t *axisml.Tenant) bool {
	for _, r := range refs {
		if r.UID == t.UID {
			return true
		}
	}
	return false
}

func errf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
