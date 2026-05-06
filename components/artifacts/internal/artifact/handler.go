package artifact

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/artifacts/internal/auth"
	"github.com/axisml/axisml/components/artifacts/internal/server"
	"github.com/axisml/axisml/components/artifacts/internal/tenantresolver"
)

// Handler exposes the artifact-scoped routes under
// /api/v1/tenants/{tenant}/repos/{kind}/{name}/artifacts/...
type Handler struct {
	svc      *Service
	tenantMW gin.HandlerFunc
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service, tenantMW gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, tenantMW: tenantMW}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/tenants/:tenant", h.tenantMW)
	g.POST("/repos/:kind/:name/artifacts", h.Initiate)
	g.GET("/repos/:kind/:name/artifacts", h.List)
	g.GET("/repos/:kind/:name/artifacts/:version", h.Get)
	g.POST("/repos/:kind/:name/artifacts/:version/complete", h.Complete)
	g.GET("/repos/:kind/:name/artifacts/:version/resolve", h.Resolve)
	g.DELETE("/repos/:kind/:name/artifacts/:version", h.Delete)
}

// Initiate handles POST /repos/{kind}/{name}/artifacts (two-phase write step 1).
func (h *Handler) Initiate(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	user := auth.User(c.Request.Context())
	var in InitiateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	res, err := h.svc.Initiate(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), user, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

// List handles GET /repos/{kind}/{name}/artifacts.
func (h *Handler) List(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	rows, total, err := h.svc.List(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), c.Query("status"), p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]View, 0, len(rows))
	for i := range rows {
		items = append(items, toView(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// Get handles GET /repos/{kind}/{name}/artifacts/{version}.
func (h *Handler) Get(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	row, err := h.svc.Get(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), c.Param("version"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

// Complete handles POST /repos/{kind}/{name}/artifacts/{version}/complete.
func (h *Handler) Complete(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	var in CompleteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.svc.Complete(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), c.Param("version"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

// Resolve handles GET /repos/{kind}/{name}/artifacts/{version}/resolve.
func (h *Handler) Resolve(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	usage := c.Query("usage")
	res, err := h.svc.Resolve(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), c.Param("version"), usage)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Delete handles DELETE /repos/{kind}/{name}/artifacts/{version}.
func (h *Handler) Delete(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	if err := h.svc.MarkDeleting(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), c.Param("version")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusAccepted)
}
