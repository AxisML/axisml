//go:build envtest

// Package envtest_test runs the mljob-operator dispatcher + handlers against
// an embedded apiserver+etcd via controller-runtime's envtest. Hermetic.
//
// CRDs loaded:
//
//   - MLJob: deploy/helm/axisml-system/crds/mljob-crd.yaml
//   - PodGroup (koordinator-vendored scheduler-plugins, scheduling.sigs.k8s.io): test/crds/external/scheduler-plugins-podgroup.yaml
//
// envtest has no kubelet, so the underlying batch/v1.Job's Pods never
// progress on their own. Tests simulate completion by directly patching
// Job.Status (see mljob_native_job_test.go).
package envtest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	axisv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/mljob-operator/internal/dispatcher"
	"github.com/axisml/axisml/components/operators/mljob-operator/internal/handlers/nativejob"
	"github.com/axisml/axisml/components/operators/mljob-operator/internal/handlers/nativepodgroup"

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
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(axisv1alpha1.AddToScheme(testScheme))
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

	registry := dispatcher.NewRegistry()
	registry.Register(nativejob.New())
	registry.Register(nativepodgroup.New())

	r := &dispatcher.MLJobReconciler{
		Client:   mgr.GetClient(),
		Registry: registry,
	}
	if err := r.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup MLJob reconciler: %w", err)
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
