package repo

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/artifacts/internal/auth"
	"github.com/axisml/axisml/components/artifacts/internal/server"
	"github.com/axisml/axisml/components/artifacts/internal/tenantresolver"
)

// Handler exposes /tenants/{tenant}/repos and /public/repos routes.
type Handler struct {
	svc        *Service
	tenantMW   gin.HandlerFunc
	publicStub gin.HandlerFunc
}

// NewHandler constructs a Handler. The tenantMW resolves :tenant to a
// tenant row + namespace before the handler runs (design §6.2).
func NewHandler(svc *Service, tenantMW gin.HandlerFunc) *Handler {
	return &Handler{
		svc:        svc,
		tenantMW:   tenantMW,
		publicStub: publicNotFound,
	}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	tenantGrp := rg.Group("/tenants/:tenant", h.tenantMW)
	tenantGrp.POST("/repos", h.Create)
	tenantGrp.GET("/repos", h.List)
	tenantGrp.GET("/repos/:kind/:name", h.Get)
	tenantGrp.PATCH("/repos/:kind/:name", h.Update)
	tenantGrp.DELETE("/repos/:kind/:name", h.Delete)

	// Public space is reserved for Phase 2 (design §8.2). Stub returns 404
	// for any path under /public/... so the URL surface is reserved.
	pub := rg.Group("/public")
	pub.Any("/*any", h.publicStub)
}

func publicNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": "public space is not available in MVP",
	})
}

// Create handles POST /tenants/{tenant}/repos.
func (h *Handler) Create(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	user := auth.User(c.Request.Context())
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.svc.Create(c.Request.Context(), tenant, user, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, toView(row))
}

// List handles GET /tenants/{tenant}/repos.
func (h *Handler) List(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	kind := c.Query("kind")
	rows, total, err := h.svc.List(c.Request.Context(), tenant, kind, p.Limit, p.Offset)
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

// Get handles GET /tenants/{tenant}/repos/{kind}/{name}.
func (h *Handler) Get(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	row, err := h.svc.Get(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

// Update handles PATCH /tenants/{tenant}/repos/{kind}/{name}.
func (h *Handler) Update(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.svc.Update(c.Request.Context(), tenant, c.Param("kind"), c.Param("name"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

// Delete handles DELETE /tenants/{tenant}/repos/{kind}/{name}.
func (h *Handler) Delete(c *gin.Context) {
	tenant := c.GetString(tenantresolver.CtxKeyTenantName)
	if err := h.svc.MarkDeleting(c.Request.Context(), tenant, c.Param("kind"), c.Param("name")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusAccepted)
}
