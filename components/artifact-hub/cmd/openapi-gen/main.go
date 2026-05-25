// openapi-gen renders an OpenAPI 3.0.3 description of the artifacts HTTP API
// to docs/openapi/artifact-hub.yaml at the repo root.
//
// Schemas are derived from the same Go request/response structs the runtime
// handlers use, so the spec stays in lock-step with the code. Routes are
// listed explicitly here (single source of truth) rather than scraped from
// the gin router so the file is reviewable.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../../docs/openapi/artifact-hub.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/axisml/axisml/components/artifact-hub/internal/artifact"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	apperrors "github.com/axisml/axisml/components/artifact-hub/pkg/errors"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

const (
	tagArtifacts = "artifacts"
	tagSystem    = "system"
)

// Artifact name / version policies. Duplicated here as regex rather than
// imported so generated client SDKs don't drag in our strutil package.
const (
	axisMLNamePattern    = "^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$"
	axisMLVersionPattern = "^[a-zA-Z0-9._-]+$"
)

func main() {
	out := flag.String("o", "../../docs/openapi/artifact-hub.yaml", "output path")
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

// withErrors returns the standard Problem-bearing error responses. Codes here
// must stay in lock-step with internal/server/problem.go's statusFor —
// artifacts has no 422 (no quotas) but emits 410 Gone.
func withErrors(success map[string]openapigen.Response) map[string]openapigen.Response {
	out := map[string]openapigen.Response{
		"400":     openapigen.JSONResp("Validation error.", "Problem"),
		"401":     openapigen.JSONResp("Unauthorized.", "Problem"),
		"403":     openapigen.JSONResp("Forbidden.", "Problem"),
		"404":     openapigen.JSONResp("Not found.", "Problem"),
		"409":     openapigen.JSONResp("Conflict.", "Problem"),
		"410":     openapigen.JSONResp("Gone.", "Problem"),
		"412":     openapigen.JSONResp("Precondition failed.", "Problem"),
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
			{Tag: "axisml_version", Pattern: axisMLVersionPattern, MinLength: 1, MaxLength: 64},
		},
	})

	g.Register("Problem", server.Problem{}, openapigen.ResponseMode)
	g.Register("ArtifactInitiateInput", artifact.InitiateInput{}, openapigen.InputMode)
	g.Register("ArtifactInitiateResult", artifact.InitiateResult{}, openapigen.ResponseMode)
	g.Register("ArtifactCompleteInput", artifact.CompleteInput{}, openapigen.InputMode)
	g.Register("ArtifactResolveResult", artifact.ResolveResult{}, openapigen.ResponseMode)
	g.Register("ArtifactView", artifact.View{}, openapigen.ResponseMode)

	g.Set("ArtifactList", openapigen.ListEnvelope("ArtifactView"))

	tags := []openapigen.TagEntry{
		{Name: tagArtifacts, Description: "Artifact registry partitioned by (namespace, kind, name, version)."},
		{Name: tagSystem, Description: "Liveness and readiness probes."},
	}

	nsParam := openapigen.PathParam("namespace", "Kubernetes namespace.")
	kindParam := openapigen.PathParam("kind", "Artifact kind (e.g. model, dataset, code).")
	nameParam := openapigen.PathParam("name", "Artifact name.")
	versionParam := openapigen.PathParam("version", "Artifact version (free-form string).")

	limitParam := openapigen.QueryParam("limit", "Page size (1–200, default 50).", openapigen.IntFormat32Param())
	offsetParam := openapigen.QueryParam("offset", "Number of items to skip.", openapigen.IntFormat32Param())
	statusParam := openapigen.QueryParam("status", "Optional status filter (pending, ready, failed).", &openapigen.Schema{Type: "string"})
	usageParam := openapigen.QueryParam("usage", "Optional usage hint forwarded to the storage handler.", &openapigen.Schema{Type: "string"})

	paths := map[string]openapigen.PathItem{}

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

	paths["/api/v1/namespaces/{namespace}/artifacts/{kind}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "List every (name, version) of a kind in a namespace", OperationID: "listArtifactsByKind",
			Parameters: []openapigen.Parameter{nsParam, kindParam, limitParam, offsetParam, statusParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "ArtifactList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/artifacts/{kind}/{name}"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Initiate a new artifact version (two-phase write step 1)", OperationID: "initiateArtifact",
			Parameters:  []openapigen.Parameter{nsParam, kindParam, nameParam},
			RequestBody: openapigen.JSONBody("ArtifactInitiateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Initiated.", "ArtifactInitiateResult")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "List versions of an artifact", OperationID: "listArtifactVersions",
			Parameters: []openapigen.Parameter{nsParam, kindParam, nameParam, limitParam, offsetParam, statusParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "ArtifactList")}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/artifacts/{kind}/{name}/{version}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Get an artifact version", OperationID: "getArtifact",
			Parameters: []openapigen.Parameter{nsParam, kindParam, nameParam, versionParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Artifact.", "ArtifactView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagArtifacts}, Summary: "Delete an artifact version", OperationID: "deleteArtifact",
			Parameters: []openapigen.Parameter{nsParam, kindParam, nameParam, versionParam},
			Responses:  withErrors(map[string]openapigen.Response{"202": {Description: "Accepted (asynchronous deletion)."}}),
		},
	}
	paths["/api/v1/namespaces/{namespace}/artifacts/{kind}/{name}/{version}/complete"] = openapigen.PathItem{Post: &openapigen.Operation{
		Tags: []string{tagArtifacts}, Summary: "Complete an artifact upload (two-phase write step 2)", OperationID: "completeArtifact",
		Parameters:  []openapigen.Parameter{nsParam, kindParam, nameParam, versionParam},
		RequestBody: openapigen.JSONBody("ArtifactCompleteInput"),
		Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Completed artifact.", "ArtifactView")}),
	}}
	paths["/api/v1/namespaces/{namespace}/artifacts/{kind}/{name}/{version}/resolve"] = openapigen.PathItem{Get: &openapigen.Operation{
		Tags: []string{tagArtifacts}, Summary: "Resolve an artifact for download", OperationID: "resolveArtifact",
		Parameters: []openapigen.Parameter{nsParam, kindParam, nameParam, versionParam, usageParam},
		Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Resolved.", "ArtifactResolveResult")}),
	}}

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Artifact Hub API",
			Version:     version,
			Description: "REST API for the artifact registry. Items are keyed by (namespace, kind, name, version). RFC7807 Problem responses on errors.",
		},
		Servers: []openapigen.ServerEntry{{URL: "/", Description: "Same-origin"}},
		Tags:    tags,
		Paths:   paths,
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}
