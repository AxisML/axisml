// openapi-gen renders an OpenAPI 3.0.3 description of the compute HTTP API
// to docs/openapi/compute-service.yaml at the repo root.
//
// Schemas are derived from the same Go request/response structs the runtime
// handlers use, so the spec stays in lock-step with the code. Routes are
// listed explicitly here (single source of truth) rather than scraped from
// the gin router so the file is reviewable.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../../docs/openapi/compute-service.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/axisml/axisml/components/compute-service/internal/job"
	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/service"
	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

// Tag names. One source of truth so a typo can't silently split a group.
const (
	tagTenants  = "tenants"
	tagJobs     = "jobs"
	tagServices = "services"
	tagSystem   = "system"
)

// AxisML §6.1 name policy is duplicated here as a regex rather than imported
// because the generator's contract is "render whatever clients need to send";
// clients don't import strutil. Same constants live in
// components/compute-service/pkg/strutil — keep them in sync if the policy changes.
const (
	axisMLNamePattern = "^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$"
)

func main() {
	out := flag.String("o", "../../docs/openapi/compute-service.yaml", "output path")
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

// withErrors returns the standard Problem-bearing error responses with the
// given success codes overlaid. Codes here must stay in lock-step with
// internal/server/problem.go's statusFor.
func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "Problem"),
		"401":     openapigen.JSONResp("Unauthorized.", "Problem"),
		"403":     openapigen.JSONResp("Forbidden.", "Problem"),
		"404":     openapigen.JSONResp("Not found.", "Problem"),
		"409":     openapigen.JSONResp("Conflict.", "Problem"),
		"412":     openapigen.JSONResp("Precondition failed.", "Problem"),
		"422":     openapigen.JSONResp("Quota exceeded.", "Problem"),
		"503":     openapigen.JSONResp("Service unavailable.", "Problem"),
		"default": openapigen.JSONResp("Unexpected error.", "Problem"),
	}
	for k, v := range success {
		out[k] = v
	}
	return out
}

