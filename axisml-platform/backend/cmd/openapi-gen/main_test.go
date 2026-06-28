package main

import (
	"strings"
	"testing"

	"github.com/axisml/axisml/pkg/openapigen"
)

// TestSchemaIntegrity asserts every $ref in the document — in component
// schemas, request bodies, responses, and operation parameters — resolves to a
// declared component schema. Parameters are walked too because platform filters
// (phase / status / metric) reference named enum components by $ref.
func TestSchemaIntegrity(t *testing.T) {
	doc := buildDocument("test")
	schemas := doc.Components.Schemas
	const prefix = "#/components/schemas/"

	checkRef := func(where, target string) {
		if !strings.HasPrefix(target, prefix) {
			t.Errorf("non-component $ref %q at %s", target, where)
			return
		}
		if _, ok := schemas[strings.TrimPrefix(target, prefix)]; !ok {
			t.Errorf("dangling $ref %q (used at %s)", target, where)
		}
	}

	for name, s := range schemas {
		openapigen.WalkSchema(s, "components."+name, checkRef)
	}
	for path, item := range doc.Paths {
		for method, op := range openapigen.OpsOf(item) {
			if op == nil {
				continue
			}
			where := method + " " + path
			for _, prm := range op.Parameters {
				openapigen.WalkSchema(prm.Schema, where+".params."+prm.Name, checkRef)
			}
			if op.RequestBody != nil {
				for _, mt := range op.RequestBody.Content {
					openapigen.WalkSchema(mt.Schema, where+".requestBody", checkRef)
				}
			}
			for code, r := range op.Responses {
				for _, mt := range r.Content {
					openapigen.WalkSchema(mt.Schema, where+".responses."+code, checkRef)
				}
			}
		}
	}
}

// TestOperationIDsUnique guards against a copy-paste leaving two operations
// sharing an operationId (which breaks client SDK codegen).
func TestOperationIDsUnique(t *testing.T) {
	doc := buildDocument("test")
	seen := map[string]string{}
	for path, item := range doc.Paths {
		for method, op := range openapigen.OpsOf(item) {
			if op == nil {
				continue
			}
			if existing, ok := seen[op.OperationID]; ok {
				t.Errorf("duplicate operationId %q at %s %s (also %s)", op.OperationID, method, path, existing)
				continue
			}
			seen[op.OperationID] = method + " " + path
		}
	}
}

// TestEveryOperationIsTagged keeps the generated spec navigable: a tagless
// operation falls out of the grouped docs view.
func TestEveryOperationIsTagged(t *testing.T) {
	doc := buildDocument("test")
	known := map[string]bool{}
	for _, tg := range tags() {
		known[tg.Name] = true
	}
	for path, item := range doc.Paths {
		for method, op := range openapigen.OpsOf(item) {
			if op == nil {
				continue
			}
			if len(op.Tags) != 1 || !known[op.Tags[0]] {
				t.Errorf("%s %s has tags %v; want exactly one known tag", method, path, op.Tags)
			}
		}
	}
}

// TestExpectedCounts pins the surface size so an accidental drop of a route or
// schema is caught.
func TestExpectedCounts(t *testing.T) {
	doc := buildDocument("test")
	if got := len(doc.Paths); got != 82 {
		t.Errorf("path count = %d, want 82", got)
	}
	if got := len(doc.Components.Schemas); got != 143 {
		t.Errorf("schema count = %d, want 143", got)
	}
}

// TestKeySchemasPresent spot-checks the headline schemas and that named enums
// carry their value sets.
func TestKeySchemasPresent(t *testing.T) {
	doc := buildDocument("test")
	for _, name := range []string{
		"Problem", "Tenant", "Workspace", "Job", "Run", "MLService",
		"Model", "Image", "Experiment", "TrafficPolicy", "ResourcePool",
		"StringMap", "ResourceMap", "ModelSpec",
	} {
		if _, ok := doc.Components.Schemas[name]; !ok {
			t.Errorf("missing component schema %q", name)
		}
	}
	rn := doc.Components.Schemas["RoleName"]
	if rn == nil || len(rn.Enum) != 3 {
		t.Errorf("RoleName enum missing or wrong size: %+v", rn)
	}
}

// TestRouteCoverage verifies a representative set of routes/methods exists.
func TestRouteCoverage(t *testing.T) {
	doc := buildDocument("test")
	want := []struct{ method, path string }{
		{"get", "/healthz"},
		{"post", "/api/v1/auth/login"},
		{"get", "/api/v1/auth/me"},
		{"post", "/api/v1/tenants"},
		{"patch", "/api/v1/tenants/{name}"},
		{"post", "/api/v1/tenants/{name}/quotas"},
		{"delete", "/api/v1/tenants/{name}/members/{userId}"},
		{"post", "/api/v1/workspaces"},
		{"get", "/api/v1/workspaces/{name}/pods/{pod}/logs"},
		{"post", "/api/v1/jobs/{name}/runs"},
		{"post", "/api/v1/jobs/{name}/runs/{run}/cancel"},
		{"post", "/api/v1/mlservices/{name}/scale"},
		{"get", "/api/v1/mlservices/{name}/metrics"},
		{"post", "/api/v1/tenants/{name}/suspend"},
		{"post", "/api/v1/experiments"},
		{"post", "/api/v1/experiments/{name}/runs"},
		{"post", "/api/v1/experiments/{name}/tensorboard"},
		{"post", "/api/v1/trafficpolicies"},
		{"post", "/api/v1/trafficpolicies/{name}/split"},
		{"post", "/api/v1/trafficpolicies/{name}/promote"},
		{"post", "/api/v1/models/{tenant}/{name}/versions"},
		{"post", "/api/v1/models/{tenant}/{name}/versions/{version}/complete"},
		{"get", "/api/v1/images/{tenant}/{name}/versions/{version}/resolve"},
		{"post", "/api/v1/resourcepools/{pool}/units"},
	}
	for _, w := range want {
		item, ok := doc.Paths[w.path]
		if !ok {
			t.Errorf("missing path %q", w.path)
			continue
		}
		if openapigen.OpsOf(item)[w.method] == nil {
			t.Errorf("missing %s on path %q", w.method, w.path)
		}
	}
}
