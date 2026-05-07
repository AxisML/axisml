// Package tenant holds the REST handlers that mutate Tenant CRs.
//
// All handlers are stateless — they translate request bodies to K8s API
// calls and stream back what the API server returns.
package tenant

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
)

// Handler bundles the Tenant REST endpoints.
type Handler struct {
	Client            client.Client
	NamespaceDenylist []string
}

// Register wires the tenant routes into a Gin engine. The /quotas
// sub-routes live in this package too because they share the same
// permissions and CR target.
func (h *Handler) Register(r *gin.RouterGroup) {
	r.POST("/tenants", h.Create)
	r.GET("/tenants", h.List)
	r.GET("/tenants/:name", h.Get)
	r.PATCH("/tenants/:name", h.Patch)
	r.DELETE("/tenants/:name", h.Delete)
	r.POST("/tenants/:name/suspend", h.suspend(true))
	r.POST("/tenants/:name/unsuspend", h.suspend(false))
	r.POST("/tenants/:name/quotas", h.AddQuota)
	r.PATCH("/tenants/:name/quotas/:pool/:quota", h.UpdateQuota)
	r.DELETE("/tenants/:name/quotas/:pool/:quota", h.DeleteQuota)
	r.GET("/tenants/:name/quotas", h.ListQuotas)
}

// Create handles POST /api/v1/tenants.
func (h *Handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	var req srv.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "invalid JSON body", err.Error())
		return
	}
	if err := srv.ValidateName("name", req.Name); err != nil {
		writeProblem(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}
	if err := srv.ValidateNamespace(req.Namespace.Name, h.NamespaceDenylist); err != nil {
		writeProblem(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}
	for _, q := range req.Quotas {
		if err := srv.ValidateQuota(q); err != nil {
			writeProblem(c, http.StatusBadRequest, "validation failed", err.Error())
			return
		}
	}

	t := &tenantv1alpha1.Tenant{Spec: req.ToTenantSpec()}
	srv.EnsureMetadata(t, req.Name, uuid.NewString())

	if err := h.Client.Create(ctx, t); err != nil {
		writeAPIErr(c, "create", err)
		return
	}
	c.JSON(http.StatusCreated, srv.FromTenant(t))
}

// Get handles GET /api/v1/tenants/{name}.
func (h *Handler) Get(c *gin.Context) {
	t, err := h.fetch(c)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, srv.FromTenant(t))
}

// List handles GET /api/v1/tenants.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()
	list := srv.NewTenantList()
	opts := []client.ListOption{}
	if cont := c.Query("continue"); cont != "" {
		opts = append(opts, client.Continue(cont))
	}
	if limit := c.Query("limit"); limit != "" {
		// Unparseable limit is silently dropped — K8s API server falls back
		// to default page size.
		if n, err := strconv.ParseInt(limit, 10, 64); err == nil && n > 0 {
			opts = append(opts, client.Limit(n))
		}
	}
	if err := h.Client.List(ctx, list, opts...); err != nil {
		writeProblem(c, http.StatusInternalServerError, "list failed", err.Error())
		return
	}
	out := srv.ListTenantsResponse{
		Items:    make([]srv.TenantResponse, 0, len(list.Items)),
		Continue: list.Continue,
	}
	for i := range list.Items {
		out.Items = append(out.Items, srv.FromTenant(&list.Items[i]))
	}
	c.JSON(http.StatusOK, out)
}

// Patch handles PATCH /api/v1/tenants/{name}.
func (h *Handler) Patch(c *gin.Context) {
	ctx := c.Request.Context()
	t, err := h.fetch(c)
	if err != nil {
		return
	}
	var req srv.PatchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "invalid JSON body", err.Error())
		return
	}
	srv.ApplyPatchToTenant(t, req)
	if err := h.Client.Update(ctx, t); err != nil {
		writeAPIErr(c, "update", err)
		return
	}
	c.JSON(http.StatusOK, srv.FromTenant(t))
}

// Delete handles DELETE /api/v1/tenants/{name}. NotFound is treated as
// success (idempotent).
func (h *Handler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	t := &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: c.Param("name")}}
	if err := h.Client.Delete(ctx, t); err != nil && !apierrors.IsNotFound(err) {
		writeAPIErr(c, "delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// suspend toggles spec.suspended. The factory is shared between
// /suspend and /unsuspend.
func (h *Handler) suspend(target bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		t, err := h.fetch(c)
		if err != nil {
			return
		}
		if t.Spec.Suspended == target {
			c.JSON(http.StatusOK, srv.FromTenant(t))
			return
		}
		t.Spec.Suspended = target
		if err := h.Client.Update(ctx, t); err != nil {
			writeAPIErr(c, "update", err)
			return
		}
		c.JSON(http.StatusOK, srv.FromTenant(t))
	}
}

// fetch is a shared helper that loads a Tenant by URL :name with full
// error response writing. Returns nil + early-exit on failure.
func (h *Handler) fetch(c *gin.Context) (*tenantv1alpha1.Tenant, error) {
	ctx := c.Request.Context()
	t := &tenantv1alpha1.Tenant{}
	if err := h.Client.Get(ctx, types.NamespacedName{Name: c.Param("name")}, t); err != nil {
		if apierrors.IsNotFound(err) {
			writeProblem(c, http.StatusNotFound, "tenant not found", err.Error())
		} else {
			writeProblem(c, http.StatusInternalServerError, "get failed", err.Error())
		}
		return nil, err
	}
	return t, nil
}

func writeProblem(c *gin.Context, status int, title, detail string) {
	c.JSON(status, srv.Problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	})
}

// writeAPIErr maps a K8s API error onto an RFC7807 response. NotFound is
// caller-routed (we want the URL :name in the title), so it isn't handled
// here; callers should check apierrors.IsNotFound first when relevant.
func writeAPIErr(c *gin.Context, op string, err error) {
	switch {
	case apierrors.IsAlreadyExists(err):
		writeProblem(c, http.StatusConflict, "resource already exists", err.Error())
	case apierrors.IsConflict(err):
		// resourceVersion mismatch: someone else mutated the Tenant
		// concurrently. Caller can retry with a fresh GET.
		writeProblem(c, http.StatusConflict, "concurrent modification", err.Error())
	case apierrors.IsInvalid(err) || apierrors.IsBadRequest(err):
		writeProblem(c, http.StatusBadRequest, "invalid request", err.Error())
	case apierrors.IsForbidden(err):
		writeProblem(c, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeProblem(c, http.StatusInternalServerError, op+" failed", err.Error())
	}
}
