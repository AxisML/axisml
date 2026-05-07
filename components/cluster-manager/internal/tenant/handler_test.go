package tenant

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(tenantv1alpha1.AddToScheme(s))
	return s
}

func newRouter(h *Handler) *gin.Engine {
	r := gin.New()
	api := r.Group("/api/v1")
	h.Register(api)
	return r
}

func seedTenant() *tenantv1alpha1.Tenant {
	return &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "alpha",
			Labels: map[string]string{tenantv1alpha1.LabelTenantID: "tnt-uuid-1"},
		},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "alpha-ns"},
			Quotas: []tenantv1alpha1.QuotaSpec{
				{Pool: "default", Name: "main", Max: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")}},
			},
		},
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/missing", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var p srv.Problem
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &p))
	assert.Equal(t, "tenant not found", p.Title)
	assert.Equal(t, http.StatusNotFound, p.Status)
}

func TestHandler_Get_Found(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/alpha", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp srv.TenantResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "alpha", resp.Name)
	assert.Equal(t, "tnt-uuid-1", resp.ID)
}

func TestHandler_Create_Conflict(t *testing.T) {
	scheme := newScheme(t)
	existing := seedTenant()
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	h := &Handler{Client: c, NamespaceDenylist: []string{"kube-system"}}
	r := newRouter(h)

	body := srv.CreateTenantRequest{
		Name:      "alpha",
		Namespace: srv.NamespaceSpec{Name: "alpha-ns"},
	}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Create_BadJSON(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Create_NamespaceOnDenylist(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	h := &Handler{Client: c, NamespaceDenylist: []string{"kube-system"}}
	r := newRouter(h)

	body := srv.CreateTenantRequest{
		Name:      "alpha",
		Namespace: srv.NamespaceSpec{Name: "kube-system"},
	}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Delete_IdempotentOnMissing(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/never-existed", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Suspend_TogglesAndIsIdempotent(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	// First suspend → 200, suspended=true.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/alpha/suspend", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp srv.TenantResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Suspended)

	// Second suspend → still 200, no transition required.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/alpha/suspend", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Unsuspend → 200, suspended=false.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/alpha/unsuspend", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Suspended)
}

func TestHandler_AddQuota_Conflict(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	body := srv.QuotaSpec{Pool: "default", Name: "main", Max: map[string]string{"cpu": "4"}}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/alpha/quotas", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_AddQuota_Created(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	body := srv.QuotaSpec{Pool: "gpu", Name: "high", Max: map[string]string{"nvidia.com/gpu": "8"}}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/alpha/quotas", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp srv.TenantResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Quotas, 2)
}

func TestHandler_UpdateQuota_NotFound(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	body := srv.QuotaSpec{Max: map[string]string{"cpu": "8"}}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/alpha/quotas/default/missing", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateQuota_BadRequestNoMax(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	body := srv.QuotaSpec{Min: map[string]string{"cpu": "1"}} // no max
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/alpha/quotas/default/main", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteQuota_NoContent(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/alpha/quotas/default/main", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_DeleteQuota_NoContentEvenWhenMissing(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(seedTenant()).Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/alpha/quotas/default/missing", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code,
		"DeleteQuota on missing entry should be idempotent (204)")
}

func TestHandler_ListQuotas_ReturnsBothSpecAndStatus(t *testing.T) {
	scheme := newScheme(t)
	tnt := seedTenant()
	tnt.Status.Quotas = []tenantv1alpha1.QuotaStatus{{Pool: "default", Name: "main", Ready: true}}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&tenantv1alpha1.Tenant{}).
		WithObjects(tnt).
		Build()
	h := &Handler{Client: c}
	r := newRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/alpha/quotas", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp["spec"])
	// status may be empty if fake client clears Status; check key exists.
	_, ok := resp["status"]
	assert.True(t, ok, "response should include status key")
}

func TestIndexOfQuota(t *testing.T) {
	qs := []tenantv1alpha1.QuotaSpec{
		{Pool: "a", Name: "x"},
		{Pool: "b", Name: "y"},
	}
	assert.Equal(t, 0, indexOfQuota(qs, "a", "x"))
	assert.Equal(t, 1, indexOfQuota(qs, "b", "y"))
	assert.Equal(t, -1, indexOfQuota(qs, "c", "z"))
	assert.Equal(t, -1, indexOfQuota(nil, "a", "x"))
}
