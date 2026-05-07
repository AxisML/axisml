// openapi-gen renders an OpenAPI 3.0.3 description of the cluster-manager
// HTTP API to docs/openapi/cluster-manager.yaml at the repo root.
//
// Schemas are derived from the request/response types in internal/server,
// which mirror the Tenant CR shape. Routes are listed explicitly here
// (single source of truth) rather than scraped from the gin router.
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
	tagTenants = "tenants"
	tagQuotas  = "quotas"
	tagSystem  = "system"
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

// withErrors returns the standard Problem-bearing error responses.
// cluster-manager statusFor (writeProblem + writeAPIErr in tenant/handler.go)
// emits 400, 403, 404, 409, 500 — no 412/422/503 path; 401 only via gateway.
func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "Problem"),
		"403":     openapigen.JSONResp("Forbidden.", "Problem"),
		"404":     openapigen.JSONResp("Not found.", "Problem"),
		"409":     openapigen.JSONResp("Conflict (already exists or concurrent modification).", "Problem"),
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
	g.Register("CreateTenantRequest", server.CreateTenantRequest{}, openapigen.InputMode)
	g.Register("PatchTenantRequest", server.PatchTenantRequest{}, openapigen.InputMode)
	g.Register("QuotaSpec", server.QuotaSpec{}, openapigen.InputMode)
	g.Register("TenantResponse", server.TenantResponse{}, openapigen.ResponseMode)
	g.Register("ListTenantsResponse", server.ListTenantsResponse{}, openapigen.ResponseMode)

	// ListQuotas returns a synthetic envelope (gin.H{"spec": ..., "status": ...})
	// rather than a typed struct. Define it explicitly so client codegen sees a
	// concrete shape.
	g.Set("QuotaListResponse", &openapigen.Schema{
		Type:     "object",
		Required: []string{"spec", "status"},
		Properties: map[string]*openapigen.Schema{
			"spec":   {Type: "array", Items: openapigen.Ref("QuotaSpec")},
			"status": {Type: "array", Items: openapigen.Ref("ServerQuotaStatus")},
		},
	})

	tags := []openapigen.TagEntry{
		{Name: tagTenants, Description: "Tenant CRUD over the Tenant CR via the K8s API."},
		{Name: tagQuotas, Description: "Per-tenant ElasticQuota declarations (spec.quotas[])."},
		{Name: tagSystem, Description: "Liveness and readiness probes."},
	}

	nameParam := openapigen.PathParam("name", "Tenant name (metadata.name).")
	poolParam := openapigen.PathParam("pool", "Resource pool name.")
	quotaParam := openapigen.PathParam("quota", "Quota name.")

	limitParam := openapigen.QueryParam("limit", "Page size for the K8s LIST call.", openapigen.IntFormat32Param())
	continueParam := openapigen.QueryParam("continue", "K8s LIST continue token from a prior page.", &openapigen.Schema{Type: "string"})

	paths := map[string]openapigen.PathItem{}

	paths["/healthz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagSystem}, Summary: "Liveness probe", OperationID: "healthz",
		Responses: map[string]openapigen.Response{"200": {Description: "ok"}},
	}}
	paths["/readyz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagSystem}, Summary: "Readiness probe", OperationID: "readyz",
		Responses: map[string]openapigen.Response{"200": {Description: "ok"}},
	}}

	paths["/api/v1/tenants"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Create a tenant", OperationID: "createTenant",
			RequestBody: openapigen.JSONBody("CreateTenantRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Tenant created.", "TenantResponse")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "List tenants", OperationID: "listTenants",
			Parameters: []openapigen.Parameter{limitParam, continueParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Tenant page.", "ListTenantsResponse")}),
		},
	}
	paths["/api/v1/tenants/{name}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Get tenant", OperationID: "getTenant",
			Parameters: []openapigen.Parameter{nameParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Tenant.", "TenantResponse")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Patch tenant", OperationID: "patchTenant",
			Parameters:  []openapigen.Parameter{nameParam},
			RequestBody: openapigen.JSONBody("PatchTenantRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated tenant.", "TenantResponse")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Delete tenant", OperationID: "deleteTenant",
			Parameters: []openapigen.Parameter{nameParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}
	paths["/api/v1/tenants/{name}/suspend"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTenants}, Summary: "Suspend tenant", OperationID: "suspendTenant",
		Parameters: []openapigen.Parameter{nameParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Suspended.", "TenantResponse")}),
	}}
	paths["/api/v1/tenants/{name}/unsuspend"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTenants}, Summary: "Unsuspend tenant", OperationID: "unsuspendTenant",
		Parameters: []openapigen.Parameter{nameParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Unsuspended.", "TenantResponse")}),
	}}

	paths["/api/v1/tenants/{name}/quotas"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagQuotas}, Summary: "Add a quota to a tenant", OperationID: "addQuota",
			Parameters:  []openapigen.Parameter{nameParam},
			RequestBody: openapigen.JSONBody("QuotaSpec"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated tenant.", "TenantResponse")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagQuotas}, Summary: "List quotas of a tenant", OperationID: "listQuotas",
			Parameters: []openapigen.Parameter{nameParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Quota spec + status.", "QuotaListResponse")}),
		},
	}
	paths["/api/v1/tenants/{name}/quotas/{pool}/{quota}"] = openapigen.PathItem{
		Patch: &openapigen.Operation{
			Tags: []string{tagQuotas}, Summary: "Update a quota", OperationID: "updateQuota",
			Parameters:  []openapigen.Parameter{nameParam, poolParam, quotaParam},
			RequestBody: openapigen.JSONBody("QuotaSpec"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated tenant.", "TenantResponse")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagQuotas}, Summary: "Delete a quota", OperationID: "deleteQuota",
			Parameters: []openapigen.Parameter{nameParam, poolParam, quotaParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Cluster Manager API",
			Version:     version,
			Description: "Stateless REST shell over Tenant CR CRUD. Forwards to the Kubernetes API server. RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}
