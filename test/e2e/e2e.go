// Package e2e exposes shared setup helpers used by the per-domain L2 e2e
// suites under test/e2e/operators, test/e2e/compute, etc. The suites are
// gated by `//go:build e2e` and run against a real minikube cluster brought
// up by `make e2e-test` from the repo root.
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	mljobv1 "axisml.io/operators/mljob/api/v1alpha1"
	tenantv1 "github.com/axisml-io/axisml/components/operators/tenant-operator/api/v1alpha1"
	mlsvcv1 "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/axisml-io/axisml/test/testutil"
)

// SystemNamespace is where the axisml control plane (operators + future
// compute / artifacts / platform) is helm-installed by `make helm-install`.
const SystemNamespace = "axisml-system"

// HelmReleaseEnv is the env var the e2e suites read for the helm release
// name. Defaults to "axisml" (the value used by the top-level Makefile).
const HelmReleaseEnv = "HELM_SYSTEM_RELEASE"

// Scheme returns a scheme registered with all AxisML CRDs and the standard
// k8s types the e2e tests inspect (Deployment, Job, RBAC, etc.). Caller
// should NOT mutate the returned scheme.
func Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(tenantv1.AddToScheme(scheme))
	utilruntime.Must(mljobv1.AddToScheme(scheme))
	utilruntime.Must(mlsvcv1.AddToScheme(scheme))
	return scheme
}

// SetupOrSkip resolves a kubeconfig + client and verifies the axisml-system
// namespace is reachable. If the cluster is unreachable, it calls t.Skip
// rather than t.Fatal so an inadvertent `go test ./...` from a workstation
// without a cluster doesn't fail loudly.
func SetupOrSkip(t *testing.T) (*rest.Config, client.Client) {
	t.Helper()
	scheme := Scheme()
	cfg, c := testutil.KubeconfigClient(t, scheme)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ns corev1.Namespace
	if err := c.Get(ctx, types.NamespacedName{Name: SystemNamespace}, &ns); err != nil {
		if apierrors.IsNotFound(err) {
			t.Skipf("e2e: %q namespace missing — run `make e2e-test` from repo root", SystemNamespace)
		}
		t.Skipf("e2e: cluster unreachable: %v", err)
	}
	return cfg, c
}

// HelmRelease returns the helm release prefix used by axisml-system templates
// (default "axisml"). Resolved from $HELM_SYSTEM_RELEASE.
func HelmRelease() string {
	if v := os.Getenv(HelmReleaseEnv); v != "" {
		return v
	}
	return "axisml"
}

// ElasticQuotaName mirrors tenant-operator/internal/reconcile.ElasticQuotaName
// for use in cross-module e2e tests, which can't import internal/. Naming is
// part of the operator's external contract (design §6.2): axisml-<tenant>-<pool>-<quota>.
func ElasticQuotaName(tenant, pool, quota string) string {
	return fmt.Sprintf("axisml-%s-%s-%s", tenant, pool, quota)
}

// WaitDeploymentAvailable polls until the named Deployment has at least
// minReplicas available, or times out.
func WaitDeploymentAvailable(t *testing.T, ctx context.Context, c client.Client, ns, name string, minReplicas int32, timeout time.Duration) {
	t.Helper()
	testutil.Eventually(t, timeout, time.Second, func() error {
		var dep appsv1.Deployment
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &dep); err != nil {
			return err
		}
		if dep.Status.AvailableReplicas < minReplicas {
			return fmt.Errorf("Deployment %s/%s availableReplicas=%d (want >=%d)",
				ns, name, dep.Status.AvailableReplicas, minReplicas)
		}
		return nil
	})
}

// DeleteAndWaitGone issues a foreground delete and waits for the object to
// disappear. Used in test cleanup hooks.
func DeleteAndWaitGone(t *testing.T, ctx context.Context, c client.Client, obj client.Object, timeout time.Duration) {
	t.Helper()
	key := client.ObjectKeyFromObject(obj)
	if err := c.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		t.Logf("delete %T %s: %v", obj, key, err)
		return
	}
	testutil.EventuallyGone(t, ctx, c, key, obj, timeout)
}
