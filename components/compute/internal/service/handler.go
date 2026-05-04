package service

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/axisml/axisml/components/compute/internal/server"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Handler exposes /tenants/:tenant/services routes.
type Handler struct {
	svc        *Module
	middleware []gin.HandlerFunc
}

// NewHandler builds the HTTP handler bound to the supplied Module.
func NewHandler(svc *Module, middleware ...gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, middleware: middleware}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/tenants/:tenant/services")
	for _, m := range h.middleware {
		g.Use(m)
	}
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:service", h.Get)
	g.POST("/:service/scale", h.Scale)
	g.DELETE("/:service", h.Delete)
}

func tenantID(c *gin.Context) (uuid.UUID, error) {
	v, ok := c.Get("tenantID")
	if !ok {
		return uuid.Nil, apperrors.New(apperrors.CodeInternal, "tenant resolver not configured")
	}
	id, _ := v.(uuid.UUID)
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
	items, total, err := h.svc.List(c.Request.Context(), id, p.Limit, p.Offset)
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
	name := strings.TrimSuffix(c.Param("service"), ":scale")
	v, err := h.svc.Get(c.Request.Context(), id, name)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Scale(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	name := strings.TrimSuffix(c.Param("service"), ":scale")
	var in ScaleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Scale(c.Request.Context(), id, name, in)
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
	if err := h.svc.Delete(c.Request.Context(), id, c.Param("service")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
