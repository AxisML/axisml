// Package apidoc builds the OpenAPI 3.0.3 description of the compute-service
// HTTP API and is the single source of truth for that contract.
//
// Schemas are derived by reflection from the same Go request/response structs
// the runtime handlers use, so the spec stays in lock-step with the code.
// Routes are listed explicitly here rather than scraped from the gin router so
// the document is reviewable.
//
// The component's own cmd/openapi-gen renders Document to
// axisml-system/docs/apis/compute-service.yaml; axisml-core imports Document
// directly to fold the compute surface into the Lite composite spec, without a
// YAML round-trip.
package apidoc

import (
	"reflect"
	"strings"

	"github.com/axisml/axisml/components/compute-service/internal/server"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/pkg/openapigen"
)

// Tag names. One source of truth so a typo can't silently split a group.
const (
	tagMLRuns          = "MLRuns"
	tagMLServices      = "MLServices"
	tagTrafficPolicies = "TrafficPolicies"
	tagCapabilities    = "Capabilities"
	tagHealth          = "Health"
)

// AxisML §6.1 name policy is duplicated here as a regex rather than imported
// because the generator's contract is "render whatever clients need to send";
// clients don't import strutil. Same constants live in
// components/compute-service/pkg/strutil — keep them in sync if the policy changes.
const (
	axisMLNamePattern = "^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$"
)

// withErrors returns the standard Problem-bearing error responses with the
// given success codes overlaid. Codes here must stay in lock-step with
// internal/server/problem.go's statusFor.
func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "ComputeServiceError"),
		"401":     openapigen.JSONResp("Unauthorized.", "ComputeServiceError"),
		"403":     openapigen.JSONResp("Forbidden.", "ComputeServiceError"),
		"404":     openapigen.JSONResp("Not found.", "ComputeServiceError"),
		"409":     openapigen.JSONResp("Conflict.", "ComputeServiceError"),
		"412":     openapigen.JSONResp("Precondition failed.", "ComputeServiceError"),
		"422":     openapigen.JSONResp("Quota exceeded.", "ComputeServiceError"),
		"503":     openapigen.JSONResp("Service unavailable.", "ComputeServiceError"),
		"default": openapigen.JSONResp("Unexpected error.", "ComputeServiceError"),
	}
	for k, v := range success {
		out[k] = v
	}
	return out
}

