// openapi-gen renders an OpenAPI 3.0.3 description of the compute HTTP API
// to docs/openapi/compute.yaml at the repo root.
//
// Schemas are derived from the same Go request/response structs the runtime
// handlers use, so the spec stays in lock-step with the code. Routes are
// listed explicitly here (single source of truth) rather than scraped from
// the gin router so the file is reviewable.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../../docs/openapi/compute.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/axisml/axisml/components/compute/internal/job"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/resourceunit"
	"github.com/axisml/axisml/components/compute/internal/server"
	servicemod "github.com/axisml/axisml/components/compute/internal/service"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

// Tag names. One source of truth so a typo can't silently split a group.
const (
	tagResourcePools = "resource-pools"
	tagResourceUnits = "resource-units"
	tagJobs          = "jobs"
	tagServices      = "services"
	tagSystem        = "system"
)

// AxisML §6.1 name policy is duplicated here as a regex rather than imported
// because the generator's contract is "render whatever clients need to send";
// clients don't import strutil. Same constants live in
// components/compute/pkg/strutil — keep them in sync if the policy changes.
const (
	axisMLNamePattern         = "^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$"
	axisMLResourceUnitPattern = axisMLNamePattern
)

func main() {
	out := flag.String("o", "../../docs/openapi/compute.yaml", "output path")
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
			{Tag: "axisml_resource_unit", Pattern: axisMLResourceUnitPattern, MinLength: 3, MaxLength: 40},
		},
		PackageNamer: operatorAPIPrefix,
	})

	// Core component schemas (referenced from operations).
	g.Register("Problem", server.Problem{}, openapigen.ResponseMode)
	g.Register("ResourcepoolCreateInput", resourcepool.CreateInput{}, openapigen.InputMode)
	g.Register("ResourcepoolUpdateInput", resourcepool.UpdateInput{}, openapigen.InputMode)
	g.Register("ResourcepoolView", resourcepool.View{}, openapigen.ResponseMode)
	g.Register("ResourceunitCreateInput", resourceunit.CreateInput{}, openapigen.InputMode)
	g.Register("ResourceunitUpdateInput", resourceunit.UpdateInput{}, openapigen.InputMode)
	g.Register("ResourceunitView", resourceunit.View{}, openapigen.ResponseMode)
	g.Register("JobCreateInput", job.CreateInput{}, openapigen.InputMode)
	g.Register("JobView", job.View{}, openapigen.ResponseMode)
	g.Register("MLServiceCreateInput", servicemod.CreateInput{}, openapigen.InputMode)
	g.Register("MLServiceScaleInput", servicemod.ScaleInput{}, openapigen.InputMode)
	g.Register("MLServiceView", servicemod.View{}, openapigen.ResponseMode)

	g.Set("ResourcepoolList", openapigen.ListEnvelope("ResourcepoolView"))
	g.Set("ResourceunitList", openapigen.ListEnvelope("ResourceunitView"))
	g.Set("JobList", openapigen.ListEnvelope("JobView"))
	g.Set("MLServiceList", openapigen.ListEnvelope("MLServiceView"))

	tags := []openapigen.TagEntry{
		{Name: tagResourcePools, Description: "Cluster-scoped resource pool registry."},
		{Name: tagResourceUnits, Description: "Reusable CPU/GPU/memory recipes scoped to a pool."},
		{Name: tagJobs, Description: "MLJob CRUD per namespace."},
		{Name: tagServices, Description: "MLService CRUD per namespace."},
		{Name: tagSystem, Description: "Liveness and readiness probes."},
	}

	nsParam := openapigen.PathParam("namespace", "Kubernetes namespace.")
	poolParam := openapigen.PathParam("pool", "Resource pool name.")
	unitParam := openapigen.PathParam("unit", "Resource unit name.")
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

	// resource pools
	paths["/api/v1/resource-pools"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Create a resource pool", OperationID: "createResourcePool",
			RequestBody: openapigen.JSONBody("ResourcepoolCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "ResourcepoolView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "List resource pools", OperationID: "listResourcePools",
			Parameters: []openapigen.Parameter{limitParam, offsetParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "ResourcepoolList")}),
		},
	}
	paths["/api/v1/resource-pools/{pool}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Get resource pool", OperationID: "getResourcePool",
			Parameters: []openapigen.Parameter{poolParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Resource pool.", "ResourcepoolView")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Patch resource pool", OperationID: "updateResourcePool",
			Parameters:  []openapigen.Parameter{poolParam},
			RequestBody: openapigen.JSONBody("ResourcepoolUpdateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated.", "ResourcepoolView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagResourcePools}, Summary: "Delete resource pool", OperationID: "deleteResourcePool",
			Parameters: []openapigen.Parameter{poolParam},
			Responses:  withErrors(map[string]openapigen.Response{"204": openapigen.NoContentResp}),
		},
	}

	// resource units (under a pool)
	paths["/api/v1/resource-pools/{pool}/resource-units"] = openapigen.PathItem{
		Post: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Create a resource unit", OperationID: "createResourceUnit",
			Parameters:  []openapigen.Parameter{poolParam},
			RequestBody: openapigen.JSONBody("ResourceunitCreateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"201": openapigen.JSONResp("Created.", "ResourceunitView")}),
		},
		Get: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "List resource units in a pool", OperationID: "listResourceUnits",
			Parameters: []openapigen.Parameter{poolParam, limitParam, offsetParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Page.", "ResourceunitList")}),
		},
	}
	paths["/api/v1/resource-pools/{pool}/resource-units/{unit}"] = openapigen.PathItem{
		Get: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Get resource unit", OperationID: "getResourceUnit",
			Parameters: []openapigen.Parameter{poolParam, unitParam},
			Responses:  withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Resource unit.", "ResourceunitView")}),
		},
		Patch: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Patch resource unit", OperationID: "updateResourceUnit",
			Parameters:  []openapigen.Parameter{poolParam, unitParam},
			RequestBody: openapigen.JSONBody("ResourceunitUpdateInput"),
			Responses:   withErrors(map[string]openapigen.Response{"200": openapigen.JSONResp("Updated.", "ResourceunitView")}),
		},
		Delete: &openapigen.Operation{
			Tags: []string{tagResourceUnits}, Summary: "Delete resource unit", OperationID: "deleteResourceUnit",
			Parameters: []openapigen.Parameter{poolParam, unitParam},
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

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:       "AxisML Compute API",
			Version:     version,
			Description: "REST API for resource pools, resource units, jobs, and services. Partitioned by Kubernetes namespace. RFC7807 Problem responses on errors.",
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
