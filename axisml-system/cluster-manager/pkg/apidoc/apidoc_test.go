package apidoc

import (
	"strings"
	"testing"

	"github.com/axisml/axisml/pkg/openapigen"
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

func TestRouteCoverage(t *testing.T) {
	doc := Document("test")
	want := []struct{ method, path string }{
		{"get", "/healthz"},
		{"get", "/readyz"},
		{"post", "/api/v1/resourcepools"},
		{"get", "/api/v1/resourcepools"},
		{"get", "/api/v1/resourcepools/{pool}"},
		{"patch", "/api/v1/resourcepools/{pool}"},
		{"delete", "/api/v1/resourcepools/{pool}"},
		{"post", "/api/v1/resourcepools/{pool}/units"},
		{"get", "/api/v1/resourcepools/{pool}/units"},
		{"get", "/api/v1/resourcepools/{pool}/units/{unit}"},
		{"patch", "/api/v1/resourcepools/{pool}/units/{unit}"},
		{"delete", "/api/v1/resourcepools/{pool}/units/{unit}"},
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

// TestErrorOverlay pins the cluster-manager error-response overlay:
// 400/401/404/409/422/500 (plus default).
func TestErrorOverlay(t *testing.T) {
	resps := withErrors(map[string]openapigen.Response{"200": {Description: "ok"}})
	for _, code := range []string{"400", "401", "404", "409", "422", "500", "default"} {
		if _, ok := resps[code]; !ok {
			t.Errorf("withErrors missing %q response", code)
		}
	}
}
