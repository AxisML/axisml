//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// TestTenantLifecycle covers the happy-path Create / Get / Patch / Quota
// add+update / Suspend / Delete sequence end-to-end via the cluster-
// manager handlers backed by a real K8s API server (envtest).
func TestTenantLifecycle(t *testing.T) {
	const tenantName = "team-alpha"

	// Create
	createBody := `{
	  "name": "team-alpha",
	  "displayName": "Team Alpha",
	  "namespace": {"name": "team-alpha-ns"},
	  "quotas": [{"pool": "default", "name": "default", "max": {"cpu": "10", "memory": "20Gi"}}]
	}`
	rr := doRequest(t, http.MethodPost, "/api/v1/tenants", createBody)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Verify on the API server.
	var stored tenantv1alpha1.Tenant
	require.NoError(t, testCli.Get(context.Background(),
		types.NamespacedName{Name: tenantName}, &stored))
	assert.Equal(t, "team-alpha-ns", stored.Spec.Namespace.Name)
	require.Len(t, stored.Spec.Quotas, 1)
	assert.Equal(t, "default", stored.Spec.Quotas[0].Pool)
	assert.NotEmpty(t, stored.Labels[tenantv1alpha1.LabelTenantID])

	// Get
	rr = doRequest(t, http.MethodGet, "/api/v1/tenants/"+tenantName, "")
	require.Equal(t, http.StatusOK, rr.Code)

	// Duplicate Create -> 409 (mapped from IsAlreadyExists).
	rr = doRequest(t, http.MethodPost, "/api/v1/tenants", createBody)
	require.Equal(t, http.StatusConflict, rr.Code)

	// Patch displayName
	rr = doRequest(t, http.MethodPatch,
		"/api/v1/tenants/"+tenantName,
		`{"displayName": "Alpha Squad"}`)
	require.Equal(t, http.StatusOK, rr.Code)

	// Add a second quota
	rr = doRequest(t, http.MethodPost,
		"/api/v1/tenants/"+tenantName+"/quotas",
		`{"pool": "gpu", "name": "default", "max": {"nvidia.com/gpu": "4"}}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Duplicate quota -> 409.
	rr = doRequest(t, http.MethodPost,
		"/api/v1/tenants/"+tenantName+"/quotas",
		`{"pool": "gpu", "name": "default", "max": {"nvidia.com/gpu": "4"}}`)
	require.Equal(t, http.StatusConflict, rr.Code)

	// Update existing quota max.
	rr = doRequest(t, http.MethodPatch,
		"/api/v1/tenants/"+tenantName+"/quotas/gpu/default",
		`{"pool": "gpu", "name": "default", "max": {"nvidia.com/gpu": "8"}}`)
	require.Equal(t, http.StatusOK, rr.Code)

	// Suspend
	rr = doRequest(t, http.MethodPost,
		"/api/v1/tenants/"+tenantName+"/suspend", "")
	require.Equal(t, http.StatusOK, rr.Code)
	require.NoError(t, testCli.Get(context.Background(),
		types.NamespacedName{Name: tenantName}, &stored))
	assert.True(t, stored.Spec.Suspended)

	// Unsuspend.
	rr = doRequest(t, http.MethodPost,
		"/api/v1/tenants/"+tenantName+"/unsuspend", "")
	require.Equal(t, http.StatusOK, rr.Code)

	// List has the tenant.
	rr = doRequest(t, http.MethodGet, "/api/v1/tenants", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var listResp struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &listResp))
	assert.NotEmpty(t, listResp.Items)

	// Delete -> 204; second delete also 204 (idempotent).
	rr = doRequest(t, http.MethodDelete, "/api/v1/tenants/"+tenantName, "")
	require.Equal(t, http.StatusNoContent, rr.Code)
	rr = doRequest(t, http.MethodDelete, "/api/v1/tenants/"+tenantName, "")
	require.Equal(t, http.StatusNoContent, rr.Code)
}

// TestValidationRejects covers the up-front DNS-1123 + denylist rejections
// cluster-manager performs before forwarding to the API server.
func TestValidationRejects(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"empty name", `{"name":"","namespace":{"name":"x-ns"}}`, http.StatusBadRequest},
		{"uppercase name", `{"name":"Alpha","namespace":{"name":"x-ns"}}`, http.StatusBadRequest},
		{"denylisted ns", `{"name":"alpha","namespace":{"name":"kube-system"}}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doRequest(t, http.MethodPost, "/api/v1/tenants", tc.body)
			require.Equal(t, tc.want, rr.Code, rr.Body.String())
		})
	}
}

// TestNotFound covers the path where the Tenant CR doesn't exist.
func TestNotFound(t *testing.T) {
	rr := doRequest(t, http.MethodGet, "/api/v1/tenants/does-not-exist", "")
	require.Equal(t, http.StatusNotFound, rr.Code)
}
