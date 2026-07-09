package audit

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
)

func init() { gin.SetMode(gin.TestMode) }

// ctxIdentityKey is the (unexported) gin context key auth.setIdentity uses;
// mirrored here so tests can inject an identity that auth.Current will read.
const identityCtxKey = "axisml.identity"

// newCtx builds a gin context for derive: it only needs the request method,
// URL path and route params (derive reads the body from its argument, not the
// request body).
func newCtx(method, path string, params gin.Params, header map[string]string, id *auth.Identity) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(method, path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	c.Request = req
	c.Params = params
	if id != nil {
		c.Set(identityCtxKey, id)
	}
	return c
}

func TestIsMutating(t *testing.T) {
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		assert.True(t, isMutating(m), m)
	}
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions, "TRACE"} {
		assert.False(t, isMutating(m), m)
	}
}

func TestFirstParam(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "name", Value: "myjob"}, {Key: "pool", Value: "gpu"}}

	assert.Equal(t, "myjob", firstParam(c, "run", "name", "pool"))
	assert.Equal(t, "gpu", firstParam(c, "run", "pool"))
	assert.Equal(t, "", firstParam(c, "run", "version"))
	assert.Equal(t, "", firstParam(c))
}

func TestFieldFromJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
		keys []string
		want string
	}{
		{"first key present", `{"name":"a","identifier":"b"}`, []string{"name", "identifier"}, "a"},
		{"second key present", `{"identifier":"b"}`, []string{"name", "identifier"}, "b"},
		{"none present", `{"other":"x"}`, []string{"name", "identifier"}, ""},
		{"non-string value skipped", `{"name":123,"identifier":"b"}`, []string{"name", "identifier"}, "b"},
		{"empty-string value skipped", `{"name":"","identifier":"b"}`, []string{"name", "identifier"}, "b"},
		{"empty body", ``, []string{"name"}, ""},
		{"invalid json", `{not json`, []string{"name"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fieldFromJSON([]byte(tt.body), tt.keys...))
		})
	}
}

func TestNonEmpty(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, nonEmpty([]string{"a", "", "b", ""}))
	assert.Equal(t, []string{"x"}, nonEmpty([]string{"x"}))
	assert.Empty(t, nonEmpty([]string{"", ""}))
	assert.Empty(t, nonEmpty([]string{}))
}

func TestContains(t *testing.T) {
	in := []string{"tenants", "acme", "quotas", "gpu"}
	assert.True(t, contains(in, "quotas"))
	assert.True(t, contains(in, "tenants"))
	assert.False(t, contains(in, "members"))
	assert.False(t, contains(nil, "x"))
}

