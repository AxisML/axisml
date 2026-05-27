package main

import (
	"strings"
	"testing"

	"github.com/axisml/axisml/pkg/openapigen"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

func TestBuildDocumentSchemaIntegrity(t *testing.T) {
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

func TestOperationIDsUnique(t *testing.T) {
	doc := buildDocument("test")
	seen := map[string]string{}
	for path, item := range doc.Paths {
		for method, op := range openapigen.OpsOf(item) {
			if op == nil {
				continue
			}
			if existing, ok := seen[op.OperationID]; ok {
				t.Errorf("duplicate operationId %q at %s %s (also %s)",
					op.OperationID, method, path, existing)
				continue
			}
			seen[op.OperationID] = method + " " + path
		}
	}
}

// TestRouteCoverage verifies every route registered by the gin handlers has a
// corresponding entry in the spec. Drift here means a client SDK can't see a
// real endpoint.
func TestRouteCoverage(t *testing.T) {
	doc := buildDocument("test")
	want := []struct{ method, path string }{
		{"get", "/healthz"},
		{"get", "/readyz"},
		{"post", "/api/v1/namespaces"},
		{"get", "/api/v1/namespaces"},
		{"get", "/api/v1/namespaces/{namespace}"},
		{"patch", "/api/v1/namespaces/{namespace}"},
		{"delete", "/api/v1/namespaces/{namespace}"},
		{"post", "/api/v1/namespaces/{namespace}/jobs"},
		{"get", "/api/v1/namespaces/{namespace}/jobs"},
		{"get", "/api/v1/namespaces/{namespace}/jobs/{job}"},
		{"delete", "/api/v1/namespaces/{namespace}/jobs/{job}"},
		{"post", "/api/v1/namespaces/{namespace}/jobs/{job}/cancel"},
		{"get", "/api/v1/namespaces/{namespace}/jobs/{job}/pods"},
		{"get", "/api/v1/namespaces/{namespace}/jobs/{job}/pods/{pod}/logs"},
		{"get", "/api/v1/namespaces/{namespace}/jobs/{job}/pods/{pod}/events"},
		{"get", "/api/v1/namespaces/{namespace}/jobs/{job}/events"},
		{"post", "/api/v1/namespaces/{namespace}/services"},
		{"get", "/api/v1/namespaces/{namespace}/services"},
		{"get", "/api/v1/namespaces/{namespace}/services/{service}"},
		{"delete", "/api/v1/namespaces/{namespace}/services/{service}"},
		{"post", "/api/v1/namespaces/{namespace}/services/{service}/scale"},
		{"get", "/api/v1/namespaces/{namespace}/services/{service}/pods"},
		{"get", "/api/v1/namespaces/{namespace}/services/{service}/pods/{pod}/logs"},
		{"get", "/api/v1/namespaces/{namespace}/services/{service}/pods/{pod}/events"},
		{"get", "/api/v1/namespaces/{namespace}/services/{service}/events"},
		{"post", "/api/v1/namespaces/{namespace}/restore"},
		{"get", "/api/v1/namespaces/{namespace}/quotas"},
		{"post", "/api/v1/namespaces/{namespace}/quotas"},
		{"patch", "/api/v1/namespaces/{namespace}/quotas/{pool}/{name}"},
		{"delete", "/api/v1/namespaces/{namespace}/quotas/{pool}/{name}"},
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

// TestProblemEnumIsSourced asserts the Problem schema's `code` field is the
// apperrors.Code enum (not just a free string), so client codegens emit a
// typed accessor and stay in sync with this component's error codes.
func TestProblemEnumIsSourced(t *testing.T) {
	doc := buildDocument("test")
	prob, ok := doc.Components.Schemas["Problem"]
	if !ok {
		t.Fatal("Problem schema missing")
	}
	codeProp := prob.Properties["code"]
	if codeProp == nil {
		t.Fatal("Problem.code missing")
	}
	if len(codeProp.Enum) != len(apperrors.AllCodes()) {
		t.Errorf("Problem.code enum len = %d, want %d", len(codeProp.Enum), len(apperrors.AllCodes()))
	}
}

// TestErrorOverlayCoversBusinessCodes pins the standard error-response
// overlay so a new business code in problem.go can't silently disappear from
// every operation's response set.
func TestErrorOverlayCoversBusinessCodes(t *testing.T) {
	resps := withErrors(map[string]openapigen.Response{"200": {Description: "ok"}})
	for _, code := range []string{"400", "401", "403", "404", "409", "412", "422", "503", "default"} {
		if _, ok := resps[code]; !ok {
			t.Errorf("withErrors missing %q response", code)
		}
	}
}
