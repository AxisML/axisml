//go:build e2e

// Package operators_test holds the L2 e2e tests that exercise the deployed
// AxisML operators (tenant / mljob / mlservice) against a real minikube
// cluster. They assume `make e2e-test` has already brought the cluster up,
// loaded images, and helm-installed both charts.
package operators_test

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/axisml/axisml/test/e2e"
)

// Package-level cluster handles, lazily initialized by the first setup() call.
// Tests in this package MUST NOT call t.Parallel — concurrent setup() invocations
// would race on these vars and on e2e.SetupOrSkip's namespace probe.
var (
	cfg *rest.Config
	c   client.Client
)

// setup is invoked from each test's first line so a missing cluster yields
// a per-test t.Skip rather than a process-wide TestMain failure.
func setup(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	if cfg == nil {
		cfg, c = e2e.SetupOrSkip(t)
	}
	return context.WithTimeout(context.Background(), 5*time.Minute)
}
