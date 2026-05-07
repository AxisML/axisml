//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// TestTenant_QuotaList covers GET /tenants/{name}/quotas, which returns
// the spec.quotas[] under "spec" and status.quotas[] under "status".
func TestTenant_QuotaList(t *testing.T) {
	const name = "team-quota-list"
	t.Cleanup(func() { _ = deleteTenant(name) })

	rr := doRequest(t, http.MethodPost, "/api/v1/tenants", `{
	  "name": "team-quota-list",
	  "namespace": {"name": "team-quota-list-ns"},
	  "quotas": [
	    {"pool": "default", "name": "default", "max": {"cpu": "10"}},
	    {"pool": "gpu",     "name": "default", "max": {"nvidia.com/gpu": "4"}}
	  ]
	}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	rr = doRequest(t, http.MethodGet, "/api/v1/tenants/"+name+"/quotas", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp struct {
		Spec []map[string]any `json:"spec"`
		// status may be absent until the tenant-operator reconciles; we
		// don't assert on its contents here.
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Spec, 2)
	pools := map[string]bool{}
	for _, q := range resp.Spec {
		pools[fmt.Sprintf("%v/%v", q["pool"], q["name"])] = true
	}
	assert.True(t, pools["default/default"], "spec should include default/default quota")
	assert.True(t, pools["gpu/default"], "spec should include gpu/default quota")
}

// TestTenant_QuotaDelete drives DELETE /tenants/{name}/quotas/{pool}/{quota}.
// Removing an existing quota returns 204 and the quota is gone from spec;
// removing a non-existent quota is idempotent (also 204) — same convention
// as Tenant DELETE.
func TestTenant_QuotaDelete(t *testing.T) {
	const name = "team-quota-delete"
	t.Cleanup(func() { _ = deleteTenant(name) })

	rr := doRequest(t, http.MethodPost, "/api/v1/tenants", `{
	  "name": "team-quota-delete",
	  "namespace": {"name": "team-quota-delete-ns"},
	  "quotas": [
	    {"pool": "default", "name": "default", "max": {"cpu": "10"}},
	    {"pool": "gpu",     "name": "default", "max": {"nvidia.com/gpu": "4"}}
	  ]
	}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	// Delete the gpu quota.
	rr = doRequest(t, http.MethodDelete,
		"/api/v1/tenants/"+name+"/quotas/gpu/default", "")
	require.Equal(t, http.StatusNoContent, rr.Code)

	// Verify spec.quotas[] no longer contains it.
	var stored tenantv1alpha1.Tenant
	require.NoError(t, testCli.Get(context.Background(),
		types.NamespacedName{Name: name}, &stored))
	for _, q := range stored.Spec.Quotas {
		if q.Pool == "gpu" && q.Name == "default" {
			t.Fatalf("gpu/default quota still present after DELETE: %+v", stored.Spec.Quotas)
		}
	}

	// Idempotent: a second DELETE on the same now-missing quota is also 204.
	rr = doRequest(t, http.MethodDelete,
		"/api/v1/tenants/"+name+"/quotas/gpu/default", "")
	require.Equal(t, http.StatusNoContent, rr.Code)

	// DELETE on an unknown pool/quota tuple is also 204 (idempotent).
	rr = doRequest(t, http.MethodDelete,
		"/api/v1/tenants/"+name+"/quotas/no-such-pool/no-such-quota", "")
	require.Equal(t, http.StatusNoContent, rr.Code)
}

// TestTenant_ListPagination seeds three tenants and verifies the ?limit=N
// query parameter is honoured: the first page returns at most N items
// plus a continue token, and the follow-up page returns the rest.
func TestTenant_ListPagination(t *testing.T) {
	names := []string{"page-alpha", "page-beta", "page-gamma"}
	t.Cleanup(func() {
		for _, n := range names {
			_ = deleteTenant(n)
		}
	})
	for _, n := range names {
		rr := doRequest(t, http.MethodPost, "/api/v1/tenants",
			fmt.Sprintf(`{"name": %q, "namespace": {"name": %q}}`, n, n+"-ns"))
		require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	}

	rr := doRequest(t, http.MethodGet, "/api/v1/tenants?limit=2", "")
	require.Equal(t, http.StatusOK, rr.Code)
	var page1 struct {
		Items    []map[string]any `json:"items"`
		Continue string           `json:"continue"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &page1))
	assert.LessOrEqual(t, len(page1.Items), 2)
	// The K8s API server only returns a continue token when the list was
	// truly paginated; with three tenants and limit=2, one is expected.
	require.NotEmpty(t, page1.Continue)

	rr = doRequest(t, http.MethodGet,
		"/api/v1/tenants?limit=2&continue="+page1.Continue, "")
	require.Equal(t, http.StatusOK, rr.Code)
	var page2 struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &page2))
	assert.GreaterOrEqual(t, len(page1.Items)+len(page2.Items), 3)
}

func deleteTenant(name string) error {
	err := testCli.Delete(context.Background(), &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
