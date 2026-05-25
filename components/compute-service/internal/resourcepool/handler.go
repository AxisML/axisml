package resourcepool

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes the /resource-pools routes.
type Handler struct{ svc *Service }

// NewHandler constructs a Handler bound to the supplied Service.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/resource-pools")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:pool", h.Get)
	g.PATCH("/:pool", h.Update)
	g.DELETE("/:pool", h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), in)
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
	items, total, err := h.svc.List(c.Request.Context(), p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("pool"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Update(c *gin.Context) {
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Update(c.Request.Context(), c.Param("pool"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("pool")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
