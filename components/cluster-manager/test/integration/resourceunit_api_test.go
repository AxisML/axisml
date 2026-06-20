//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
)

// TestResourceUnit_Lifecycle verifies the unit sub-routes: add → get →
// patch → delete operate atomically on the parent pool's spec.units[].
func TestResourceUnit_Lifecycle(t *testing.T) {
	const poolName = "cpu-unit-test"

	// Seed pool with one unit.
	rr := doRequest(t, "POST", "/api/v1/resourcepools", `{
	  "name": "`+poolName+`",
	  "units": [
	    {"name": "cpu-small", "requests": {"cpu":"1"}, "limits": {"cpu":"1"}}
	  ]
	}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	t.Cleanup(func() { _ = doRequest(t, "DELETE", "/api/v1/resourcepools/"+poolName, "") })

	// Add a second unit.
	rr = doRequest(t, "POST", "/api/v1/resourcepools/"+poolName+"/units", `{
	  "name": "cpu-medium",
	  "description": "4-core medium",
	  "requests": {"cpu": "4", "memory": "8Gi"},
	  "limits":   {"cpu": "4", "memory": "8Gi"}
	}`)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var added srv.ResourceUnitDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &added))
	require.Equal(t, "cpu-medium", added.Name)
	cpuQ := added.Requests["cpu"]
	require.Equal(t, "4", cpuQ.String())

	// List shows both.
	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+poolName+"/units", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var list srv.ResourceUnitList
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	require.Equal(t, 2, list.Count)

	// Get one.
	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+poolName+"/units/cpu-medium", "")
	require.Equal(t, http.StatusOK, rr.Code)

	// Patch cpu-medium's limits.
	rr = doRequest(t, "PATCH", "/api/v1/resourcepools/"+poolName+"/units/cpu-medium",
		`{"limits": {"cpu": "8", "memory": "16Gi"}}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var patched srv.ResourceUnitDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &patched))
	cpuLim := patched.Limits["cpu"]
	require.Equal(t, "8", cpuLim.String())

	// Duplicate add returns 409.
	rr = doRequest(t, "POST", "/api/v1/resourcepools/"+poolName+"/units", `{
	  "name": "cpu-medium",
	  "requests": {"cpu":"1"},
	  "limits":   {"cpu":"1"}
	}`)
	require.Equal(t, http.StatusConflict, rr.Code)

	// Delete cpu-medium.
	rr = doRequest(t, "DELETE", "/api/v1/resourcepools/"+poolName+"/units/cpu-medium", "")
	require.Equal(t, http.StatusNoContent, rr.Code)

	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+poolName+"/units", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	require.Equal(t, 1, list.Count)
}
