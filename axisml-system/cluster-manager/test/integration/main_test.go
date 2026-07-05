//go:build integration

// Package integration_test drives the cluster-manager ResourcePool REST API
// against an embedded apiserver+etcd (controller-runtime envtest), with the
// ResourcePool CRD loaded from axisml-system/deploy/helm/crds.
package integration_test

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/cluster-manager/api/v1alpha1"
	cmapp "github.com/axisml/axisml/axisml-system/cluster-manager/internal/app"
	cmk8sclient "github.com/axisml/axisml/axisml-system/cluster-manager/internal/k8sclient"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/testutil"
)

var (
	testEnv *testutil.EnvtestHandle
	testCli client.Client
	testRtr *gin.Engine
)

func TestMain(m *testing.M) {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repo root: %v\n", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(axismlv1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))

	testEnv, err = testutil.StartEnvtestE(testutil.EnvtestOptions{
		Scheme: scheme,
		CRDPaths: []string{
			filepath.Join(repoRoot, "axisml-system", "deploy", "helm", "crds"),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start envtest: %v\n", err)
		os.Exit(1)
	}

	c, err := cmk8sclient.BuildWithConfig(testEnv.Cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build client: %v\n", err)
		_ = testEnv.Stop()
		os.Exit(1)
	}
	testCli = c

	gin.SetMode(gin.TestMode)
	// Metrics querier is nil here: the per-pool metrics endpoint reports
	// metrics-unavailable, exercised in the resourcepool usage/metrics test.
	testRtr = cmapp.NewRouter(testCli, nil)

	code := m.Run()

	_ = testEnv.Stop()
	os.Exit(code)
}

// doRequest is a tiny helper around the in-process gin router. All requests
// receive a default X-Axisml-User unless the test explicitly overrides it.
func doRequest(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequestAs(t, method, path, body, "test-user")
}

func doRequestAs(t *testing.T, method, path, body, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, stringReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set("X-Axisml-User", user)
	}
	rr := httptest.NewRecorder()
	testRtr.ServeHTTP(rr, req)
	return rr
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; dir != "/" && dir != ""; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "axisml-system", "deploy", "helm", "crds")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("could not locate repo root from %s", cwd)
}