// Document builds the complete OpenAPI document for the compute-service API.
func Document(version string) *openapigen.Document {
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
		PackageNamer: packageNamer,
	})

	// Core component schemas (referenced from operations). All request /
	// response DTOs live in the flat internal/server package, so the
	// PackageNamer maps it to an empty prefix and the schema name equals the
	// Go type name verbatim.
	g.Register("ComputeServiceError", server.Error{}, openapigen.ResponseMode)
	g.Register("MLRunCreateRequest", server.MLRunCreateRequest{}, openapigen.InputMode)
	g.Register("MLRunPatchRequest", server.MLRunPatchRequest{}, openapigen.InputMode)
	g.Register("MLRun", server.MLRun{}, openapigen.ResponseMode)
	g.Register("MLServiceCreateRequest", server.MLServiceCreateRequest{}, openapigen.InputMode)
	g.Register("MLServicePatchRequest", server.MLServicePatchRequest{}, openapigen.InputMode)
	g.Register("MLServiceScaleRequest", server.MLServiceScaleRequest{}, openapigen.InputMode)
	g.Register("MLService", server.MLService{}, openapigen.ResponseMode)
	g.Register("TrafficPolicyCreateRequest", server.TrafficPolicyCreateRequest{}, openapigen.InputMode)
	g.Register("TrafficPolicyPatchRequest", server.TrafficPolicyPatchRequest{}, openapigen.InputMode)
	g.Register("TrafficPolicySplitRequest", server.TrafficPolicySplitRequest{}, openapigen.InputMode)
	g.Register("TrafficPolicy", server.TrafficPolicy{}, openapigen.ResponseMode)
	g.Register("Pod", server.Pod{}, openapigen.ResponseMode)
	g.Register("Event", server.Event{}, openapigen.ResponseMode)
	g.Register("Capabilities", server.Capabilities{}, openapigen.ResponseMode)

	g.Set("MLRunList", openapigen.ListEnvelope("MLRun"))
	g.Set("MLServiceList", openapigen.ListEnvelope("MLService"))
	g.Set("TrafficPolicyList", openapigen.ListEnvelope("TrafficPolicy"))
	g.Set("PodList", openapigen.ListEnvelope("Pod"))
	g.Set("EventList", openapigen.ListEnvelope("Event"))

	tags := []openapigen.TagEntry{
		{Name: tagMLRuns, Description: "MLRun CRUD per namespace. ResourcePool/Unit referenced by name (read from K8s Informer cache)."},
		{Name: tagMLServices, Description: "MLService CRUD per namespace."},
		{Name: tagTrafficPolicies, Description: "MLTrafficPolicy CRUD per namespace: weighted / canary / blue-green traffic split over member online services."},
		{Name: tagCapabilities, Description: "Deployment-form capability document (runtime engine / quota enforcement)."},
		{Name: tagHealth, Description: "Liveness and readiness probes."},
	}

	nsParam := openapigen.PathParam("namespace", "Tenant name (= mlruns/mlservices partition key).")
	mlrunParam := openapigen.PathParam("mlrun", "MLRun name.")
	mlserviceParam := openapigen.PathParam("mlservice", "MLService name.")
	policyParam := openapigen.PathParam("policy", "Traffic policy name.")

	limitParam := openapigen.QueryParam("limit", "Page size (1–200, default 50).", openapigen.IntFormat32Param())
	continueParam := openapigen.QueryParam("continue", "Opaque continuation token from a previous page.", &openapigen.Schema{Type: "string"})
	labelSelectorParam := openapigen.QueryParam("labelSelector", "K8s-style label selector filtered against the row's labels jsonb.", &openapigen.Schema{Type: "string"})
	logParams := []openapigen.Parameter{
		openapigen.QueryParam("container", "Target container (defaults to the first).", &openapigen.Schema{Type: "string"}),
		openapigen.QueryParam("tailLines", "Return only the last N lines.", openapigen.IntFormat32Param()),
		openapigen.QueryParam("follow", "Stream new lines as Server-Sent Events (text/event-stream) instead of a one-shot text/plain body.", &openapigen.Schema{Type: "boolean"}),
		openapigen.QueryParam("previous", "Read the previous (crashed) container instance's log.", &openapigen.Schema{Type: "boolean"}),
	}

	paths := map[string]openapigen.PathItem{}

	// system probes
	paths["/healthz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagHealth}, Summary: "Liveness probe", OperationID: "healthz",
		Responses: map[string]openapigen.Response{"200": openapigen.StringResp("ok")},
	}}
	paths["/readyz"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagHealth}, Summary: "Readiness probe", OperationID: "readyz",
		Responses: map[string]openapigen.Response{
			"200": openapigen.StringResp("ok"),
			"503": openapigen.StringResp("dependency not yet ready"),
		},
	}}

	paths["/api/v1/capabilities"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagCapabilities}, Summary: "Get deployment-form capabilities", OperationID: "getCapabilities",
		Responses: map[string]openapigen.Response{"200": openapigen.JSONResp("Capability document.", "Capabilities")},
	}}

	// mlruns (per namespace)
	paths["/api/v1/namespaces/{namespace}/mlruns"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "Submit an MLRun", OperationID: "createMLRun",
			Parameters:  []openapigen.Parameter{nsParam},
			RequestBody: openapigen.JSONBody("MLRunCreateRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "MLRun")}),
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
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("MLRun.", "MLRun")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagMLRuns}, Summary: "Patch MLRun display fields", OperationID: "patchMLRun",
			Parameters:  []openapigen.Parameter{nsParam, mlrunParam},
			RequestBody: openapigen.JSONBody("MLRunPatchRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Patched MLRun.", "MLRun")}),
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
		Responses:  withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Cancellation queued (row is Canceling).", "MLRun")}),
	}}

	podParam := openapigen.PathParam("pod", "Pod name.")
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "List pods of an MLRun", OperationID: "listMLRunPods",
		Parameters: []openapigen.Parameter{nsParam, mlrunParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Pods.", "PodList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods/{pod}/logs"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLRuns}, Summary: "Stream a pod's container log", OperationID: "getMLRunPodLogs",
		Parameters: append([]openapigen.Parameter{nsParam, mlrunParam, podParam}, logParams...),
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
			RequestBody: openapigen.JSONBody("MLServiceCreateRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "MLService")}),
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
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("MLService.", "MLService")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagMLServices}, Summary: "Patch MLService display fields", OperationID: "patchMLService",
			Parameters:  []openapigen.Parameter{nsParam, mlserviceParam},
			RequestBody: openapigen.JSONBody("MLServicePatchRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Patched service.", "MLService")}),
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
		RequestBody: openapigen.JSONBody("MLServiceScaleRequest"),
		Responses:   withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Scale queued (generation bumped).", "MLService")}),
	}}

	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "List pods of an MLService", OperationID: "listMLServicePods",
		Parameters: []openapigen.Parameter{nsParam, mlserviceParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Pods.", "PodList")}),
	}}
	paths["/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods/{pod}/logs"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagMLServices}, Summary: "Stream an MLService pod's container log", OperationID: "getMLServicePodLogs",
		Parameters: append([]openapigen.Parameter{nsParam, mlserviceParam, podParam}, logParams...),
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
			RequestBody: openapigen.JSONBody("TrafficPolicyCreateRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "TrafficPolicy")}),
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
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Traffic policy.", "TrafficPolicy")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagTrafficPolicies}, Summary: "Patch traffic policy display fields", OperationID: "patchTrafficPolicy",
			Parameters:  []openapigen.Parameter{nsParam, policyParam},
			RequestBody: openapigen.JSONBody("TrafficPolicyPatchRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Patched traffic policy.", "TrafficPolicy")}),
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
		RequestBody: openapigen.JSONBody("TrafficPolicySplitRequest"),
		Responses:   withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Split queued (generation bumped).", "TrafficPolicy")}),
	}}
	paths["/api/v1/namespaces/{namespace}/traffic-policies/{policy}/promote"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTrafficPolicies}, Summary: "Promote the canary to stable (canary mode)", OperationID: "promoteTrafficPolicy",
		Parameters: []openapigen.Parameter{nsParam, policyParam},
		Responses:  withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Promote queued (generation bumped).", "TrafficPolicy")}),
	}}
	paths["/api/v1/namespaces/{namespace}/traffic-policies/{policy}/rollback"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagTrafficPolicies}, Summary: "Roll the canary back to 0 (canary mode)", OperationID: "rollbackTrafficPolicy",
		Parameters: []openapigen.Parameter{nsParam, policyParam},
		Responses:  withErrors(map[string]openapigen.Response{"202": openapigen.JSONResp("Rollback queued (generation bumped).", "TrafficPolicy")}),
	}}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Compute Service API",
			Version:     version,
			Description: "REST API for per-namespace Jobs, Services and traffic policies. Tenant/Quota and ResourcePool/Unit live in the cluster-manager CRDs; compute partitions on the bare namespace string (= tenant identifier) supplied by Platform. RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}

// packageNamer routes a Go type's package path to its schema-name prefix.
// The flat internal/server package (which now owns every request / response
// DTO) maps to an empty prefix so the schema name equals the Go type name
// verbatim — no "Server" stutter. The compute-operator's nested API packages
// fall through to operatorAPIPrefix.
func packageNamer(pkg string) (string, bool) {
	if strings.HasSuffix(pkg, "/components/compute-service/internal/server") {
		return "", true
	}
	return operatorAPIPrefix(pkg)
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