func buildDocument(version string) *openapigen.Document {
	apperrorsCodeType := reflect.TypeOf(apperrors.Code(""))

	g := openapigen.New(openapigen.Options{
		WellKnown: func(t reflect.Type) *openapigen.Schema {
			if t == apperrorsCodeType {
				return &openapigen.Schema{
					Type:        "string",
					Description: "Discrete business error class.",
					Enum:        apperrors.AllCodes(),
				}
			}
			return nil
		},
		PatternRules: []openapigen.PatternRule{
			{Tag: "axisml_name", Pattern: axisMLNamePattern, MinLength: 3, MaxLength: 40},
		},
		PackageNamer: operatorAPIPrefix,
	})

	// Core component schemas (referenced from operations).
	g.Register("Problem", server.Problem{}, openapigen.ResponseMode)
	g.Register("TenantCreateInput", tenantmod.CreateInput{}, openapigen.InputMode)
	g.Register("TenantPatchInput", tenantmod.PatchInput{}, openapigen.InputMode)
	g.Register("TenantResponse", tenantmod.Response{}, openapigen.ResponseMode)
	g.Register("TenantListResponse", tenantmod.ListResponse{}, openapigen.ResponseMode)
	g.Register("TenantQuotaInput", tenantmod.QuotaPatchInput{}, openapigen.InputMode)
	g.Register("TenantQuota", tenantmod.QuotaSpec{}, openapigen.ResponseMode)
	g.Register("JobCreateInput", job.CreateInput{}, openapigen.InputMode)
	g.Register("JobView", job.View{}, openapigen.ResponseMode)
	g.Register("MLServiceCreateInput", servicemod.CreateInput{}, openapigen.InputMode)
	g.Register("MLServiceScaleInput", servicemod.ScaleInput{}, openapigen.InputMode)
	g.Register("MLServiceView", servicemod.View{}, openapigen.ResponseMode)
	g.Register("PodView", kubeproxy.PodView{}, openapigen.ResponseMode)
	g.Register("EventView", kubeproxy.EventView{}, openapigen.ResponseMode)

	g.Set("JobList", openapigen.ListEnvelope("JobView"))
	g.Set("MLServiceList", openapigen.ListEnvelope("MLServiceView"))
	g.Set("PodList", openapigen.ListEnvelope("PodView"))
	g.Set("EventList", openapigen.ListEnvelope("EventView"))
	g.Set("TenantQuotaList", openapigen.ListEnvelope("TenantQuota"))

	tags := []openapigen.TagEntry{
		{Name: tagTenants, Description: "Tenant CRUD. Compute owns the Tenant CR; PG is authoritative, CR is derived."},
		{Name: tagJobs, Description: "MLJob CRUD per namespace. ResourcePool/Unit referenced by name (read from K8s Informer cache)."},
		{Name: tagServices, Description: "MLService CRUD per namespace."},
		{Name: tagSystem, Description: "Liveness and readiness probes."},
	}

	nsParam := openapigen.PathParam("namespace", "Tenant name (= jobs/services partition key).")
	jobParam := openapigen.PathParam("job", "Job name.")
	serviceParam := openapigen.PathParam("service", "Service name.")

	limitParam := openapigen.QueryParam("limit", "Page size (1–200, default 50).", openapigen.IntFormat32Param())
	offsetParam := openapigen.QueryParam("offset", "Number of items to skip.", openapigen.IntFormat32Param())

	paths := map[string]openapigen.PathItem{}

	// system probes
	paths["/healthz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagSystem}, Summary: "Liveness probe", OperationID: "healthz",
		Responses: map[string]openapigen.Response{"200": openapigen.StringResp("ok")},
	}}
	paths["/readyz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagSystem}, Summary: "Readiness probe", OperationID: "readyz",
		Responses: map[string]openapigen.Response{
			"200": openapigen.StringResp("ok"),
			"503": openapigen.StringResp("dependency not yet ready"),
		},
	}}

	// tenants (the URL token is the tenant name, mirroring the design's
	// "namespace = tenant identifier" naming).
	paths["/api/v1/namespaces"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Create a tenant", OperationID: "createTenant",
			RequestBody: openapigen.JSONBody("TenantCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "TenantResponse")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "List tenants", OperationID: "listTenants",
			Parameters: []openapigen.Parameter{limitParam, offsetParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "TenantListResponse")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Get tenant", OperationID: "getTenant",
			Parameters: []openapigen.Parameter{nsParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Tenant.", "TenantResponse")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Patch tenant", OperationID: "patchTenant",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("TenantPatchInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated.", "TenantResponse")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Delete tenant (soft delete)", OperationID: "deleteTenant",
			Parameters: []openapigen.Parameter{nsParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	paths["/api/v1/namespaces/{namespace}/restore"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTenants}, Summary: "Restore a soft-deleted tenant", OperationID: "restoreTenant",
		Parameters: []openapigen.Parameter{nsParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Restored tenant.", "TenantResponse")}),
	}}

	poolParam := openapigen.PathParam("pool", "Pool name.")
	quotaNameParam := openapigen.PathParam("name", "Quota name (within the (tenant, pool)).")

	paths["/api/v1/namespaces/{namespace}/quotas"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "List quotas of a tenant", OperationID: "listTenantQuotas",
			Parameters: []openapigen.Parameter{nsParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Quotas.", "TenantQuotaList")}),
		},
		Post: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Add a quota to a tenant", OperationID: "addTenantQuota",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("TenantQuotaInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Added.", "TenantQuota")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/quotas/{pool}/{name}"] = openapigen.PathItem{
		Patch: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Patch a quota's min/max", OperationID: "patchTenantQuota",
			Parameters:  []openapigen.Parameter{nsParam, poolParam, quotaNameParam},
			RequestBody: openapigen.JSONBody("TenantQuotaInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated.", "TenantQuota")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTenants}, Summary: "Delete a quota", OperationID: "deleteTenantQuota",
			Parameters: []openapigen.Parameter{nsParam, poolParam, quotaNameParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	// jobs (per namespace)
	paths["/api/v1/namespaces/{namespace}/jobs"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagJobs}, Summary: "Submit an MLJob", OperationID: "createJob",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("JobCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "JobView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagJobs}, Summary: "List jobs in a namespace", OperationID: "listJobs",
			Parameters: []openapigen.Parameter{nsParam, limitParam, offsetParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "JobList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/jobs/{job}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagJobs}, Summary: "Get job", OperationID: "getJob",
			Parameters: []openapigen.Parameter{nsParam, jobParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Job.", "JobView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagJobs}, Summary: "Delete job", OperationID: "deleteJob",
			Parameters: []openapigen.Parameter{nsParam, jobParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/jobs/{job}/cancel"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagJobs}, Summary: "Cancel a running job", OperationID: "cancelJob",
		Parameters: []openapigen.Parameter{nsParam, jobParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Cancelled job.", "JobView")}),
	}}

	podParam := openapigen.PathParam("pod", "Pod name.")
	paths["/api/v1/namespaces/{namespace}/jobs/{job}/pods"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagJobs}, Summary: "List pods of a job", OperationID: "listJobPods",
		Parameters: []openapigen.Parameter{nsParam, jobParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Pods.", "PodList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/jobs/{job}/pods/{pod}/logs"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagJobs}, Summary: "Stream a pod's container log", OperationID: "getJobPodLogs",
		Parameters: []openapigen.Parameter{nsParam, jobParam, podParam},
		Responses: withErrors(map[string]openapigen.Response{
			"200": openapigen.StringResp("text/plain stream of pod log"),
		}),
	}}
	paths["/api/v1/namespaces/{namespace}/jobs/{job}/pods/{pod}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagJobs}, Summary: "List events targeted at a pod", OperationID: "listJobPodEvents",
		Parameters: []openapigen.Parameter{nsParam, jobParam, podParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/jobs/{job}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagJobs}, Summary: "List events targeted at the MLJob CR", OperationID: "listJobEvents",
		Parameters: []openapigen.Parameter{nsParam, jobParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}

	// services (per namespace)
	paths["/api/v1/namespaces/{namespace}/services"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagServices}, Summary: "Submit an MLService", OperationID: "createMLService",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("MLServiceCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "MLServiceView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagServices}, Summary: "List services in a namespace", OperationID: "listMLServices",
			Parameters: []openapigen.Parameter{nsParam, limitParam, offsetParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "MLServiceList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/services/{service}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagServices}, Summary: "Get service", OperationID: "getMLService",
			Parameters: []openapigen.Parameter{nsParam, serviceParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Service.", "MLServiceView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagServices}, Summary: "Delete service", OperationID: "deleteMLService",
			Parameters: []openapigen.Parameter{nsParam, serviceParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/services/{service}/scale"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagServices}, Summary: "Scale a service", OperationID: "scaleMLService",
		Parameters:  []openapigen.Parameter{nsParam, serviceParam},
		RequestBody: openapigen.JSONBody("MLServiceScaleInput"),
		Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Scaled service.", "MLServiceView")}),
	}}

	paths["/api/v1/namespaces/{namespace}/services/{service}/pods"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagServices}, Summary: "List pods of a service", OperationID: "listServicePods",
		Parameters: []openapigen.Parameter{nsParam, serviceParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Pods.", "PodList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/services/{service}/pods/{pod}/logs"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagServices}, Summary: "Stream a service pod's container log", OperationID: "getServicePodLogs",
		Parameters: []openapigen.Parameter{nsParam, serviceParam, podParam},
		Responses: withErrors(map[string]openapigen.Response{
			"200": openapigen.StringResp("text/plain stream of pod log"),
		}),
	}}
	paths["/api/v1/namespaces/{namespace}/services/{service}/pods/{pod}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagServices}, Summary: "List events targeted at a service pod", OperationID: "listServicePodEvents",
		Parameters: []openapigen.Parameter{nsParam, serviceParam, podParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/services/{service}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagServices}, Summary: "List events targeted at the MLService CR", OperationID: "listServiceEvents",
		Parameters: []openapigen.Parameter{nsParam, serviceParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Compute Service API",
			Version:     version,
			Description: "REST API for Tenant CRUD plus per-namespace Jobs and Services. ResourcePool/Unit live in the cluster-manager CRD; compute references them by name. RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}

// operatorAPIPrefix maps the compute-operator's nested API package paths
// (.../compute-operator/api/{mljob,mlservice}/v1alpha1) to per-CRD prefixes
// so MLJobSpec / MLServiceSpec don't collide on a shared "v1alpha1" segment.
func operatorAPIPrefix(pkg string) (string, bool) {
	const root = "components/compute-operator/api/"
	i := strings.Index(pkg, root)
	if i < 0 {
		return "", false
	}
	rest := pkg[i+len(root):]
	parts := strings.Split(rest, "/")
	if len(parts) < 1 {
		return "", false
	}
	switch parts[0] {
	case "mljob":
		return "MLJob", true
	case "mlservice":
		return "MLService", true
	}
	return "", false
}
