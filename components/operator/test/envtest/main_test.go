//go:build envtest

// Package envtest_test runs the merged axisml-operator (Tenant, MLJob,
// MLService reconcilers) against an embedded apiserver+etcd via
// controller-runtime's envtest. Hermetic — no minikube, no in-cluster
// operator, no shared state between runs.
//
// Prerequisites (handled by `make axisml-operator-envtest`):
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
//   - MLJob: deploy/helm/axisml-system/crds/mljob-crd.yaml
//   - MLService: deploy/helm/axisml-system/crds/mlservice-crd.yaml
//   - ElasticQuota: test/crds/external/koordinator-elasticquota.yaml
//   - PodGroup: test/crds/external/scheduler-plugins-podgroup.yaml
//   - HTTPRoute: test/crds/external/gateway-api-httproute.yaml
//
// envtest has no kubelet, so workload controllers (Job, Deployment) don't
// progress on their own. Tests simulate progress by directly patching
// status (see mljob_native_job_test.go, mlservice_native_deployment_test.go).
package envtest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
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
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
	mljobdispatcher "github.com/axisml/axisml/components/operator/internal/mljob/dispatcher"
	"github.com/axisml/axisml/components/operator/internal/mljob/handlers/nativejob"
	"github.com/axisml/axisml/components/operator/internal/mljob/handlers/nativepodgroup"
	mlservicedispatcher "github.com/axisml/axisml/components/operator/internal/mlservice/dispatcher"
	mlservicehandler "github.com/axisml/axisml/components/operator/internal/mlservice/handler"
	tenantcontroller "github.com/axisml/axisml/components/operator/internal/tenant/controller"
	tenantvalidate "github.com/axisml/axisml/components/operator/internal/tenant/validate"

	// Side-effect import: registers the (native, deployment) MLService handler.
	_ "github.com/axisml/axisml/components/operator/internal/mlservice/handler/nativedeployment"

	"github.com/axisml/axisml/test/testutil"
)

// SourceNamespace holds source Secret/ConfigMap referenced from Tenant
// spec.initResources.*.sourceSecretRef / sourceConfigMapRef. Created in
// TestMain so tests can pre-populate fixture data without re-creating it.
const SourceNamespace = "axisml-system"

var (
	testScheme = runtime.NewScheme()

	testEnv *testutil.EnvtestHandle
	testCfg *rest.Config
	mgrStop context.CancelFunc
	mgrWg   sync.WaitGroup
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(gwapiv1.Install(testScheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(mljobv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(mlservicev1alpha1.AddToScheme(testScheme))

	mlservicehandler.RegisterStubs()
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

// bootstrapManager wires a single manager with the Tenant, MLJob, and
// MLService reconcilers and runs it in a goroutine for the duration of the
// test process. The cache is left unrestricted (unlike the production
// manager which scopes Tenant child resources by managed-by label) — envtest
// is a fresh apiserver scoped to the test process, so we want tests to see
// every object the reconcilers create.
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

	if err := (&tenantcontroller.TenantReconciler{
		Client:    mgr.GetClient(),
		APIReader: mgr.GetAPIReader(),
		Scheme:    mgr.GetScheme(),
		ValidateOpts: tenantvalidate.Options{
			NamespaceDenylist: tenantvalidate.DefaultNamespaceDenylist(),
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup tenant reconciler: %w", err)
	}

	mljobRegistry := mljobdispatcher.NewRegistry()
	mljobRegistry.Register(nativejob.New())
	mljobRegistry.Register(nativepodgroup.New())
	if err := (&mljobdispatcher.MLJobReconciler{
		Client:   mgr.GetClient(),
		Registry: mljobRegistry,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup MLJob reconciler: %w", err)
	}

	mlserviceHandlersByKey, mlserviceAll, err := mlservicehandler.Build(mgr)
	if err != nil {
		return fmt.Errorf("MLService handler build: %w", err)
	}
	if err := mlservicedispatcher.NewReconciler(mgr, mlserviceHandlersByKey).SetupWithManager(mgr, mlserviceAll); err != nil {
		return fmt.Errorf("setup MLService reconciler: %w", err)
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
