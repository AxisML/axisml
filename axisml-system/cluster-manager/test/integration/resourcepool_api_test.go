//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

// TestResourcePool_Lifecycle drives the full Create → Get → List → Patch →
// Delete cycle for a ResourcePool, verifying the cluster-manager REST API
// against the live K8s API server (envtest). This is the headline
// acceptance test the design calls out.
func TestResourcePool_Lifecycle(t *testing.T) {
	const name = "gpu-a100"

	// Create with two embedded units.
	body := `{
	  "name": "` + name + `",
	  "description": "A100 80GB single-node pool",
	  "nodeSelector": {"axisml.io/pool": "gpu-a100"},
	  "capacity": {"cpu": "64", "memory": "512Gi", "nvidia.com/gpu": "8"},
	  "labels": {"axisml.io/accelerator": "a100"},
	  "units": [
	    {
	      "name": "a100-1x-large",
	      "description": "1xA100 single-node",
	      "requests": {"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "1"},
	      "limits":   {"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "1"}
	    },
	    {
	      "name": "a100-2x-large",
	      "requests": {"cpu": "32", "memory": "256Gi", "nvidia.com/gpu": "2"},
	      "limits":   {"cpu": "32", "memory": "256Gi", "nvidia.com/gpu": "2"}
	    }
	  ]
	}`

	rr := doRequest(t, "POST", "/api/v1/resourcepools", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var created srv.ResourcePool
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
	require.Equal(t, name, created.Name)
	require.Equal(t, "A100 80GB single-node pool", created.Description)
	require.Len(t, created.Units, 2)
	gpuCapacity := created.Capacity["nvidia.com/gpu"]
	require.Equal(t, int64(8), gpuCapacity.Value())
	require.Equal(t, "a100", created.Labels["axisml.io/accelerator"])

	// Get returns the same thing.
	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+name, "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var got srv.ResourcePool
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, name, got.Name)
	cpuQ := got.Units[0].Requests["cpu"]
	require.Equal(t, "16", cpuQ.String())

	// List by label selector picks it up.
	rr = doRequest(t, "GET", "/api/v1/resourcepools?labelSelector=axisml.io/accelerator=a100", "")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var list srv.ResourcePoolList
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &list))
	require.GreaterOrEqual(t, list.Count, 1)

	// Patch description and capacity.
	rr = doRequest(t, "PATCH", "/api/v1/resourcepools/"+name, `{"description": "updated", "capacity": {"cpu": "96"}}`)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, "updated", got.Description)
	require.Equal(t, int64(96), got.Capacity.Cpu().Value())

	// Delete.
	rr = doRequest(t, "DELETE", "/api/v1/resourcepools/"+name, "")
	require.Equal(t, http.StatusNoContent, rr.Code, rr.Body.String())

	// Get now 404s.
	rr = doRequest(t, "GET", "/api/v1/resourcepools/"+name, "")
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestResourcePool_Create_DuplicateName ensures a second create with the
// same name returns 409.
func TestResourcePool_Create_DuplicateName(t *testing.T) {
	const name = "cpu-dup"

	body := `{
	  "name": "` + name + `",
	  "units": [
	    {"name": "cpu-small", "requests": {"cpu":"1"}, "limits":{"cpu":"1"}}
	  ]
	}`

	rr := doRequest(t, "POST", "/api/v1/resourcepools", body)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	t.Cleanup(func() { _ = doRequest(t, "DELETE", "/api/v1/resourcepools/"+name, "") })

	rr = doRequest(t, "POST", "/api/v1/resourcepools", body)
	require.Equal(t, http.StatusConflict, rr.Code)
}

// TestResourcePool_Create_BadName ensures an invalid name is rejected
// before the K8s API is contacted.
func TestResourcePool_Create_BadName(t *testing.T) {
	rr := doRequest(t, "POST", "/api/v1/resourcepools",
		`{"name": "BadName_With_Underscores"}`)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestResourcePool_Create_RejectsInvalidResources(t *testing.T) {
	rr := doRequest(t, "POST", "/api/v1/resourcepools", `{
	  "name": "bad-resource-pool",
	  "units": [{"name": "gpu-1x", "requests": {"gpu": "1"}, "limits": {"gpu": "1"}}]
	}`)
	require.Equal(t, http.StatusBadRequest, rr.Code, rr.Body.String())

	var problem srv.Error
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &problem))
	require.Equal(t, "InvalidResources", problem.Code)
	require.Contains(t, problem.Title, `resource "gpu"`)
}

// TestResourcePool_Auth_RequiresUser confirms /api/v1 routes 401 without
// X-Axisml-User.
func TestResourcePool_Auth_RequiresUser(t *testing.T) {
	rr := doRequestAs(t, "GET", "/api/v1/resourcepools", "", "")
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