func TestDerive(t *testing.T) {
	const tenantHdr = auth.HeaderTenant
	id := &auth.Identity{Username: "alice"}

	tests := []struct {
		name       string
		method     string
		path       string
		params     gin.Params
		body       string
		header     map[string]string
		identity   *auth.Identity
		wantNil    bool
		wantKind   string
		wantAction string
		wantName   string
		wantTenant string
		wantActor  string
	}{
		{
			name:   "POST create job, name from JSON body, tenant from header, actor from identity",
			method: http.MethodPost, path: "/api/v1/jobs",
			body: `{"name":"myjob"}`, header: map[string]string{tenantHdr: "acme"}, identity: id,
			wantKind: "job", wantAction: "created", wantName: "myjob", wantTenant: "acme", wantActor: "alice",
		},
		{
			name:   "PATCH update job, name from path param",
			method: http.MethodPatch, path: "/api/v1/jobs/myjob",
			params: gin.Params{{Key: "name", Value: "myjob"}}, header: map[string]string{tenantHdr: "acme"},
			wantKind: "job", wantAction: "updated", wantName: "myjob", wantTenant: "acme",
		},
		{
			name:   "PUT update job maps to updated",
			method: http.MethodPut, path: "/api/v1/jobs/myjob",
			params: gin.Params{{Key: "name", Value: "myjob"}}, header: map[string]string{tenantHdr: "acme"},
			wantKind: "job", wantAction: "updated", wantName: "myjob", wantTenant: "acme",
		},
		{
			name:   "DELETE job maps to deleted",
			method: http.MethodDelete, path: "/api/v1/jobs/myjob",
			params: gin.Params{{Key: "name", Value: "myjob"}}, header: map[string]string{tenantHdr: "acme"},
			wantKind: "job", wantAction: "deleted", wantName: "myjob", wantTenant: "acme",
		},
		{
			name:   "run kind refined by run path param, cancel sub-action",
			method: http.MethodPost, path: "/api/v1/jobs/myjob/runs/r1/cancel",
			params:   gin.Params{{Key: "name", Value: "myjob"}, {Key: "run", Value: "r1"}},
			header:   map[string]string{tenantHdr: "acme"},
			wantKind: "run", wantAction: "canceled", wantName: "r1", wantTenant: "acme",
		},
		{
			// last segment "runs" refines kind to run; the parent :name path
			// param resolves the subject name (it wins over the JSON body).
			name:   "run kind refined when last segment is runs",
			method: http.MethodPost, path: "/api/v1/jobs/myjob/runs",
			params: gin.Params{{Key: "name", Value: "myjob"}}, body: `{"name":"r9"}`,
			header:   map[string]string{tenantHdr: "acme"},
			wantKind: "run", wantAction: "created", wantName: "myjob", wantTenant: "acme",
		},
		{
			name:   "quota PATCH: kind refined, name from pool param, tenant from tenants path param",
			method: http.MethodPatch, path: "/api/v1/tenants/acme/quotas/gpu-pool",
			params:   gin.Params{{Key: "name", Value: "acme"}, {Key: "pool", Value: "gpu-pool"}},
			wantKind: "quota", wantAction: "updated", wantName: "gpu-pool", wantTenant: "acme",
		},
		{
			name:   "quota POST: name from JSON pool",
			method: http.MethodPost, path: "/api/v1/tenants/acme/quotas",
			params: gin.Params{{Key: "name", Value: "acme"}}, body: `{"pool":"gpu-pool"}`,
			wantKind: "quota", wantAction: "created", wantName: "gpu-pool", wantTenant: "acme",
		},
		{
			name:   "member DELETE: kind refined, name from userId param",
			method: http.MethodDelete, path: "/api/v1/tenants/acme/members/u-42",
			params:   gin.Params{{Key: "name", Value: "acme"}, {Key: "userId", Value: "u-42"}},
			wantKind: "member", wantAction: "deleted", wantName: "u-42", wantTenant: "acme",
		},
		{
			name:   "member POST: name from JSON userId",
			method: http.MethodPost, path: "/api/v1/tenants/acme/members",
			params: gin.Params{{Key: "name", Value: "acme"}}, body: `{"userId":"u-99"}`,
			wantKind: "member", wantAction: "created", wantName: "u-99", wantTenant: "acme",
		},
		{
			name:   "tenant from JSON body fallback",
			method: http.MethodPost, path: "/api/v1/jobs",
			body:     `{"name":"j1","tenantName":"acme"}`,
			wantKind: "job", wantAction: "created", wantName: "j1", wantTenant: "acme",
		},
		{
			name:   "tenant from identifier param fallback",
			method: http.MethodDelete, path: "/api/v1/experiments/exp1",
			params:   gin.Params{{Key: "identifier", Value: "exp1"}},
			wantKind: "experiment", wantAction: "deleted", wantName: "exp1", wantTenant: "exp1",
		},
		{
			name:   "no identity yields empty actor",
			method: http.MethodPost, path: "/api/v1/jobs",
			body: `{"name":"j"}`, header: map[string]string{tenantHdr: "acme"},
			wantKind: "job", wantAction: "created", wantName: "j", wantTenant: "acme", wantActor: "",
		},
		{
			name:   "unknown leading segment yields nil",
			method: http.MethodPost, path: "/api/v1/unknown/x",
			wantNil: true,
		},
		{
			name:   "empty path yields nil",
			method: http.MethodPost, path: "/api/v1/",
			wantNil: true,
		},
		{
			name:   "no resolvable name yields nil",
			method: http.MethodPost, path: "/api/v1/jobs",
			wantNil: true,
		},
		{
			name:   "non-mutating method yields nil",
			method: http.MethodGet, path: "/api/v1/jobs/myjob",
			params:  gin.Params{{Key: "name", Value: "myjob"}},
			wantNil: true,
		},
	}

	r := &Recorder{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCtx(tt.method, tt.path, tt.params, tt.header, tt.identity)
			ev := r.derive(c, []byte(tt.body))
			if tt.wantNil {
				assert.Nil(t, ev)
				return
			}
			require.NotNil(t, ev)
			assert.Equal(t, tt.wantKind, ev.Kind)
			assert.Equal(t, tt.wantAction, ev.Action)
			assert.Equal(t, tt.wantName, ev.Name)
			assert.Equal(t, tt.wantTenant, ev.Tenant)
			assert.Equal(t, tt.wantActor, ev.Actor)
		})
	}
}

// TestDeriveActionBySuffix exercises every POST sub-action suffix mapping.
func TestDeriveActionBySuffix(t *testing.T) {
	want := map[string]string{
		"cancel":   "canceled",
		"scale":    "scaled",
		"stop":     "stopped",
		"start":    "started",
		"split":    "split",
		"promote":  "promoted",
		"rollback": "rolled-back",
		"suspend":  "suspended",
		"resume":   "resumed",
	}
	r := &Recorder{}
	for suffix, action := range want {
		t.Run(suffix, func(t *testing.T) {
			c := newCtx(http.MethodPost, "/api/v1/mlservices/svc1/"+suffix,
				gin.Params{{Key: "name", Value: "svc1"}},
				map[string]string{auth.HeaderTenant: "acme"}, nil)
			ev := r.derive(c, nil)
			require.NotNil(t, ev)
			assert.Equal(t, "service", ev.Kind)
			assert.Equal(t, action, ev.Action)
			assert.Equal(t, "svc1", ev.Name)
		})
	}
}

func TestRespCapture_BuffersSmallWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	w := &respCapture{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}

	n, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", w.buf.String())
	assert.Equal(t, "hello", rec.Body.String())
}

func TestRespCapture_WriteString(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	w := &respCapture{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}

	n, err := w.WriteString("hi")
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, "hi", w.buf.String())
	assert.Equal(t, "hi", rec.Body.String())
}

func TestRespCapture_StopsBufferingAtLimit(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	w := &respCapture{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}

	// First write fills the buffer to exactly the cap.
	_, err := w.Write(bytes.Repeat([]byte("a"), captureLimit))
	require.NoError(t, err)
	assert.Equal(t, captureLimit, w.buf.Len())

	// Once at the cap, further bytes pass through but are no longer buffered.
	_, err = w.Write(bytes.Repeat([]byte("b"), 100))
	require.NoError(t, err)
	assert.Equal(t, captureLimit, w.buf.Len())
	assert.Equal(t, captureLimit+100, rec.Body.Len())
}
