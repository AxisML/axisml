//go:build lite

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
)

// httpClient is a bare client for probing arbitrary in-container workloads over
// their base URL (e.g. GET /api/v1/capabilities against axisml-core), which have
// no OpenAPI contract to generate a typed client from. The AxisML System
// components are reached through their typed clients instead. Only the Lite form
// probes raw endpoints this way, so it lives under the lite tag.
type httpClient struct {
	baseURL string
	c       *http.Client
}

func newHTTPClient(baseURL string) *httpClient {
	return &httpClient{baseURL: baseURL, c: &http.Client{Timeout: 30 * time.Second}}
}

// resp is the result of a raw HTTP call.
type resp struct {
	status int
	body   []byte
}

// do issues a request. body may be nil.
func (hc *httpClient) do(ctx context.Context, method, path string, body any) (resp, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return resp{}, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, hc.baseURL+path, rdr)
	if err != nil {
		return resp{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpResp, err := hc.c.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer func() { _ = httpResp.Body.Close() }()
	b, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp{}, err
	}
	return resp{status: httpResp.StatusCode, body: b}, nil
}

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
