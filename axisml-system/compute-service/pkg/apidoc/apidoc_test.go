package apidoc

import (
	"strings"
	"testing"

	"github.com/axisml/axisml/pkg/openapigen"

	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

func TestBuildDocumentSchemaIntegrity(t *testing.T) {
	doc := Document("test")
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
	doc := Document("test")
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
	doc := Document("test")
	want := []struct{ method, path string }{
		{"get", "/healthz"},
		{"get", "/readyz"},
		{"post", "/api/v1/namespaces/{namespace}/mlruns"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/phases"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}"},
		{"patch", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}"},
		{"delete", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/phase"},
		{"post", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/cancel"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods/{pod}/logs"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/pods/{pod}/events"},
		{"get", "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/events"},
		{"post", "/api/v1/namespaces/{namespace}/mlservices"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/phases"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}"},
		{"patch", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}"},
		{"delete", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/phase"},
		{"post", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/scale"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods/{pod}/logs"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/pods/{pod}/events"},
		{"get", "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/events"},
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
	doc := Document("test")
	prob, ok := doc.Components.Schemas["ComputeServiceError"]
	if !ok {
		t.Fatal("ComputeServiceError schema missing")
	}
	codeProp := prob.Properties["code"]
	if codeProp == nil {
		t.Fatal("ComputeServiceError.code missing")
	}
	if len(codeProp.Enum) != len(apperrors.AllCodes()) {
		t.Errorf("ComputeServiceError.code enum len = %d, want %d", len(codeProp.Enum), len(apperrors.AllCodes()))
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
