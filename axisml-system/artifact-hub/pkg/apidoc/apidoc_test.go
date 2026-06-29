package apidoc

import (
	"strings"
	"testing"

	"github.com/axisml/axisml/pkg/openapigen"

	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
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
		// One representative kind (models); the loop in main.go renders all
		// three kinds (models/datasets/images) symmetrically.
		{"get", "/api/v1/namespaces/{namespace}/models"},
		{"post", "/api/v1/namespaces/{namespace}/models/{name}"},
		{"get", "/api/v1/namespaces/{namespace}/models/{name}"},
		{"get", "/api/v1/namespaces/{namespace}/models/{name}/{version}"},
		{"patch", "/api/v1/namespaces/{namespace}/models/{name}/{version}"},
		{"delete", "/api/v1/namespaces/{namespace}/models/{name}/{version}"},
		{"post", "/api/v1/namespaces/{namespace}/models/{name}/{version}/complete"},
		{"get", "/api/v1/namespaces/{namespace}/models/{name}/{version}/resolve"},
		// Datasets / images symmetry spot-check.
		{"post", "/api/v1/namespaces/{namespace}/datasets/{name}"},
		{"post", "/api/v1/namespaces/{namespace}/images/{name}"},
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

func TestProblemEnumIsSourced(t *testing.T) {
	doc := Document("test")
	prob, ok := doc.Components.Schemas["ArtifactHubError"]
	if !ok {
		t.Fatal("ArtifactHubError schema missing")
	}
	codeProp := prob.Properties["code"]
	if codeProp == nil {
		t.Fatal("ArtifactHubError.code missing")
	}
	if len(codeProp.Enum) != len(apperrors.AllCodes()) {
		t.Errorf("ArtifactHubError.code enum len = %d, want %d", len(codeProp.Enum), len(apperrors.AllCodes()))
	}
}

// TestErrorOverlayCoversArtifactsCodes pins the standard error-response
// overlay. Artifacts swaps 422 (no quotas) for 410 Gone vs the compute set.
func TestErrorOverlayCoversArtifactsCodes(t *testing.T) {
	resps := withErrors(map[string]openapigen.Response{"200": {Description: "ok"}})
	for _, code := range []string{"400", "401", "403", "404", "409", "410", "412", "503", "default"} {
		if _, ok := resps[code]; !ok {
			t.Errorf("withErrors missing %q response", code)
		}
	}
}
