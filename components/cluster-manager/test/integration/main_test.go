//go:build integration

// Package integration_test exercises the cluster-manager Tenant + Quota
// REST handlers against an embedded apiserver+etcd via controller-runtime
// envtest. The Tenant CRD is loaded from deploy/helm/axisml-system/crds.
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

	cmk8sclient "github.com/axisml/axisml/components/cluster-manager/internal/k8sclient"
	cmtenant "github.com/axisml/axisml/components/cluster-manager/internal/tenant"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/test/testutil"
)

var (
	testEnv  *testutil.EnvtestHandle
	testCli  client.Client
	testRtr  *gin.Engine
	denylist = []string{"kube-system", "default", "axisml-system"}
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
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))

	testEnv, err = testutil.StartEnvtestE(testutil.EnvtestOptions{
		Scheme: scheme,
		CRDPaths: []string{
			filepath.Join(repoRoot, "deploy", "helm", "axisml-system", "crds"),
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
	r := gin.New()
	(&cmtenant.Handler{Client: testCli, NamespaceDenylist: denylist}).Register(r.Group("/api/v1"))
	testRtr = r

	code := m.Run()

	_ = testEnv.Stop()
	os.Exit(code)
}

// doRequest is a tiny helper around the in-process gin router.
func doRequest(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, stringReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
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
		if _, err := os.Stat(filepath.Join(dir, "deploy", "helm", "axisml-system", "crds")); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("could not locate repo root from %s", cwd)
}
