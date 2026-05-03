//go:build envtest

// Package envtest_test runs the tenant-operator reconciler against an
// embedded apiserver+etcd via controller-runtime's envtest. It is hermetic:
// no minikube, no in-cluster operator, no shared state between runs.
//
// Prerequisites (handled by `make tenant-operator-envtest`):
//
//   - The shared `setup-envtest` binary at test/setup-envtest/setup-envtest
//     (installed by `make setup-envtest` from repo root).
//   - KUBEBUILDER_ASSETS env var pointing at a directory containing
//     `etcd`, `kube-apiserver`, `kubectl`. The Makefile sets it via
//     `setup-envtest use $K8S_VERSION -p path`.
//
// CRDs loaded:
//
//   - Tenant: deploy/helm/axisml-system/crds/tenant-crd.yaml
//   - ElasticQuota (Koordinator-flavored): test/crds/external/koordinator-elasticquota.yaml
package envtest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	axisml "github.com/axisml-io/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml-io/axisml/components/operators/tenant-operator/internal/controller"
	"github.com/axisml-io/axisml/components/operators/tenant-operator/internal/validate"
	"github.com/axisml-io/axisml/test/testutil"
)

// SourceNamespace holds source Secret/ConfigMap referenced from Tenant
// spec.initResources.*.sourceSecretRef / sourceConfigMapRef. Created in
// TestMain so tests can pre-populate fixture data without re-creating it.
const SourceNamespace = "axisml-system"

var (
	testScheme = runtime.NewScheme()

	// Populated by TestMain so individual tests can interact with the
	// envtest apiserver through a shared client.
	testEnv *testutil.EnvtestHandle
	testCfg *rest.Config
	mgrStop context.CancelFunc
	mgrWg   sync.WaitGroup
)

func init() {
	// clientgoscheme covers core/v1, apps/v1, batch/v1, rbac/v1, etc.
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(schedv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(axisml.AddToScheme(testScheme))
}

func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repo root: %v\n", err)
		os.Exit(1)
	}

	testEnv, err = testutil.StartEnvtestE(testutil.EnvtestOptions{
		Scheme: testScheme,
		CRDPaths: []string{
			filepath.Join(repoRoot, "deploy", "helm", "axisml-system", "crds"),
			filepath.Join(repoRoot, "test", "crds", "external"),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start envtest: %v\n", err)
		os.Exit(1)
	}
	testCfg = testEnv.Cfg

	if err := bootstrapManager(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap manager: %v\n", err)
		_ = testEnv.Stop()
		os.Exit(1)
	}

	if err := ensureSourceNamespace(); err != nil {
		fmt.Fprintf(os.Stderr, "create source namespace: %v\n", err)
		mgrStop()
		mgrWg.Wait()
		_ = testEnv.Stop()
		os.Exit(1)
	}

	code := m.Run()

	mgrStop()
	mgrWg.Wait()
	if err := testEnv.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "envtest stop: %v\n", err)
	}
	os.Exit(code)
}

// bootstrapManager wires a manager + the real TenantReconciler against the
// envtest apiserver and runs it in a goroutine for the duration of the test
// process. The cache is left unrestricted (unlike the production manager,
// which restricts most resource caches to the managed-by label) — restricting
// it here is unnecessary because envtest is a fresh apiserver scoped to the
// test process; we want the test to see every object the reconciler creates.
func bootstrapManager() error {
	mgr, err := ctrl.NewManager(testCfg, ctrl.Options{
		Scheme:                 testScheme,
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
		Cache:                  cache.Options{},
	})
	if err != nil {
		return fmt.Errorf("new manager: %w", err)
	}

	if err := (&controller.TenantReconciler{
		Client:    mgr.GetClient(),
		APIReader: mgr.GetAPIReader(),
		Scheme:    mgr.GetScheme(),
		ValidateOpts: validate.Options{
			NamespaceDenylist: validate.DefaultNamespaceDenylist(),
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup tenant reconciler: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	mgrStop = cancel
	mgrWg.Add(1)
	go func() {
		defer mgrWg.Done()
		if err := mgr.Start(ctx); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "manager exited: %v\n", err)
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		return fmt.Errorf("cache sync failed")
	}
	return nil
}

func ensureSourceNamespace() error {
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	if err != nil {
		return err
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: SourceNamespace}}
	if err := c.Create(context.Background(), ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// findRepoRoot walks up from the package directory until it finds the
// "deploy/helm/axisml-system/crds" tree. Used to anchor CRDPaths.
func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; dir != "/" && dir != ""; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "deploy", "helm", "axisml-system", "crds")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("could not locate repo root from %s", cwd)
}
