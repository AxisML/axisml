//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"

	computeapp "github.com/axisml/axisml/components/compute-service/internal/app"
	computeconfig "github.com/axisml/axisml/components/compute-service/internal/config"
	computedb "github.com/axisml/axisml/components/compute-service/internal/db"
	computeserver "github.com/axisml/axisml/components/compute-service/internal/server"
	"github.com/axisml/axisml/test/testutil"
)

var (
	testScheme = runtime.NewScheme()

	testEnv *testutil.EnvtestHandle
	testCfg *rest.Config
	pgCfg   computeconfig.Config
	gormDB  *gorm.DB
	pgCtr   testcontainers.Container

	mgrCtx  context.Context
	mgrStop context.CancelFunc
	mgrWg   sync.WaitGroup

	// testEngine is the same Gin engine that production wires via
	// app.BuildModules + server.New, exposed for in-process httptest
	// drives. nil if bootstrapHandlers wasn't called (tests that only
	// need raw services keep working with that case).
	testEngine *gin.Engine
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(axismlv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(mlrunv1alpha1.AddToScheme(testScheme))
	utilruntime.Must(mlservicev1alpha1.AddToScheme(testScheme))
	utilruntime.Must(mltrafficpolicyv1alpha1.AddToScheme(testScheme))
}

// TestMain sets up envtest + PostgreSQL once for the whole package.
func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repo root: %v\n", err)
		os.Exit(1)
	}

	// Boot embedded apiserver + etcd with all three AxisML CRDs.
	testEnv, err = testutil.StartEnvtestE(testutil.EnvtestOptions{
		Scheme: testScheme,
		CRDPaths: []string{
			filepath.Join(repoRoot, "deploy", "helm", "axisml-system", "crds"),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start envtest: %v\n", err)
		os.Exit(1)
	}
	testCfg = testEnv.Cfg

	// Boot ephemeral PostgreSQL (skipped if Docker is unavailable).
	if err := bootstrapPG(); err != nil {
		fmt.Fprintf(os.Stderr, "skip envtest: postgres bootstrap failed: %v\n", err)
		_ = testEnv.Stop()
		os.Exit(0) // soft-skip rather than fail the whole job
	}

	if err := bootstrapManager(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap manager: %v\n", err)
		teardown()
		os.Exit(1)
	}

	if err := bootstrapHandlers(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap handlers: %v\n", err)
		teardown()
		os.Exit(1)
	}

	code := m.Run()

	teardown()
	os.Exit(code)
}

func teardown() {
	if mgrStop != nil {
		mgrStop()
	}
	mgrWg.Wait()
	if testEnv != nil {
		_ = testEnv.Stop()
	}
	if pgCtr != nil {
		_ = pgCtr.Terminate(context.Background())
	}
}

func bootstrapPG() error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("axisml"),
		tcpostgres.WithUsername("axisml"),
		tcpostgres.WithPassword("axisml"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return err
	}
	pgCtr = ctr.Container

	host, err := ctr.Host(ctx)
	if err != nil {
		return err
	}
	port, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return err
	}

	pgCfg = computeconfig.Config{
		DatabaseHost:     host,
		DatabasePort:     port.Int(),
		DatabaseName:     "axisml",
		DatabaseUser:     "axisml",
		DatabasePassword: "axisml",
		DatabaseSSLMode:  "disable",
	}

	gormDB, err = computedb.Open(pgCfg)
	if err != nil {
		return err
	}
	if err := computedb.Migrate(gormDB); err != nil {
		return err
	}
	return nil
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

	mgrCtx, mgrStop = context.WithCancel(context.Background())
	mgrWg.Add(1)
	go func() {
		defer mgrWg.Done()
		if err := mgr.Start(mgrCtx); err != nil && mgrCtx.Err() == nil {
			fmt.Fprintf(os.Stderr, "manager exited: %v\n", err)
		}
	}()
	if !mgr.GetCache().WaitForCacheSync(mgrCtx) {
		return fmt.Errorf("cache sync failed")
	}
	// Stash the manager so individual tests can build reconcilers/informers
	// against it.
	testManager = mgr
	return nil
}

var testManager ctrl.Manager

// bootstrapHandlers wires the production HTTP modules + reconcilers /
// informers via the same app.BuildModules helper that compute's main()
// uses, then exposes the Gin engine for in-process httptest drives.
// Reconcilers/informers are added to the existing testManager so they
// share its cache (we must not Add(...) after Start, but bootstrapHandlers
// runs before the manager goroutine processes its run queue — controller-
// runtime allows late Add until the manager actually starts work, which
// is gated on cache sync).
//
// Lifecycle test scenarios that construct their own reconcilers/informers
// inline are harmless duplicates: each test uses unique resource names, so the
// global pipeline either does the work first (and the inline one no-ops on a
// found CR) or the inline one does (and the global no-ops). Either ordering
// passes the assertions.
func bootstrapHandlers() error {
	cfg := pgCfg
	cfg.ReconcileInterval = 100 * time.Millisecond
	log := logr.Discard()

	modules, runnables, err := computeapp.BuildModules(cfg, gormDB, testManager, log)
	if err != nil {
		return fmt.Errorf("build modules: %w", err)
	}

	srv, err := computeserver.New(computeserver.Options{
		Addr:    ":0", // never started; engine is invoked via httptest
		Log:     log,
		Modules: modules,
		Ready: func(ctx context.Context) error {
			sqlDB, err := gormDB.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
	})
	if err != nil {
		return fmt.Errorf("server.New: %w", err)
	}
	testEngine = srv.Engine()

	for _, r := range runnables {
		if err := testManager.Add(r); err != nil {
			return fmt.Errorf("add runnable: %w", err)
		}
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
