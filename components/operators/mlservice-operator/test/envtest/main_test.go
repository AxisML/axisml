//go:build envtest

// Package envtest_test runs the mlservice-operator dispatcher + handler
// against an embedded apiserver+etcd via controller-runtime's envtest.
//
// CRDs loaded:
//
//   - MLService: deploy/helm/axisml-system/crds/mlservice-crd.yaml
//   - HTTPRoute (gateway.networking.k8s.io/v1): test/crds/external/gateway-api-httproute.yaml
//
// HTTPRoute is needed even for tests that don't enable spec.route because
// the dispatcher's controller-runtime SetupWithManager watches HTTPRoute
// to react to route-enabled changes; without the CRD the watch loop logs
// "no matches for kind HTTPRoute" forever and the test times out.
//
// Envoy Gateway SecurityPolicy / BackendTrafficPolicy are only needed when
// route.auth or route.rateLimit is set — vendor those CRDs alongside this
// one when route-enabled tests are introduced.
//
// envtest has no kubelet, so the underlying Deployment's ReplicaSet/Pods
// don't progress on their own. Tests simulate readiness by directly
// patching Deployment.Status (see mlservice_native_deployment_test.go).
package envtest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/mlservice-operator/internal/dispatcher"
	"github.com/axisml/axisml/components/operators/mlservice-operator/internal/handler"

	// Register the (native, deployment) handler into the package-global registry.
	_ "github.com/axisml/axisml/components/operators/mlservice-operator/internal/handler/nativedeployment"

	"github.com/axisml/axisml/test/testutil"
)

var (
	testScheme = runtime.NewScheme()

	testEnv *testutil.EnvtestHandle
	testCfg *rest.Config
	mgrStop context.CancelFunc
	mgrWg   sync.WaitGroup
)

func init() {
	// clientgoscheme covers core/v1, apps/v1, batch/v1, rbac/v1, etc.
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(gwapiv1.Install(testScheme))
	utilruntime.Must(axisml.AddToScheme(testScheme))

	handler.RegisterStubs()
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

	code := m.Run()

	mgrStop()
	mgrWg.Wait()
	if err := testEnv.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "envtest stop: %v\n", err)
	}
	os.Exit(code)
}

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

	handlersByKey, allHandlers, err := handler.Build(mgr)
	if err != nil {
		return fmt.Errorf("handler build: %w", err)
	}

	r := dispatcher.NewReconciler(mgr, handlersByKey)
	if err := r.SetupWithManager(mgr, allHandlers); err != nil {
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
