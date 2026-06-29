package resourcepool

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Handler serves the ResourcePools and ResourceUnits tags. Reads are open to any
// authenticated user; writes are system-admin only (auth.md §3.1).
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts pool/unit routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	p := rg.Group("/resourcepools", h.authn.RequireAuthenticated())
	p.GET("", h.listPools)
	p.POST("", h.authn.RequireSystemAdmin(), h.createPool)
	p.GET("/:pool", h.getPool)
	p.PATCH("/:pool", h.authn.RequireSystemAdmin(), h.updatePool)
	p.DELETE("/:pool", h.authn.RequireSystemAdmin(), h.deletePool)

	p.GET("/:pool/units", h.listUnits)
	p.POST("/:pool/units", h.authn.RequireSystemAdmin(), h.createUnit)
	p.GET("/:pool/units/:unit", h.getUnit)
	p.PATCH("/:pool/units/:unit", h.authn.RequireSystemAdmin(), h.updateUnit)
	p.DELETE("/:pool/units/:unit", h.authn.RequireSystemAdmin(), h.deleteUnit)
}

func (h *Handler) listPools(c *gin.Context) {
	pools, err := h.svc.ListPools(c.Request.Context(), c.Query("q"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.ResourcePoolList{Items: pools, Count: len(pools)})
}

func (h *Handler) getPool(c *gin.Context) {
	p, err := h.svc.GetPool(c.Request.Context(), c.Param("pool"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) createPool(c *gin.Context) {
	var req server.ResourcePoolCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	p, err := h.svc.CreatePool(c.Request.Context(), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) updatePool(c *gin.Context) {
	var req server.ResourcePoolPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	p, err := h.svc.UpdatePool(c.Request.Context(), c.Param("pool"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) deletePool(c *gin.Context) {
	if err := h.svc.DeletePool(c.Request.Context(), c.Param("pool")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listUnits(c *gin.Context) {
	units, err := h.svc.ListUnits(c.Request.Context(), c.Param("pool"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.ResourceUnitList{Items: units, Count: len(units)})
}

func (h *Handler) getUnit(c *gin.Context) {
	u, err := h.svc.GetUnit(c.Request.Context(), c.Param("pool"), c.Param("unit"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *Handler) createUnit(c *gin.Context) {
	var req server.ResourceUnitCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	u, err := h.svc.CreateUnit(c.Request.Context(), c.Param("pool"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, u)
}

func (h *Handler) updateUnit(c *gin.Context) {
	var req server.ResourceUnitPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	u, err := h.svc.UpdateUnit(c.Request.Context(), c.Param("pool"), c.Param("unit"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *Handler) deleteUnit(c *gin.Context) {
	if err := h.svc.DeleteUnit(c.Request.Context(), c.Param("pool"), c.Param("unit")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
