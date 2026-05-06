//go:build integration

// Package integration_test runs the axisml-tenant-operator (Tenant
// reconciler) against an embedded apiserver+etcd via controller-runtime's
// envtest.
//
// CRDs loaded:
//
//   - Tenant: deploy/helm/axisml-system/crds/tenant-crd.yaml
//   - ElasticQuota: test/crds/external/koordinator-elasticquota.yaml
package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	tenantcontroller "github.com/axisml/axisml/components/tenant-operator/internal/controller"
	"github.com/axisml/axisml/components/tenant-operator/internal/setup"
	tenantvalidate "github.com/axisml/axisml/components/tenant-operator/internal/validate"

	"github.com/axisml/axisml/test/testutil"
)

// SourceNamespace holds source Secret/ConfigMap referenced from Tenant
// spec.initResources.*.sourceSecretRef / sourceConfigMapRef.
const SourceNamespace = "axisml-system"

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
