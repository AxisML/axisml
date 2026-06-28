// Package apidoc builds the OpenAPI 3.0.3 description of the cluster-manager
// HTTP API and is the single source of truth for that contract.
//
// The cluster-manager exposes ResourcePool and Tenant CRD CRUD (with embedded
// spec.units[]); request/response types are mirrored from internal/server.
//
// The component's own cmd/openapi-gen renders Document to
// axisml-system/docs/apis/cluster-manager.yaml; axisml-core imports Document
// directly to fold the cluster-manager surface into the Lite composite spec,
// without a YAML round-trip.
package apidoc

import (
	"github.com/axisml/axisml/components/cluster-manager/internal/server"
	"github.com/axisml/axisml/pkg/openapigen"
)

const (
	tagResourcePools = "ResourcePools"
	tagResourceUnits = "ResourceUnits"
	tagTenants       = "Tenants"
	tagVolumes       = "Volumes"
	tagCapabilities  = "Capabilities"
	tagHealth        = "Health"
)

func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "ClusterManagerError"),
		"401":     openapigen.JSONResp("Missing or invalid X-Axisml-User.", "ClusterManagerError"),
		"404":     openapigen.JSONResp("Not found.", "ClusterManagerError"),
		"409":     openapigen.JSONResp("Conflict (already exists or concurrent modification).", "ClusterManagerError"),
		"422":     openapigen.JSONResp("Invariant violated.", "ClusterManagerError"),
		"500":     openapigen.JSONResp("Internal error.", "ClusterManagerError"),
		"default": openapigen.JSONResp("Unexpected error.", "ClusterManagerError"),
	}
	for k, v := range success {
		out[k] = v
	}
	return out
}

