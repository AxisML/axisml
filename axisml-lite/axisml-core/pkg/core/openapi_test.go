package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// parseSpec renders and decodes the spec so tests can assert on its structure the
// way a consumer's tooling would.
func parseSpec(t *testing.T, opts ...SpecOption) map[string]any {
	t.Helper()
	data, err := OpenAPISpec(SpecYAML, opts...)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(data, &doc))
	return doc
}

func specPaths(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok, "spec has no paths object")
	return paths
}

func infoBlock(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	info, ok := doc["info"].(map[string]any)
	require.True(t, ok, "spec has no info object")
	return info
}

// assertNoDanglingRefs asserts every schema reference in the document resolves.
func assertNoDanglingRefs(t *testing.T, doc map[string]any) {
	t.Helper()
	refs := map[string]bool{}
	collectRefs(doc, refs)
	schemas, _ := nestedMap(doc, "components", "schemas")
	for name := range refs {
		_, ok := schemas[name]
		assert.True(t, ok, "dangling $ref to schema %q", name)
	}
}

func TestOpenAPISpecFull(t *testing.T) {
	doc := parseSpec(t)

	assert.Equal(t, "3.0.3", doc["openapi"])
	info := infoBlock(t, doc)
	assert.Equal(t, "AxisML Core API", info["title"])
	assert.Equal(t, defaultSpecVersion, info["version"])

	paths := specPaths(t, doc)
	for _, p := range []string{
		"/healthz",
		"/readyz",
		"/api/v1/capabilities",
		"/api/v1/resourcepools",
		"/api/v1/namespaces/{namespace}/mlruns",
		"/api/v1/namespaces/{namespace}/mlservices",
		"/api/v1/namespaces/{namespace}/traffic-policies",
		"/api/v1/namespaces/{namespace}/artifacts",
	} {
		assert.Contains(t, paths, p, "expected composite path %q", p)
	}
	assertNoDanglingRefs(t, doc)
}

func TestOpenAPISpecScopedPrunesPathsAndSchemas(t *testing.T) {
	doc := parseSpec(t, WithPathPrefixes(
		"/api/v1/resourcepools",
		"/api/v1/namespaces/{namespace}/mlruns",
	))
	paths := specPaths(t, doc)

	// Kept: the allowlisted surfaces, including their nested sub-paths.
	for _, p := range []string{
		"/api/v1/resourcepools",
		"/api/v1/resourcepools/{pool}",
		"/api/v1/resourcepools/{pool}/units",
		"/api/v1/namespaces/{namespace}/mlruns",
		"/api/v1/namespaces/{namespace}/mlruns/{mlrun}",
	} {
		assert.Contains(t, paths, p, "prefix should keep %q", p)
	}

	// Dropped: everything outside the allowlist, including the Lite probes and
	// the sibling resources under the shared /namespaces/{namespace} prefix.
	for _, p := range []string{
		"/healthz",
		"/api/v1/capabilities",
		"/api/v1/namespaces/{namespace}/mlservices",
		"/api/v1/namespaces/{namespace}/traffic-policies",
		"/api/v1/namespaces/{namespace}/artifacts",
		"/api/v1/tenants",
	} {
		assert.NotContains(t, paths, p, "prefix should drop %q", p)
	}

	// Component closure: entities reachable only from the dropped surfaces are
	// pruned; the ones the kept surfaces reference survive.
	schemas, ok := nestedMap(doc, "components", "schemas")
	require.True(t, ok)
	assert.Contains(t, schemas, "ResourcePool")
	assert.Contains(t, schemas, "MLRun")
	for _, s := range []string{"MLService", "MLServiceList", "TrafficPolicy", "Tenant", "Artifact"} {
		assert.NotContains(t, schemas, s, "unexposed schema %q should be pruned", s)
	}

	// Every surviving $ref still resolves, and tags with no operation are gone.
	assertNoDanglingRefs(t, doc)
	tags, _ := doc["tags"].([]any)
	for _, tag := range tags {
		name, _ := tag.(map[string]any)["name"].(string)
		assert.NotEqual(t, "MLServices", name, "tag for a dropped surface should be pruned")
		assert.NotEqual(t, "Health", name, "tag for a dropped surface should be pruned")
	}
}

