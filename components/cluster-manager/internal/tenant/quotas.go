package tenant

import (
	"net/http"

	"github.com/gin-gonic/gin"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
)

// AddQuota appends one entry to spec.quotas[]. Conflicts on
// (pool, name) → 409.
func (h *Handler) AddQuota(c *gin.Context) {
	ctx := c.Request.Context()
	t, err := h.fetch(c)
	if err != nil {
		return
	}
	var req srv.QuotaSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "invalid JSON body", err.Error())
		return
	}
	if err := srv.ValidateQuota(req); err != nil {
		writeProblem(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}
	for _, q := range t.Spec.Quotas {
		if q.Pool == req.Pool && q.Name == req.Name {
			writeProblem(c, http.StatusConflict, "quota already exists", "(pool,name) collision")
			return
		}
	}
	t.Spec.Quotas = append(t.Spec.Quotas, tenantv1alpha1.QuotaSpec{
		Pool: req.Pool,
		Name: req.Name,
		Min:  srv.ParseResourceList(req.Min),
		Max:  srv.ParseResourceList(req.Max),
	})
	if err := h.Client.Update(ctx, t); err != nil {
		writeProblem(c, http.StatusInternalServerError, "update failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, srv.FromTenant(t))
}

// UpdateQuota mutates min / max of an existing entry. Pool / name
// stay frozen (they're the URL identifier).
func (h *Handler) UpdateQuota(c *gin.Context) {
	ctx := c.Request.Context()
	t, err := h.fetch(c)
	if err != nil {
		return
	}
	pool, name := c.Param("pool"), c.Param("quota")
	idx := indexOfQuota(t.Spec.Quotas, pool, name)
	if idx < 0 {
		writeProblem(c, http.StatusNotFound, "quota not found", pool+"/"+name)
		return
	}
	var req srv.QuotaSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "invalid JSON body", err.Error())
		return
	}
	if req.Max == nil {
		writeProblem(c, http.StatusBadRequest, "validation failed", "max is required")
		return
	}
	t.Spec.Quotas[idx].Min = srv.ParseResourceList(req.Min)
	t.Spec.Quotas[idx].Max = srv.ParseResourceList(req.Max)
	if err := h.Client.Update(ctx, t); err != nil {
		writeProblem(c, http.StatusInternalServerError, "update failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, srv.FromTenant(t))
}

// DeleteQuota removes an entry from spec.quotas[]. Tenant-operator
// will explicitly Delete the matching ElasticQuota CR on the next
// reconcile.
func (h *Handler) DeleteQuota(c *gin.Context) {
	ctx := c.Request.Context()
	t, err := h.fetch(c)
	if err != nil {
		return
	}
	pool, name := c.Param("pool"), c.Param("quota")
	idx := indexOfQuota(t.Spec.Quotas, pool, name)
	if idx < 0 {
		c.Status(http.StatusNoContent)
		return
	}
	t.Spec.Quotas = append(t.Spec.Quotas[:idx], t.Spec.Quotas[idx+1:]...)
	if err := h.Client.Update(ctx, t); err != nil {
		writeProblem(c, http.StatusInternalServerError, "update failed", err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// ListQuotas returns spec.quotas[] + status.quotas[] for the tenant.
func (h *Handler) ListQuotas(c *gin.Context) {
	t, err := h.fetch(c)
	if err != nil {
		return
	}
	resp := srv.FromTenant(t)
	c.JSON(http.StatusOK, gin.H{
		"spec":   resp.Quotas,
		"status": resp.Status.Quotas,
	})
}

func indexOfQuota(qs []tenantv1alpha1.QuotaSpec, pool, name string) int {
	for i, q := range qs {
		if q.Pool == pool && q.Name == name {
			return i
		}
	}
	return -1
}
