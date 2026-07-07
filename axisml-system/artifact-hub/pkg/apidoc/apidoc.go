// Package apidoc builds the OpenAPI 3.0.3 description of the artifacts HTTP API
// and is the single source of truth for that contract.
//
// Schemas are derived by reflection from the same Go request/response structs
// the runtime handlers use, so the spec stays in lock-step with the code.
// Routes are listed explicitly here rather than scraped from the gin router so
// the document is reviewable.
//
// The component's own cmd/openapi-gen renders Document to
// axisml-system/docs/apis/artifact-hub.yaml; axisml-core imports Document
// directly to fold the artifacts surface into the Lite composite spec, without a
// YAML round-trip.
package apidoc

import (
	"reflect"
	"strings"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/server"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
	"github.com/axisml/axisml/pkg/openapigen"
)

const (
	tagArtifacts    = "Artifacts"
	tagCapabilities = "Capabilities"
	tagHealth       = "Health"
)

// Artifact name / version policies. Duplicated here as regex rather than
// imported so generated client SDKs don't drag in our strutil package.
const (
	axisMLNamePattern    = "^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$"
	axisMLVersionPattern = "^[a-zA-Z0-9._-]+$"
)

// withErrors returns the standard Problem-bearing error responses. Codes here
// must stay in lock-step with internal/server/problem.go's statusFor —
// artifacts has no 422 (no quotas) but emits 410 Gone.
func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "ArtifactHubError"),
		"401":     openapigen.JSONResp("Unauthorized.", "ArtifactHubError"),
		"403":     openapigen.JSONResp("Forbidden.", "ArtifactHubError"),
		"404":     openapigen.JSONResp("Not found.", "ArtifactHubError"),
		"409":     openapigen.JSONResp("Conflict.", "ArtifactHubError"),
		"410":     openapigen.JSONResp("Gone.", "ArtifactHubError"),
		"412":     openapigen.JSONResp("Precondition failed.", "ArtifactHubError"),
		"503":     openapigen.JSONResp("Service unavailable.", "ArtifactHubError"),
		"default": openapigen.JSONResp("Unexpected error.", "ArtifactHubError"),
	}
	for k, v := range success {
		out[k] = v
	}
	return out
}

