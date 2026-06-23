//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// doTenant drives the engine like `do` but also sets the active-tenant header
// (X-Axisml-Tenant) that name-addressed workload endpoints scope on.
func doTenant(t *testing.T, method, path, token, tenant string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tenant != "" {
		req.Header.Set("X-Axisml-Tenant", tenant)
	}
	rec := httptest.NewRecorder()
	testEngine.ServeHTTP(rec, req)
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// TestSuspensionGate verifies guard.TenantActive actually blocks workload
// creation on a suspended tenant (409 tenant-suspended) and that resuming lifts
// the block. The workspace-create path stands in for every gated workload entry
// point — run / service / traffic all route through the same guard.TenantActive.
//
// The compute backend is unstubbed in this suite (only cluster-manager is), so
// the gate is the only thing that can return tenant-suspended: while suspended
// we assert the 409+code; after resume we assert the call clears the gate (it
// then reaches compute and fails downstream, but never with tenant-suspended).
func TestSuspensionGate(t *testing.T) {
	admin := loginAdmin(t)

	code, body := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": "suspowner", "password": "password123", "displayName": "suspowner",
	})
	require.Equal(t, http.StatusCreated, code, "create user: %v", body)

	code, tn := do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "gated-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Gated", "initialAdmin": "suspowner",
	})
	require.Equal(t, http.StatusCreated, code, "create tenant: %v", tn)

	owner := loginAs(t, "suspowner", "password123")
	wsBody := map[string]any{
		"name": "ws1", "image": "jupyter:latest",
		"poolName": "default", "unitName": "gpu-small", "containerPort": 8888,
	}

	// Suspend → workload create is refused at the gate, before any compute call.
	code, _ = do(t, http.MethodPost, "/api/v1/tenants/gated-team/suspend", admin, nil)
	require.Equal(t, http.StatusOK, code)

	code, blocked := doTenant(t, http.MethodPost, "/api/v1/workspaces", owner, "gated-team", wsBody)
	require.Equal(t, http.StatusConflict, code, "suspended create must be 409: %v", blocked)
	require.Equal(t, "tenant-suspended", blocked["code"])

	// Resume → the gate no longer blocks; the call proceeds past it.
	code, _ = do(t, http.MethodPost, "/api/v1/tenants/gated-team/resume", admin, nil)
	require.Equal(t, http.StatusOK, code)

	_, after := doTenant(t, http.MethodPost, "/api/v1/workspaces", owner, "gated-team", wsBody)
	require.NotEqual(t, "tenant-suspended", after["code"], "resume must lift the gate: %v", after)
}