func TestOpenAPISpecGlobalHeaderParam(t *testing.T) {
	doc := parseSpec(t,
		WithPathPrefixes("/api/v1/resourcepools"),
		WithGlobalHeaderParam("X-Axisml-User", "Caller identity.", true),
	)

	found := false
	for _, item := range specPaths(t, doc) {
		for _, op := range operationMaps(item) {
			var got map[string]any
			if params, ok := op["parameters"].([]any); ok {
				for _, p := range params {
					pm, _ := p.(map[string]any)
					if pm["in"] == "header" && pm["name"] == "X-Axisml-User" {
						got = pm
					}
				}
			}
			require.NotNil(t, got, "every kept operation must carry the header param")
			assert.Equal(t, true, got["required"])
			assert.Equal(t, "Caller identity.", got["description"])
			found = true
		}
	}
	assert.True(t, found, "expected at least one operation")
}

func TestOpenAPISpecWithInfo(t *testing.T) {
	doc := parseSpec(t, WithInfo("Training Backend API", "2.1.0", "Proxied axisml surface."))
	info := infoBlock(t, doc)
	assert.Equal(t, "Training Backend API", info["title"])
	assert.Equal(t, "2.1.0", info["version"])
	assert.Equal(t, "Proxied axisml surface.", info["description"])
}

func TestOpenAPISpecWithVersion(t *testing.T) {
	doc := parseSpec(t, WithVersion("9.9.9"))
	info := infoBlock(t, doc)
	assert.Equal(t, "9.9.9", info["version"])
	assert.Equal(t, "AxisML Core API", info["title"], "WithVersion leaves title at the default")
}

func TestOpenAPISpecJSON(t *testing.T) {
	data, err := OpenAPISpec(SpecJSON)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc), "SpecJSON output must be valid JSON")
	assert.Equal(t, "3.0.3", doc["openapi"])
	assert.Contains(t, doc["paths"], "/api/v1/resourcepools")
}

func TestOpenAPISpecDeterministic(t *testing.T) {
	// Verbatim path.
	a, err := OpenAPISpec(SpecYAML)
	require.NoError(t, err)
	b, err := OpenAPISpec(SpecYAML)
	require.NoError(t, err)
	assert.Equal(t, a, b, "verbatim output must be byte-stable")

	// Transformed path.
	c, err := OpenAPISpec(SpecYAML, WithPathPrefixes("/api/v1/resourcepools"))
	require.NoError(t, err)
	d, err := OpenAPISpec(SpecYAML, WithPathPrefixes("/api/v1/resourcepools"))
	require.NoError(t, err)
	assert.Equal(t, c, d, "scoped output must be byte-stable")
}

func TestOpenAPISpecUnknownFormat(t *testing.T) {
	_, err := OpenAPISpec(SpecFormat(99))
	require.Error(t, err)
}

// TestOpenAPIHandlerServesCachedSpec exercises the runtime serving path without a
// full App: openAPISpec only touches the spec-cache fields, so a zero-value App
// is enough to serve both formats and populate the cache.
func TestOpenAPIHandlerServesCachedSpec(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := &App{}
	e := gin.New()
	e.GET("/openapi.yaml", a.openapiHandler(SpecYAML, "application/yaml"))
	e.GET("/openapi.json", a.openapiHandler(SpecJSON, "application/json"))

	for _, tc := range []struct{ path, ctype string }{
		{"/openapi.yaml", "application/yaml"},
		{"/openapi.json", "application/json"},
	} {
		w := httptest.NewRecorder()
		e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, tc.ctype, w.Header().Get("Content-Type"))
		assert.NotEmpty(t, w.Body.Bytes())
	}

	require.NoError(t, a.specErr)
	assert.NotEmpty(t, a.specYAML)
	assert.NotEmpty(t, a.specJSON)
}
