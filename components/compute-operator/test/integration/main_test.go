//go:build integration

// Package integration_test runs the axisml-compute-operator (MLJob,
// MLService reconcilers) against an embedded apiserver+etcd via
// controller-runtime's envtest.
//
// CRDs loaded:
//
//   - MLJob: deploy/helm/axisml-system/crds/mljob-crd.yaml
//   - MLService: deploy/helm/axisml-system/crds/mlservice-crd.yaml
//   - HTTPRoute: test/crds/external/gateway-api-httproute.yaml
//
// envtest has no kubelet, so workload controllers (Job, Deployment) don't
// progress on their own. Tests simulate progress by directly patching
// status (see mljob_native_job_test.go, mlservice_native_deployment_test.go).
package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	mljobdispatcher "github.com/axisml/axisml/components/compute-operator/internal/mljob/dispatcher"
	"github.com/axisml/axisml/components/compute-operator/internal/mljob/handlers/nativejob"
	mlservicedispatcher "github.com/axisml/axisml/components/compute-operator/internal/mlservice/dispatcher"
	mlservicehandler "github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler"
	mltrafficpolicydispatcher "github.com/axisml/axisml/components/compute-operator/internal/mltrafficpolicy/dispatcher"
	mltrafficpolicyhandler "github.com/axisml/axisml/components/compute-operator/internal/mltrafficpolicy/handler"
	"github.com/axisml/axisml/components/compute-operator/internal/setup"

	// Side-effect imports: register the native MLService handlers.
	_ "github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler/nativedeployment"
	_ "github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler/nativestatefulset"

	// Side-effect import: register the native MLTrafficPolicy handler.
	_ "github.com/axisml/axisml/components/compute-operator/internal/mltrafficpolicy/handler/nativehttproute"

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
	setup.AddToScheme(testScheme)
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

	mljobRegistry := mljobdispatcher.NewRegistry()
	mljobRegistry.Register(nativejob.New())
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

	mltpHandlersByKey, mltpAll, err := mltrafficpolicyhandler.Build(mgr)
	if err != nil {
		return fmt.Errorf("MLTrafficPolicy handler build: %w", err)
	}
	if err := mltrafficpolicydispatcher.NewReconciler(mgr, mltpHandlersByKey).SetupWithManager(mgr, mltpAll); err != nil {
		return fmt.Errorf("setup MLTrafficPolicy reconciler: %w", err)
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