// Document builds the complete OpenAPI document for the cluster-manager API.
func Document(version string) *openapigen.Document {
	g := openapigen.New(openapigen.Options{})

	g.Register("ClusterManagerError", server.Error{}, openapigen.ResponseMode)
	g.Register("CreateResourcePoolRequest", server.CreateResourcePoolRequest{}, openapigen.InputMode)
	g.Register("PatchResourcePoolRequest", server.PatchResourcePoolRequest{}, openapigen.InputMode)
	g.Register("CreateResourceUnitRequest", server.CreateResourceUnitRequest{}, openapigen.InputMode)
	g.Register("PatchResourceUnitRequest", server.PatchResourceUnitRequest{}, openapigen.InputMode)
	g.Register("ResourcePool", server.ResourcePool{}, openapigen.ResponseMode)
	g.Register("ResourceUnit", server.ResourceUnit{}, openapigen.ResponseMode)
	g.Register("ResourcePoolList", server.ResourcePoolList{}, openapigen.ResponseMode)
	g.Register("ResourceUnitList", server.ResourceUnitList{}, openapigen.ResponseMode)
	g.Register("CreateTenantRequest", server.CreateTenantRequest{}, openapigen.InputMode)
	g.Register("PatchTenantRequest", server.PatchTenantRequest{}, openapigen.InputMode)
	g.Register("SetQuotaRequest", server.SetQuotaRequest{}, openapigen.InputMode)
	g.Register("PatchQuotaRequest", server.PatchQuotaRequest{}, openapigen.InputMode)
	g.Register("Tenant", server.Tenant{}, openapigen.ResponseMode)
	g.Register("TenantList", server.TenantList{}, openapigen.ResponseMode)
	g.Register("Quota", server.Quota{}, openapigen.ResponseMode)
	g.Register("QuotaList", server.QuotaList{}, openapigen.ResponseMode)
	g.Register("Capabilities", server.Capabilities{}, openapigen.ResponseMode)
	g.Register("CreateVolumeRequest", server.CreateVolumeRequest{}, openapigen.InputMode)
	g.Register("PatchVolumeRequest", server.PatchVolumeRequest{}, openapigen.InputMode)
	g.Register("Volume", server.Volume{}, openapigen.ResponseMode)
	g.Register("VolumeList", server.VolumeList{}, openapigen.ResponseMode)
	g.Register("StorageClass", server.StorageClass{}, openapigen.ResponseMode)
	g.Register("StorageClassList", server.StorageClassList{}, openapigen.ResponseMode)

	registerExamples(g)

	tags := []openapigen.TagEntry{
		{Name: tagResourcePools, Description: "ResourcePool CRD CRUD."},
		{Name: tagResourceUnits, Description: "Sub-routes over pool.spec.units[]."},
		{Name: tagTenants, Description: "Tenant CRD CRUD and per-pool tenant quotas (unit × quantity, folded to ElasticQuota); cluster-manager is the REST writer."},
		{Name: tagVolumes, Description: "Durable data-volume lifecycle (PersistentVolumeClaim create/list/get/expand/delete) with mount-occupancy reporting; idempotent create/delete."},
		{Name: tagCapabilities, Description: "Deployment-form capability document (multi-tenant / writable resource pools)."},
		{Name: tagHealth, Description: "Liveness and readiness probes."},
	}

	poolParam := openapigen.PathParam("pool", "ResourcePool name.")
	unitParam := openapigen.PathParam("unit", "ResourceUnit name (within a pool).")
	tenantParam := openapigen.PathParam("tenant", "Tenant name (== identifier == namespace).")
	quotaPoolParam := openapigen.PathParam("pool", "ResourcePool the quota applies to.")
	volNamespaceParam := openapigen.PathParam("namespace", "Physical Kubernetes namespace holding the volume.")
	volNameParam := openapigen.PathParam("name", "Volume (PersistentVolumeClaim) name.")
	selectorParam := openapigen.QueryParam("labelSelector", "K8s-style label selector.", &openapigen.Schema{Type: "string"})
	volNamespaceQuery := openapigen.QueryParam("namespace", "Physical Kubernetes namespace to list volumes in.", &openapigen.Schema{Type: "string"})
	forceQuery := openapigen.QueryParam("force", "Delete even when mounted by running workloads.", &openapigen.Schema{Type: "boolean"})

	paths := map[string]openapigen.PathItem{}

	paths["/healthz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagHealth}, Summary: "Liveness probe", OperationID: "healthz",
		Responses: map[string]openapigen.Response{"200": {Description: "ok"}},
	}}
	paths["/readyz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagHealth}, Summary: "Readiness probe", OperationID: "readyz",
		Responses: map[string]openapigen.Response{"200": {Description: "ok"}},
	}}

	paths["/api/v1/capabilities"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagCapabilities}, Summary: "Get deployment-form capabilities", OperationID: "getCapabilities",
		Responses: map[string]openapigen.Response{"200": openapigen.JSONResp("Capability document.", "Capabilities")},
	}}

	paths["/api/v1/resourcepools"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Create a ResourcePool", OperationID: "createResourcePool",
			RequestBody: openapigen.JSONBody("CreateResourcePoolRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("ResourcePool created.", "ResourcePool")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "List ResourcePools", OperationID: "listResourcePools",
			Parameters: []openapigen.Parameter{selectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("ResourcePool page.", "ResourcePoolList")}),
		},
	}

	paths["/api/v1/resourcepools/{pool}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Get ResourcePool", OperationID: "getResourcePool",
			Parameters: []openapigen.Parameter{poolParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("ResourcePool.", "ResourcePool")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Patch ResourcePool", OperationID: "updateResourcePool",
			Parameters:  []openapigen.Parameter{poolParam},
			RequestBody: openapigen.JSONBody("PatchResourcePoolRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated ResourcePool.", "ResourcePool")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Delete ResourcePool", OperationID: "deleteResourcePool",
			Parameters: []openapigen.Parameter{poolParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/resourcepools/{pool}/units"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Add a unit to the pool", OperationID: "createResourceUnit",
			Parameters:  []openapigen.Parameter{poolParam},
			RequestBody: openapigen.JSONBody("CreateResourceUnitRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Unit created.", "ResourceUnit")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "List units in a pool", OperationID: "listResourceUnits",
			Parameters: []openapigen.Parameter{poolParam, selectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Unit page.", "ResourceUnitList")}),
		},
	}

	paths["/api/v1/resourcepools/{pool}/units/{unit}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Get unit", OperationID: "getResourceUnit",
			Parameters: []openapigen.Parameter{poolParam, unitParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Unit.", "ResourceUnit")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Patch unit", OperationID: "updateResourceUnit",
			Parameters:  []openapigen.Parameter{poolParam, unitParam},
			RequestBody: openapigen.JSONBody("PatchResourceUnitRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated unit.", "ResourceUnit")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Remove unit from pool", OperationID: "deleteResourceUnit",
			Parameters: []openapigen.Parameter{poolParam, unitParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/tenants"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Create a Tenant", OperationID: "createTenant",
			RequestBody: openapigen.JSONBody("CreateTenantRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Tenant created.", "Tenant")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "List Tenants", OperationID: "listTenants",
			Parameters: []openapigen.Parameter{selectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Tenant page.", "TenantList")}),
		},
	}

	paths["/api/v1/tenants/{tenant}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Get Tenant", OperationID: "getTenant",
			Parameters: []openapigen.Parameter{tenantParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Tenant.", "Tenant")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Patch Tenant", OperationID: "updateTenant",
			Parameters:  []openapigen.Parameter{tenantParam},
			RequestBody: openapigen.JSONBody("PatchTenantRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated Tenant.", "Tenant")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Delete Tenant", OperationID: "deleteTenant",
			Parameters: []openapigen.Parameter{tenantParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/tenants/{tenant}/quotas"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "List tenant quotas", OperationID: "listTenantQuotas",
			Parameters: []openapigen.Parameter{tenantParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Quota list.", "QuotaList")}),
		},
		Post: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Create or replace a pool quota", OperationID: "setTenantQuota",
			Parameters:  []openapigen.Parameter{tenantParam},
			RequestBody: openapigen.JSONBody("SetQuotaRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Quota set.", "Quota")}),
		},
	}

	paths["/api/v1/tenants/{tenant}/quotas/{pool}"] = openapigen.PathItem{
		Patch: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Update a pool quota", OperationID: "updateTenantQuota",
			Parameters:  []openapigen.Parameter{tenantParam, quotaPoolParam},
			RequestBody: openapigen.JSONBody("PatchQuotaRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated quota.", "Quota")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Delete a pool quota", OperationID: "deleteTenantQuota",
			Parameters: []openapigen.Parameter{tenantParam, quotaPoolParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/volumes"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagVolumes}, Summary: "Materialise a durable volume", OperationID: "createVolume",
			RequestBody: openapigen.JSONBody("CreateVolumeRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Volume materialised (idempotent).", "Volume")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagVolumes}, Summary: "List durable volumes", OperationID: "listVolumes",
			Parameters: []openapigen.Parameter{volNamespaceQuery, selectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Volume page (with live phase/capacity).", "VolumeList")}),
		},
	}

	paths["/api/v1/storageclasses"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagVolumes}, Summary: "List storage classes", OperationID: "listStorageClasses",
			Responses: withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Available storage classes for new volumes.", "StorageClassList")}),
		},
	}

	paths["/api/v1/volumes/{namespace}/{name}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagVolumes}, Summary: "Get a durable volume", OperationID: "getVolume",
			Parameters: []openapigen.Parameter{volNamespaceParam, volNameParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Volume detail (with mount occupancy).", "Volume")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagVolumes}, Summary: "Expand or relabel a durable volume", OperationID: "updateVolume",
			Parameters:  []openapigen.Parameter{volNamespaceParam, volNameParam},
			RequestBody: openapigen.JSONBody("PatchVolumeRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated volume.", "Volume")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagVolumes}, Summary: "Delete a durable volume", OperationID: "deleteVolume",
			Parameters: []openapigen.Parameter{volNamespaceParam, volNameParam, forceQuery},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Cluster Manager API",
			Version:     version,
			Description: "Stateless REST shell over the ResourcePool and Tenant CRDs (cluster-scoped) plus durable volume (PVC) materialisation. RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}
