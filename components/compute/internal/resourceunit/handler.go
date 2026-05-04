package resourceunit

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/server"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Handler exposes /resource-pools/:pool/resource-units routes.
type Handler struct {
	svc   *Service
	pools *resourcepool.Service
}

// NewHandler constructs a Handler. The pool service resolves :pool to FK ID.
func NewHandler(svc *Service, pools *resourcepool.Service) *Handler {
	return &Handler{svc: svc, pools: pools}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/resource-pools/:pool/resource-units")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:unit", h.Get)
	g.PATCH("/:unit", h.Update)
	g.DELETE("/:unit", h.Delete)
}

func (h *Handler) resolvePool(c *gin.Context) (uuid.UUID, error) {
	if h.pools == nil {
		return uuid.Nil, apperrors.New(apperrors.CodeInternal, "pool service unavailable")
	}
	pool, err := h.pools.Get(c.Request.Context(), c.Param("pool"))
	if err != nil {
		return uuid.Nil, err
	}
	return pool.ID, nil
}

func (h *Handler) Create(c *gin.Context) {
	poolID, err := h.resolvePool(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), poolID, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) List(c *gin.Context) {
	poolID, err := h.resolvePool(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items, total, err := h.svc.ListByPool(c.Request.Context(), poolID, p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	poolID, err := h.resolvePool(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Get(c.Request.Context(), poolID, c.Param("unit"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Update(c *gin.Context) {
	poolID, err := h.resolvePool(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	var in UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Update(c.Request.Context(), poolID, c.Param("unit"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	poolID, err := h.resolvePool(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), poolID, c.Param("unit")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
