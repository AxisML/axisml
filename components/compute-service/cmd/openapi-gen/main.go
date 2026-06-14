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

	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/components/compute-service/internal/mlrun"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/mlservice"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
	trafficpolicymod "github.com/axisml/axisml/components/compute-service/internal/trafficpolicy"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

// Tag names. One source of truth so a typo can't silently split a group.
const (
	tagTenants         = "tenants"
	tagMLRuns          = "mlruns"
	tagMLServices      = "mlservices"
	tagTrafficPolicies = "traffic-policies"
	tagSystem          = "system"
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
	g.Register("MLRunCreateInput", mlrun.CreateInput{}, openapigen.InputMode)
	g.Register("MLRunPatchInput", mlrun.PatchInput{}, openapigen.InputMode)
	g.Register("MLRunView", mlrun.View{}, openapigen.ResponseMode)
	g.Register("MLServiceCreateInput", servicemod.CreateInput{}, openapigen.InputMode)
	g.Register("MLServicePatchInput", servicemod.PatchInput{}, openapigen.InputMode)
	g.Register("MLServiceScaleInput", servicemod.ScaleInput{}, openapigen.InputMode)
	g.Register("MLServiceView", servicemod.View{}, openapigen.ResponseMode)
	g.Register("TrafficPolicyCreateInput", trafficpolicymod.CreateInput{}, openapigen.InputMode)
	g.Register("TrafficPolicyPatchInput", trafficpolicymod.PatchInput{}, openapigen.InputMode)
	g.Register("TrafficPolicySplitInput", trafficpolicymod.SplitInput{}, openapigen.InputMode)
	g.Register("TrafficPolicyView", trafficpolicymod.View{}, openapigen.ResponseMode)
	g.Register("PodView", kubeproxy.PodView{}, openapigen.ResponseMode)
	g.Register("EventView", kubeproxy.EventView{}, openapigen.ResponseMode)

	g.Set("MLRunList", openapigen.ListEnvelope("MLRunView"))
	g.Set("MLServiceList", openapigen.ListEnvelope("MLServiceView"))
	g.Set("TrafficPolicyList", openapigen.ListEnvelope("TrafficPolicyView"))
	g.Set("PodList", openapigen.ListEnvelope("PodView"))
	g.Set("EventList", openapigen.ListEnvelope("EventView"))
	g.Set("TenantQuotaList", openapigen.ListEnvelope("TenantQuota"))

	tags := []openapigen.TagEntry{
		{Name: tagTenants, Description: "Tenant CRUD. Compute owns the Tenant CR; PG is authoritative, CR is derived."},
		{Name: tagMLRuns, Description: "MLRun CRUD per namespace. ResourcePool/Unit referenced by name (read from K8s Informer cache)."},
		{Name: tagMLServices, Description: "MLService CRUD per namespace."},
		{Name: tagTrafficPolicies, Description: "MLTrafficPolicy CRUD per namespace: weighted / canary / blue-green traffic split over member online services."},
		{Name: tagSystem, Description: "Liveness and readiness probes."},
	}

	nsParam := openapigen.PathParam("namespace", "Tenant name (= mlruns/mlservices partition key).")
	mlrunParam := openapigen.PathParam("mlrun", "MLRun name.")
	mlserviceParam := openapigen.PathParam("mlservice", "MLService name.")
	policyParam := openapigen.PathParam("policy", "Traffic policy name.")

	limitParam := openapigen.QueryParam("limit", "Page size (1–200, default 50).", openapigen.IntFormat32Param())
	continueParam := openapigen.QueryParam("continue", "Opaque continuation token from a previous page.", &openapigen.Schema{Type: "string"})
	labelSelectorParam := openapigen.QueryParam("labelSelector", "K8s-style label selector filtered against the row's labels jsonb.", &openapigen.Schema{Type: "string"})

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
			Parameters: []openapigen.Parameter{limitParam, continueParam, labelSelectorParam},
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
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Soft-deleted tenant (phase=Deleting).", "TenantResponse")}),
		},
	}

	paths["/api/v1/namespaces/{namespace}/restore"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTenants}, Summary: "Restore a soft-deleted tenant", OperationID: "restoreTenant",
		Parameters: []openapigen.Parameter{nsParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Restored tenant.", "TenantResponse")}),
	}}

	poolParam := openapigen.PathParam("pool", "Pool name.")
	quotaNameParam := openapigen.PathParam("quotaName", "Quota name (within the (tenant, pool)).")

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
	paths["/api/v1/namespaces/{namespace}/quotas/{pool}/{quotaName}"] = openapigen.PathItem{
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

	// mlruns (per namespace)
	paths["/api/v1/namespaces/{namespace}/mlruns"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "Submit an MLRun", OperationID: "createMLRun",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("MLRunCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "MLRunView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "List MLRuns in a namespace", OperationID: "listMLRuns",
			Parameters: []openapigen.Parameter{nsParam, limitParam, continueParam, labelSelectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "MLRunList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "Get MLRun", OperationID: "getMLRun",
			Parameters: []openapigen.Parameter{nsParam, mlrunParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("MLRun.", "MLRunView")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "Patch MLRun display fields", OperationID: "patchMLRun",
			Parameters:  []openapigen.Parameter{nsParam, mlrunParam},
			RequestBody: openapigen.JSONBody("MLRunPatchInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Patched MLRun.", "MLRunView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "Delete MLRun", OperationID: "deleteMLRun",
			Parameters: []openapigen.Parameter{nsParam, mlrunParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/cancel"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "Cancel a running MLRun", OperationID: "cancelMLRun",
		Parameters: []openapigen.Parameter{nsParam, mlrunParam},
		Responses:  withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Cancellation queued (row is Canceling).", "MLRunView")}),
	}}

	podParam := openapigen.PathParam("pod", "Pod name.")
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "List pods of an MLRun", OperationID: "listMLRunPods",
		Parameters: []openapigen.Parameter{nsParam, mlrunParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Pods.", "PodList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods/{pod}/logs"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "Stream a pod's container log", OperationID: "getMLRunPodLogs",
		Parameters: []openapigen.Parameter{nsParam, mlrunParam, podParam},
		Responses: withErrors(map[string]openapigen.Response{
			"200": openapigen.StringResp("text/plain stream of pod log"),
		}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods/{pod}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "List events targeted at a pod", OperationID: "listMLRunPodEvents",
		Parameters: []openapigen.Parameter{nsParam, mlrunParam, podParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "List events targeted at the MLRun CR", OperationID: "listMLRunEvents",
		Parameters: []openapigen.Parameter{nsParam, mlrunParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}

	// mlservices (per namespace)
	paths["/api/v1/namespaces/{namespace}/mlservices"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagMLServices}, Summary: "Submit an MLService", OperationID: "createMLService",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("MLServiceCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "MLServiceView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagMLServices}, Summary: "List MLServices in a namespace", OperationID: "listMLServices",
			Parameters: []openapigen.Parameter{nsParam, limitParam, continueParam, labelSelectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "MLServiceList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagMLServices}, Summary: "Get MLService", OperationID: "getMLService",
			Parameters: []openapigen.Parameter{nsParam, mlserviceParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("MLService.", "MLServiceView")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagMLServices}, Summary: "Patch MLService display fields", OperationID: "patchMLService",
			Parameters:  []openapigen.Parameter{nsParam, mlserviceParam},
			RequestBody: openapigen.JSONBody("MLServicePatchInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Patched service.", "MLServiceView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagMLServices}, Summary: "Delete MLService", OperationID: "deleteMLService",
			Parameters: []openapigen.Parameter{nsParam, mlserviceParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/scale"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "Scale an MLService", OperationID: "scaleMLService",
		Parameters:  []openapigen.Parameter{nsParam, mlserviceParam},
		RequestBody: openapigen.JSONBody("MLServiceScaleInput"),
		Responses:   withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Scale queued (generation bumped).", "MLServiceView")}),
	}}

	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "List pods of an MLService", OperationID: "listMLServicePods",
		Parameters: []openapigen.Parameter{nsParam, mlserviceParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Pods.", "PodList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods/{pod}/logs"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "Stream an MLService pod's container log", OperationID: "getMLServicePodLogs",
		Parameters: []openapigen.Parameter{nsParam, mlserviceParam, podParam},
		Responses: withErrors(map[string]openapigen.Response{
			"200": openapigen.StringResp("text/plain stream of pod log"),
		}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods/{pod}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "List events targeted at an MLService pod", OperationID: "listMLServicePodEvents",
		Parameters: []openapigen.Parameter{nsParam, mlserviceParam, podParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/events"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "List events targeted at the MLService CR", OperationID: "listMLServiceEvents",
		Parameters: []openapigen.Parameter{nsParam, mlserviceParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Events.", "EventList")}),
	}}

	// traffic policies (per namespace)
	paths["/api/v1/namespaces/{namespace}/traffic-policies"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagTrafficPolicies}, Summary: "Create a traffic policy", OperationID: "createTrafficPolicy",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("TrafficPolicyCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "TrafficPolicyView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagTrafficPolicies}, Summary: "List traffic policies in a namespace", OperationID: "listTrafficPolicies",
			Parameters: []openapigen.Parameter{nsParam, limitParam, continueParam, labelSelectorParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "TrafficPolicyList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/traffic-policies/{policy}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagTrafficPolicies}, Summary: "Get traffic policy", OperationID: "getTrafficPolicy",
			Parameters: []openapigen.Parameter{nsParam, policyParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Traffic policy.", "TrafficPolicyView")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagTrafficPolicies}, Summary: "Patch traffic policy display fields", OperationID: "patchTrafficPolicy",
			Parameters:  []openapigen.Parameter{nsParam, policyParam},
			RequestBody: openapigen.JSONBody("TrafficPolicyPatchInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Patched traffic policy.", "TrafficPolicyView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagTrafficPolicies}, Summary: "Delete traffic policy (members retained)", OperationID: "deleteTrafficPolicy",
			Parameters: []openapigen.Parameter{nsParam, policyParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/traffic-policies/{policy}/split"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTrafficPolicies}, Summary: "Adjust per-backend weights", OperationID: "splitTrafficPolicy",
		Parameters:  []openapigen.Parameter{nsParam, policyParam},
		RequestBody: openapigen.JSONBody("TrafficPolicySplitInput"),
		Responses:   withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Split queued (generation bumped).", "TrafficPolicyView")}),
	}}
	paths["/api/v1/namespaces/{namespace}/traffic-policies/{policy}/promote"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTrafficPolicies}, Summary: "Promote the canary to stable (canary mode)", OperationID: "promoteTrafficPolicy",
		Parameters: []openapigen.Parameter{nsParam, policyParam},
		Responses:  withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Promote queued (generation bumped).", "TrafficPolicyView")}),
	}}
	paths["/api/v1/namespaces/{namespace}/traffic-policies/{policy}/rollback"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTrafficPolicies}, Summary: "Roll the canary back to 0 (canary mode)", OperationID: "rollbackTrafficPolicy",
		Parameters: []openapigen.Parameter{nsParam, policyParam},
		Responses:  withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Rollback queued (generation bumped).", "TrafficPolicyView")}),
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
// (.../compute-operator/api/{mlrun,mlservice}/v1alpha1) to per-CRD prefixes
// so MLRunSpec / MLServiceSpec don't collide on a shared "v1alpha1" segment.
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
	case "mlrun":
		return "MLRun", true
	case "mlservice":
		return "MLService", true
	case "mltrafficpolicy":
		return "MLTrafficPolicy", true
	}
	return "", false
}
