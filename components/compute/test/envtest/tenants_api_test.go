//go:build envtest

package envtest_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/test/testutil"
)

// TestTenantsAPI_CreateAndSuspend drives the full tenant lifecycle through
// the HTTP API: create → wait for the reconciler to create the CR → suspend
// → verify the CR's spec.suspended=true → unsuspend → verify back to false.
// This locks in the contract that the suspend/unsuspend custom actions in
// /api/v1/tenants/:tenant/{suspend,unsuspend} actually drive the CR's
// spec.suspended boolean rather than just flipping a DB flag.
func TestTenantsAPI_CreateAndSuspend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	const (
		tenantName = "api-tenant-susp"
		tenantNS   = "api-tenant-susp-ns"
	)
	t.Cleanup(func() {
		// DELETE may fail with not-found if a previous step already removed
		// the row; ignore status here.
		_ = doJSON(t, context.Background(), http.MethodDelete, pathf("/api/v1/tenants/%s", tenantName), nil, nil)
	})

	// Create.
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/tenants", map[string]any{
		"name":        tenantName,
		"displayName": "API Test Tenant",
		"namespace":   map[string]any{"name": tenantNS},
	}, nil)
	requireStatus(t, rr, http.StatusCreated)
	idAndName(t, rr)

	// Reconciler should land the CR.
	cr := &tenantv1alpha1.Tenant{}
	testutil.EventuallyExists(t, ctx, cl, client.ObjectKey{Name: tenantName}, cr, 10*time.Second)
	require.Equal(t, tenantNS, cr.Spec.Namespace.Name)
	require.False(t, cr.Spec.Suspended, "CR should not be suspended at creation")

	// Suspend.
	rr = doJSON(t, ctx, http.MethodPost, pathf("/api/v1/tenants/%s/suspend", tenantName), nil, nil)
	requireStatus(t, rr, http.StatusOK)

	testutil.Eventually(t, 10*time.Second, testutil.DefaultPollInterval, func() error {
		fresh := &tenantv1alpha1.Tenant{}
		if err := cl.Get(ctx, client.ObjectKey{Name: tenantName}, fresh); err != nil {
			return err
		}
		if !fresh.Spec.Suspended {
			return errBecause("CR.spec.suspended still false")
		}
		return nil
	})

	// Unsuspend.
	rr = doJSON(t, ctx, http.MethodPost, pathf("/api/v1/tenants/%s/unsuspend", tenantName), nil, nil)
	requireStatus(t, rr, http.StatusOK)

	testutil.Eventually(t, 10*time.Second, testutil.DefaultPollInterval, func() error {
		fresh := &tenantv1alpha1.Tenant{}
		if err := cl.Get(ctx, client.ObjectKey{Name: tenantName}, fresh); err != nil {
			return err
		}
		if fresh.Spec.Suspended {
			return errBecause("CR.spec.suspended still true")
		}
		return nil
	})
}

// TestTenantsAPI_DuplicateNameConflict ensures the service rejects a
// second create for the same active tenant — protecting against double-
// submit races at the API layer.
func TestTenantsAPI_DuplicateNameConflict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const tenantName = "api-tenant-dup"
	t.Cleanup(func() {
		_ = doJSON(t, context.Background(), http.MethodDelete, pathf("/api/v1/tenants/%s", tenantName), nil, nil)
	})

	body := map[string]any{
		"name":      tenantName,
		"namespace": map[string]any{"name": tenantName + "-ns"},
	}
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/tenants", body, nil)
	requireStatus(t, rr, http.StatusCreated)

	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/tenants", body, nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate create: status=%d body=%s, want 409", rr.Code, rr.Body.String())
	}
}

// errBecause is a tiny helper to keep Eventually polls readable when the
// only thing they assert is "this hasn't happened yet."
func errBecause(msg string) error { return &polledErr{msg: msg} }

type polledErr struct{ msg string }

func (e *polledErr) Error() string { return e.msg }
