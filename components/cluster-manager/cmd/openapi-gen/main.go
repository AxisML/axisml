// openapi-gen renders an OpenAPI 3.0.3 description of the cluster-manager
// HTTP API to docs/openapi/cluster-manager.yaml at the repo root.
//
// The cluster-manager exposes ResourcePool CRD CRUD (with embedded
// spec.units[]); request/response types are mirrored from internal/server.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../../docs/openapi/cluster-manager.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axisml/axisml/components/cluster-manager/internal/server"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

const (
	tagResourcePools = "resource-pools"
	tagResourceUnits = "resource-units"
	tagTenants       = "tenants"
	tagTenantQuotas  = "tenant-quotas"
	tagSystem        = "system"
)

func main() {
	out := flag.String("o", "../../docs/openapi/cluster-manager.yaml", "output path")
	v := flag.String("version", defaultVersion, "info.version field")
	flag.Parse()

	doc := buildDocument(*v)
	data, err := openapigen.MarshalYAML(doc)
	if err != nil {
		fail("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail("write: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "openapi-gen: "+format+"\n", args...)
	os.Exit(1)
}

func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "Problem"),
		"401":     openapigen.JSONResp("Missing or invalid X-Axisml-User.", "Problem"),
		"404":     openapigen.JSONResp("Not found.", "Problem"),
		"409":     openapigen.JSONResp("Conflict (already exists or concurrent modification).", "Problem"),
		"422":     openapigen.JSONResp("Invariant violated.", "Problem"),
		"500":     openapigen.JSONResp("Internal error.", "Problem"),
		"default": openapigen.JSONResp("Unexpected error.", "Problem"),
	}
	for k, v := range success {
		out[k] = v
	}
	return out
}

