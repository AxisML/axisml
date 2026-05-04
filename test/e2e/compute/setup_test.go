//go:build e2e

// Package compute_test exercises the compute service end-to-end against the
// helm-installed AxisML stack. The bootstrap path is the smoke test: after
// `make helm-install` runs, the compute post-install Job must seed PG with
// the default tenant, and compute's reconciler must create the corresponding
// Tenant CR for the tenant-operator to land.
package compute_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"

	"github.com/axisml/axisml/test/e2e"
	"github.com/axisml/axisml/test/testutil"
)

var clusterClient client.Client

func setup(t *testing.T) (context.Context, context.CancelFunc, client.Client) {
	t.Helper()
	if clusterClient == nil {
		_, clusterClient = e2e.SetupOrSkip(t)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	return ctx, cancel, clusterClient
}

// TestComputeDeploymentReady verifies the helm-installed compute Deployment
// reaches at least one available replica.
func TestComputeDeploymentReady(t *testing.T) {
	ctx, cancel, c := setup(t)
	defer cancel()

	depName := e2e.HelmRelease() + "-compute"
	e2e.WaitDeploymentAvailable(t, ctx, c, e2e.SystemNamespace, depName, 1, 3*time.Minute)

	// Sanity-check the Deployment shape (image + args), not just availability.
	var dep appsv1.Deployment
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: e2e.SystemNamespace, Name: depName}, &dep))
	require.Equal(t, "compute", dep.Spec.Template.Spec.Containers[0].Name)
	require.Contains(t, dep.Spec.Template.Spec.Containers[0].Args, "serve")
}

// TestComputeBootstrapTenant verifies the post-install Job seeded PG with
// the default tenant and compute's reconciler created the Tenant CR.
func TestComputeBootstrapTenant(t *testing.T) {
	ctx, cancel, c := setup(t)
	defer cancel()

	cr := &tenantv1alpha1.Tenant{}
	testutil.EventuallyExists(t, ctx, c, client.ObjectKey{Name: "default"}, cr, 3*time.Minute)
	require.NotEmpty(t, cr.Labels[tenantv1alpha1.LabelTenantID], "compute must stamp tenant-id label on CR")

	// Spec sync may lag CR creation when PG state predates a helm values
	// change (e.g. defaultTenantNamespace tweak); poll until the compute
	// reconciler patches the CR.
	require.Eventually(t, func() bool {
		fresh := &tenantv1alpha1.Tenant{}
		if err := c.Get(ctx, client.ObjectKey{Name: "default"}, fresh); err != nil {
			return false
		}
		return fresh.Spec.Namespace.Name == "axisml-default"
	}, 3*time.Minute, 5*time.Second, "default tenant CR did not converge to spec.namespace.name=axisml-default (K8s 'default' is in tenant-operator denylist)")

	// Wait for tenant-operator to reach Active.
	require.Eventually(t, func() bool {
		fresh := &tenantv1alpha1.Tenant{}
		if err := c.Get(ctx, client.ObjectKey{Name: "default"}, fresh); err != nil {
			return false
		}
		return fresh.Status.Phase == tenantv1alpha1.TenantPhaseActive
	}, 3*time.Minute, 5*time.Second, "default tenant did not reach Active phase")
}
