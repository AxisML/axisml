package tenant

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes /api/v1/namespaces routes. The "namespace" URL token is
// the tenant identifier (= tenants.name).
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:namespace", h.Get)
	g.PATCH("/:namespace", h.Patch)
	g.DELETE("/:namespace", h.Delete)
	g.POST("/:namespace/restore", h.Restore)

	q := g.Group("/:namespace/quotas")
	q.GET("", h.ListQuotas)
	q.POST("", h.AddQuota)
	q.PATCH("/:pool/:quotaName", h.PatchQuota)
	q.DELETE("/:pool/:quotaName", h.DeleteQuota)
}

func (h *Handler) ListQuotas(c *gin.Context) {
	qs, err := h.svc.ListQuotas(c.Request.Context(), c.Param("namespace"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": qs})
}

func (h *Handler) AddQuota(c *gin.Context) {
	var in QuotaPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	q, err := h.svc.AddQuota(c.Request.Context(), c.Param("namespace"), in, callerUser(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, q)
}

func (h *Handler) PatchQuota(c *gin.Context) {
	var in QuotaPatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	q, err := h.svc.PatchQuota(c.Request.Context(),
		c.Param("namespace"), c.Param("pool"), c.Param("quotaName"), in, callerUser(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, q)
}

func (h *Handler) DeleteQuota(c *gin.Context) {
	if err := h.svc.DeleteQuota(c.Request.Context(),
		c.Param("namespace"), c.Param("pool"), c.Param("quotaName"), callerUser(c)); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Restore(c *gin.Context) {
	v, err := h.svc.Restore(c.Request.Context(), c.Param("namespace"), callerUser(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), in, callerUser(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) List(c *gin.Context) {
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	clause, args, err := server.JSONLabelsSQL("labels", c.Query("labelSelector"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.List(c.Request.Context(), p.Limit, p.Offset, clause, args)
	if err != nil {
		_ = c.Error(err)
		return
	}
	v.ContinueToken = server.EncodeContinue(p.Offset, len(v.Items), v.Total)
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Patch(c *gin.Context) {
	var in PatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), in, callerUser(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace")); err != nil {
		_ = c.Error(err)
		return
	}
	// Design yaml expects 200 + the tenant body (now phase=Deleting) so
	// the caller has the tombstone state in one trip. Use the unscoped
	// lookup so we can return the just-soft-deleted row.
	v, err := h.svc.GetIncludingDeleted(c.Request.Context(), c.Param("namespace"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func callerUser(c *gin.Context) string {
	return c.GetHeader("X-Axisml-User")
}
