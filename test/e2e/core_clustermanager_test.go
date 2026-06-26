//go:build e2e || standard || lite

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
)

// Black-box cluster-manager tests. The default ResourcePool read works on both
// forms; multi-tenant provisioning is gated on CapMultiTenant (Standard backs it
// with the Tenant CR + tenant-operator; Lite serves a single static tenant and
// refuses tenant writes with 409 — asserted by the lite-only suite).

// TestClusterManagerDefaultPool reads the default ResourcePool both forms serve.
func TestClusterManagerDefaultPool(t *testing.T) {
	ctx := context.Background()
	p, err := h.ClusterManager().GetResourcePoolWithResponse(ctx, h.config().DefaultPool)
	require.NoError(t, err)
	require.Truef(t, is2xx(p.StatusCode()), "get default pool: %d: %s", p.StatusCode(), string(p.Body))
	require.NotNil(t, p.JSON200)
	assert.Equal(t, h.config().DefaultPool, p.JSON200.Name)
	assert.NotEmpty(t, p.JSON200.Units, "default pool should expose at least one unit")
}

// TestClusterManagerTenantCreate provisions a tenant through the API and verifies
// the read reflects it. Skipped on forms without multi-tenant support (Lite).
func TestClusterManagerTenantCreate(t *testing.T) {
	if !h.Supports(CapMultiTenant) {
		t.Skip("form does not support multi-tenant provisioning")
	}
	ctx := context.Background()
	name := uniqueName("e2e-apitenant")
	r, err := h.ClusterManager().CreateTenantWithResponse(ctx, clustermanager.CreateTenantRequest{
		Name:      ptr(name),
		Namespace: &clustermanager.Apiv1alpha1NamespaceSpec{Name: name},
		Quotas: &[]clustermanager.ServerQuota{{
			Pool:  h.config().DefaultPool,
			Units: []clustermanager.ServerQuotaUnit{{UnitName: h.config().DefaultUnit, Quantity: 1}},
		}},
	})
	require.NoError(t, err)
	require.Truef(t, is2xx(r.StatusCode()), "create tenant: %d: %s", r.StatusCode(), string(r.Body))
	t.Cleanup(func() {
		_, _ = h.ClusterManager().DeleteTenantWithResponse(context.Background(), name)
	})

	g, err := h.ClusterManager().GetTenantWithResponse(ctx, name)
	require.NoError(t, err)
	require.Truef(t, is2xx(g.StatusCode()), "get tenant: %d", g.StatusCode())
	require.NotNil(t, g.JSON200)
	assert.Equal(t, name, g.JSON200.Name)
}
