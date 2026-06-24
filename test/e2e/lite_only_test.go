//go:build lite

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
)

// Lite-only assertions: the aggregate capability document axisml-core serves and
// the 409 CapabilityUnavailable refusals for the writes Lite intentionally does
// not support (single read-only pool, single static tenant). The Standard form
// has no analogue (per-service capability docs; full CRUD), so these live under
// the lite tag only.

// TestLiteCapabilitiesDocument checks the aggregate document shape and that the
// compute module reports the Standalone runtime.
func TestLiteCapabilitiesDocument(t *testing.T) {
	r, err := newHTTPClient(h.baseURL).do(context.Background(), http.MethodGet, "/api/v1/capabilities", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.status)

	var doc struct {
		Components map[string]map[string]any `json:"components"`
	}
	require.NoError(t, json.Unmarshal(r.body, &doc))
	require.Contains(t, doc.Components, "cluster-manager")
	require.Contains(t, doc.Components, "compute-service")
	require.Contains(t, doc.Components, "artifact-hub")
	assert.Equal(t, "standalone", doc.Components["compute-service"]["runtime"])
	assert.Equal(t, false, doc.Components["cluster-manager"]["multiTenant"])
}

// TestLiteRefusesResourcePoolWrite verifies a pool create reaches the read-only
// store and is refused with 409 CapabilityUnavailable.
func TestLiteRefusesResourcePoolWrite(t *testing.T) {
	r, err := h.ClusterManager().CreateResourcePoolWithResponse(context.Background(),
		clustermanager.CreateResourcePoolRequest{Name: ptr("extra")})
	require.NoError(t, err)
	assert.Equalf(t, http.StatusConflict, r.StatusCode(), "create pool: %d: %s", r.StatusCode(), string(r.Body))
}

// TestLiteRefusesTenantCreate verifies multi-tenant provisioning is refused.
func TestLiteRefusesTenantCreate(t *testing.T) {
	name := uniqueName("e2e-extra-tenant")
	r, err := h.ClusterManager().CreateTenantWithResponse(context.Background(),
		clustermanager.CreateTenantRequest{
			Name:      ptr(name),
			Namespace: &clustermanager.Apiv1alpha1NamespaceSpec{Name: name},
		})
	require.NoError(t, err)
	assert.Equalf(t, http.StatusConflict, r.StatusCode(), "create tenant: %d: %s", r.StatusCode(), string(r.Body))
}
