package service

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes /namespaces/:namespace/services routes.
type Handler struct {
	svc *Module
}

func NewHandler(svc *Module) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/services")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:service", h.Get)
	g.POST("/:service/scale", h.Scale)
	g.DELETE("/:service", h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	ns := c.Param("namespace")
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), ns, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) List(c *gin.Context) {
	ns := c.Param("namespace")
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), ns, p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("service"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Scale(c *gin.Context) {
	var in ScaleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Scale(c.Request.Context(), c.Param("namespace"), c.Param("service"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("service")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