// Document builds the complete OpenAPI document for the artifacts API.
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
			{Tag: "axisml_version", Pattern: axisMLVersionPattern, MinLength: 1, MaxLength: 64},
		},
		// The API DTO types carry their own descriptive names (Artifact,
		// UploadCredentials, …); map the package to an empty prefix so nested
		// $refs resolve to the bare type name instead of stuttering (ArtifactArtifact).
		PackageNamer: func(pkg string) (string, bool) {
			if strings.HasSuffix(pkg, "/axisml-system/artifact-hub/internal/server") {
				return "", true
			}
			return "", false
		},
	})

	g.Register("ArtifactHubError", server.Error{}, openapigen.ResponseMode)
	g.Register("ArtifactInitiateRequest", server.ArtifactInitiateRequest{}, openapigen.InputMode)
	g.Register("ArtifactInitiateResponse", server.ArtifactInitiateResponse{}, openapigen.ResponseMode)
	g.Register("ArtifactCompleteRequest", server.ArtifactCompleteRequest{}, openapigen.InputMode)
	g.Register("ArtifactPatchRequest", server.ArtifactPatchRequest{}, openapigen.InputMode)
	g.Register("ArtifactResolveResponse", server.ArtifactResolveResponse{}, openapigen.ResponseMode)
	g.Register("Artifact", server.Artifact{}, openapigen.ResponseMode)
	g.Register("Capabilities", server.Capabilities{}, openapigen.ResponseMode)

	g.Set("ArtifactList", openapigen.ListEnvelope("Artifact"))

	// After the list envelope is registered, so its example can attach too.
	registerExamples(g)

	tags := []openapigen.TagEntry{
		{Name: tagArtifacts, Description: "Artifact registry addressed by (namespace, name, version); kind is an attribute."},
		{Name: tagCapabilities, Description: "Deployment-form capability document (artifact kinds / upload)."},
		{Name: tagHealth, Description: "Liveness and readiness probes."},
	}

	nsParam := openapigen.PathParam("namespace", "Tenant namespace (= compute tenants.name).")
	nameParam := openapigen.PathParam("name", "Artifact name.")
	versionParam := openapigen.PathParam("version", "Artifact version (free-form string).")

	limitParam := openapigen.QueryParam("limit", "Page size (1–200, default 50).", openapigen.IntFormat32Param())
	continueParam := openapigen.QueryParam("continue", "Opaque continuation token from a previous page.", &openapigen.Schema{Type: "string"})
	labelSelectorParam := openapigen.QueryParam("labelSelector", "K8s-style label selector filtered against the row's labels jsonb.", &openapigen.Schema{Type: "string"})
	statusParam := openapigen.QueryParam("status", "Optional status filter (pending, ready, failed).", &openapigen.Schema{Type: "string"})
	kindParam := openapigen.QueryParam("kind", "Optional kind filter (model, dataset, image).", &openapigen.Schema{Type: "string"})
	usageParam := openapigen.QueryParam("usage", "Optional usage hint forwarded to the storage handler.", &openapigen.Schema{Type: "string"})

	paths := map[string]openapigen.PathItem{}

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

	// A single /artifacts sub-tree serves every kind. Artifacts are addressed by
	// (namespace, name, version); kind travels in the Initiate body and is
	// echoed on every Artifact response.
	base := "/api/v1/namespaces/{namespace}/artifacts"
	paths[base] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagArtifacts}, Summary: "List every artifact (across all names) in a namespace",
		OperationID: "listArtifacts",
		Parameters:  []openapigen.Parameter{nsParam, kindParam, limitParam, continueParam, statusParam, labelSelectorParam},
		Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "ArtifactList")}),
	}}
	paths[base+"/{name}"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Initiate an artifact version (two-phase write step 1)",
			OperationID: "initiateArtifact",
			Parameters:  []openapigen.Parameter{nsParam, nameParam},
			RequestBody: openapigen.JSONBody("ArtifactInitiateRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Initiated.", "ArtifactInitiateResponse")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "List versions of an artifact",
			OperationID: "listArtifactVersions",
			Parameters:  []openapigen.Parameter{nsParam, nameParam, limitParam, continueParam, statusParam, labelSelectorParam},
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "ArtifactList")}),
		},
	}
	paths[base+"/{name}/{version}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Get an artifact version",
			OperationID: "getArtifact",
			Parameters:  []openapigen.Parameter{nsParam, nameParam, versionParam},
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Artifact.", "Artifact")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Patch an artifact's display_name / description / labels / annotations",
			OperationID: "updateArtifact",
			Parameters:  []openapigen.Parameter{nsParam, nameParam, versionParam},
			RequestBody: openapigen.JSONBody("ArtifactPatchRequest"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated.", "Artifact")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Delete an artifact version",
			OperationID: "deleteArtifact",
			Parameters:  []openapigen.Parameter{nsParam, nameParam, versionParam},
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Soft-deleted artifact (status=Deleting).", "Artifact")}),
		},
	}
	paths[base+"/{name}/{version}/complete"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagArtifacts}, Summary: "Complete artifact upload (two-phase write step 2)",
		OperationID: "completeArtifact",
		Parameters:  []openapigen.Parameter{nsParam, nameParam, versionParam},
		RequestBody: openapigen.JSONBody("ArtifactCompleteRequest"),
		Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Completed artifact.", "Artifact")}),
	}}
	paths[base+"/{name}/{version}/resolve"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagArtifacts}, Summary: "Resolve an artifact for download",
		OperationID: "resolveArtifact",
		Parameters:  []openapigen.Parameter{nsParam, nameParam, versionParam, usageParam},
		Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Resolved.", "ArtifactResolveResponse")}),
	}}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Artifact Hub API",
			Version:     version,
			Description: "REST API for the artifact registry. Items are addressed by (namespace, name, version); kind is an attribute supplied on create. RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}
