package quota

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/axisml/axisml/components/compute/internal/server"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Handler exposes /tenants/:tenant/quotas routes. The tenant resolver
// middleware (provided by the tenant module) is expected to set "tenantID"
// on the gin context before these handlers run.
type Handler struct {
	svc        *Service
	middleware []gin.HandlerFunc
}

// NewHandler builds a quota handler. middleware is applied to the route
// group so the tenant resolver can stash tenantID on the request.
func NewHandler(svc *Service, middleware ...gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, middleware: middleware}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/tenants/:tenant/quotas")
	for _, m := range h.middleware {
		g.Use(m)
	}
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:quota", h.Get)
	g.PATCH("/:quota", h.Update)
	g.DELETE("/:quota", h.Delete)
}

func tenantID(c *gin.Context) (uuid.UUID, error) {
	v, ok := c.Get("tenantID")
	if !ok {
		return uuid.Nil, apperrors.New(apperrors.CodeInternal, "tenant resolver not configured")
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.New(apperrors.CodeInternal, "tenant resolver returned wrong type")
	}
	return id, nil
}

func (h *Handler) Create(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), id, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) List(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items, total, err := h.svc.ListByTenant(c.Request.Context(), id, p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Get(c.Request.Context(), id, c.Param("quota"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Update(c.Request.Context(), id, c.Param("quota"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, c.Param("quota")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