func buildDocument(version string) *openapigen.Document {
	g := openapigen.New(openapigen.Options{})

	g.Register("Problem", server.Problem{}, openapigen.ResponseMode)
	g.Register("CreateResourcePoolRequest", server.CreateResourcePoolRequest{}, openapigen.InputMode)
	g.Register("PatchResourcePoolRequest", server.PatchResourcePoolRequest{}, openapigen.InputMode)
	g.Register("CreateResourceUnitRequest", server.CreateResourceUnitRequest{}, openapigen.InputMode)
	g.Register("PatchResourceUnitRequest", server.PatchResourceUnitRequest{}, openapigen.InputMode)
	g.Register("ResourcePoolDTO", server.ResourcePoolDTO{}, openapigen.ResponseMode)
	g.Register("ResourceUnitDTO", server.ResourceUnitDTO{}, openapigen.ResponseMode)
	g.Register("ResourcePoolList", server.ResourcePoolList{}, openapigen.ResponseMode)
	g.Register("ResourceUnitList", server.ResourceUnitList{}, openapigen.ResponseMode)
	g.Register("CreateTenantRequest", server.CreateTenantRequest{}, openapigen.InputMode)
	g.Register("PatchTenantRequest", server.PatchTenantRequest{}, openapigen.InputMode)
	g.Register("SetQuotaRequest", server.SetQuotaRequest{}, openapigen.InputMode)
	g.Register("PatchQuotaRequest", server.PatchQuotaRequest{}, openapigen.InputMode)
	g.Register("TenantDTO", server.TenantDTO{}, openapigen.ResponseMode)
	g.Register("TenantList", server.TenantList{}, openapigen.ResponseMode)
	g.Register("QuotaDTO", server.QuotaDTO{}, openapigen.ResponseMode)
	g.Register("QuotaList", server.QuotaList{}, openapigen.ResponseMode)

	tags := []openapigen.TagEntry{
		{Name: tagResourcePools, Description: "ResourcePool CRD CRUD."},
		{Name: tagResourceUnits, Description: "Sub-routes over pool.spec.units[]."},
		{Name: tagTenants, Description: "Tenant CRD CRUD (cluster-manager is the REST writer)."},
		{Name: tagTenantQuotas, Description: "Per-pool tenant quotas (unit × quantity, folded to ElasticQuota)."},
		{Name: tagSystem, Description: "Liveness and readiness probes."},
	}

	poolParam := openapigen.PathParam("pool", "ResourcePool name.")
	unitParam := openapigen.PathParam("unit", "ResourceUnit name (within a pool).")
	tenantParam := openapigen.PathParam("tenant", "Tenant name (== identifier == namespace).")
	quotaPoolParam := openapigen.PathParam("pool", "ResourcePool the quota applies to.")
	selectorParam := openapigen.QueryParam("labelSelector", "K8s-style label selector.", &openapigen.Schema{Type: "string"})

	paths := map[string]openapigen.PathItem{}

	paths["/healthz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagSystem}, Summary: "Liveness probe", OperationID: "healthz",
		Responses: map[string]openapigen.Response{"200": {Description: "ok"}},
	}}
	paths["/readyz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagSystem}, Summary: "Readiness probe", OperationID: "readyz",
		Responses: map[string]openapigen.Response{"200": {Description: "ok"}},
	}}

	paths["/api/v1/resource-pools"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Create a ResourcePool", OperationID: "createResourcePool",
			RequestBody: openapigen.JSONBody("CreateResourcePoolRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("ResourcePool created.", "ResourcePoolDTO")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "List ResourcePools", OperationID: "listResourcePools",
			Parameters: []openapigen.Parameter{selectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("ResourcePool page.", "ResourcePoolList")}),
		},
	}

	paths["/api/v1/resource-pools/{pool}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Get ResourcePool", OperationID: "getResourcePool",
			Parameters: []openapigen.Parameter{poolParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("ResourcePool.", "ResourcePoolDTO")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Patch ResourcePool", OperationID: "updateResourcePool",
			Parameters:  []openapigen.Parameter{poolParam},
			RequestBody: openapigen.JSONBody("PatchResourcePoolRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated ResourcePool.", "ResourcePoolDTO")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Delete ResourcePool", OperationID: "deleteResourcePool",
			Parameters: []openapigen.Parameter{poolParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/resource-pools/{pool}/units"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Add a unit to the pool", OperationID: "createResourceUnit",
			Parameters:  []openapigen.Parameter{poolParam},
			RequestBody: openapigen.JSONBody("CreateResourceUnitRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Unit created.", "ResourceUnitDTO")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "List units in a pool", OperationID: "listResourceUnits",
			Parameters: []openapigen.Parameter{poolParam, selectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Unit page.", "ResourceUnitList")}),
		},
	}

	paths["/api/v1/resource-pools/{pool}/units/{unit}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Get unit", OperationID: "getResourceUnit",
			Parameters: []openapigen.Parameter{poolParam, unitParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Unit.", "ResourceUnitDTO")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Patch unit", OperationID: "updateResourceUnit",
			Parameters:  []openapigen.Parameter{poolParam, unitParam},
			RequestBody: openapigen.JSONBody("PatchResourceUnitRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated unit.", "ResourceUnitDTO")}),
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
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Tenant created.", "TenantDTO")}),
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
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Tenant.", "TenantDTO")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Patch Tenant", OperationID: "updateTenant",
			Parameters:  []openapigen.Parameter{tenantParam},
			RequestBody: openapigen.JSONBody("PatchTenantRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated Tenant.", "TenantDTO")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Delete Tenant", OperationID: "deleteTenant",
			Parameters: []openapigen.Parameter{tenantParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/tenants/{tenant}/quotas"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTenantQuotas}, Summary: "List tenant quotas", OperationID: "listTenantQuotas",
			Parameters: []openapigen.Parameter{tenantParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Quota list.", "QuotaList")}),
		},
		Post: &openapigen.Operation{
			Tags: []string{tagTenantQuotas}, Summary: "Create or replace a pool quota", OperationID: "setTenantQuota",
			Parameters:  []openapigen.Parameter{tenantParam},
			RequestBody: openapigen.JSONBody("SetQuotaRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Quota set.", "QuotaDTO")}),
		},
	}

	paths["/api/v1/tenants/{tenant}/quotas/{pool}"] = openapigen.PathItem{
		Patch: &openapigen.Operation{
			Tags: []string{tagTenantQuotas}, Summary: "Update a pool quota", OperationID: "updateTenantQuota",
			Parameters:  []openapigen.Parameter{tenantParam, quotaPoolParam},
			RequestBody: openapigen.JSONBody("PatchQuotaRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated quota.", "QuotaDTO")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenantQuotas}, Summary: "Delete a pool quota", OperationID: "deleteTenantQuota",
			Parameters: []openapigen.Parameter{tenantParam, quotaPoolParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Cluster Manager API",
			Version:     version,
			Description: "Stateless REST shell over the ResourcePool and Tenant CRDs (cluster-scoped). RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}
