package testutil

import (
	"errors"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// EnvtestOptions configures StartEnvtest.
type EnvtestOptions struct {
	// CRDPaths is the list of directories that envtest scans for CRD manifests.
	// Pass paths to both repo-local CRDs (axisml-system/deploy/helm/crds) and
	// vendored external CRDs (test/crds/external) as needed.
	CRDPaths []string

	// Scheme is the runtime scheme used to construct the test client. Callers
	// must register every API group they intend to interact with (corev1,
	// rbacv1, the operator's own group, plus any external CRD types).
	Scheme *runtime.Scheme

	// ErrorIfCRDPathMissing causes envtest.Start to fail fast if any CRD path
	// doesn't exist. Default true; set false only for opt-in-style tests.
	ErrorIfCRDPathMissing *bool
}

// EnvtestHandle is the result of StartEnvtest. Callers MUST defer Stop to
// shut down the embedded apiserver+etcd cleanly.
type EnvtestHandle struct {
	Env    *envtest.Environment
	Cfg    *rest.Config
	Client client.Client
}

// Stop tears down the envtest environment.
func (h *EnvtestHandle) Stop() error {
	if h == nil || h.Env == nil {
		return nil
	}
	return h.Env.Stop()
}

// StartEnvtestE spins up an embedded apiserver+etcd via controller-runtime's
// envtest package, loads the CRDs in opts.CRDPaths, and returns a client wired
// to opts.Scheme. Returns an error rather than calling t.Fatal — TestMain must
// use this variant because t.Fatal in TestMain's main goroutine triggers
// runtime.Goexit and crashes the test binary instead of reporting the error.
//
// KUBEBUILDER_ASSETS must point at a directory containing etcd, kube-apiserver,
// and kubectl; each component's `make integration` target sets it via the
// shared setup-envtest binary at test/setup-envtest/setup-envtest.
func StartEnvtestE(opts EnvtestOptions) (*EnvtestHandle, error) {
	if opts.Scheme == nil {
		return nil, errors.New("StartEnvtestE: opts.Scheme is required")
	}
	errIfMissing := true
	if opts.ErrorIfCRDPathMissing != nil {
		errIfMissing = *opts.ErrorIfCRDPathMissing
	}
	env := &envtest.Environment{
		CRDDirectoryPaths:     opts.CRDPaths,
		ErrorIfCRDPathMissing: errIfMissing,
		Scheme:                opts.Scheme,
	}
	cfg, err := env.Start()
	if err != nil {
		return nil, fmt.Errorf("envtest.Start: %w", err)
	}
	c, err := client.New(cfg, client.Options{Scheme: opts.Scheme})
	if err != nil {
		_ = env.Stop()
		return nil, fmt.Errorf("envtest client.New: %w", err)
	}
	return &EnvtestHandle{Env: env, Cfg: cfg, Client: c}, nil
}

// StartEnvtest is the testing.T-friendly wrapper around StartEnvtestE for use
// from individual test functions. Do NOT call this from TestMain; use
// StartEnvtestE there.
func StartEnvtest(t *testing.T, opts EnvtestOptions) *EnvtestHandle {
	t.Helper()
	h, err := StartEnvtestE(opts)
	if err != nil {
		t.Fatalf("StartEnvtest: %v", err)
	}
	return h
}
