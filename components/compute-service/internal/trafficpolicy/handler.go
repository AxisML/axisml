package trafficpolicy

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes /namespaces/:namespace/traffic-policies routes.
type Handler struct {
	svc *Module
}

func NewHandler(svc *Module) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/traffic-policies")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:policy", h.Get)
	g.PATCH("/:policy", h.Patch)
	g.DELETE("/:policy", h.Delete)
	g.POST("/:policy/split", h.Split)
	g.POST("/:policy/promote", h.Promote)
	g.POST("/:policy/rollback", h.Rollback)
}

func (h *Handler) Create(c *gin.Context) {
	var in server.TrafficPolicyCreateRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), c.Param("namespace"), in)
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
	items, total, err := h.svc.List(c.Request.Context(), c.Param("namespace"), p.Limit, p.Offset, clause, args)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"count":         len(items),
		"total":         total,
		"continueToken": server.EncodeContinue(p.Offset, len(items), total),
	})
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("policy"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Patch(c *gin.Context) {
	var in server.TrafficPolicyPatchRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), c.Param("policy"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("policy")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Split(c *gin.Context) {
	var in server.TrafficPolicySplitRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Split(c.Request.Context(), c.Param("namespace"), c.Param("policy"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	// 202 Accepted: weight change is async — generation bumped, reconciler
	// propagates to the CR.
	c.JSON(http.StatusAccepted, v)
}

func (h *Handler) Promote(c *gin.Context) {
	v, err := h.svc.Promote(c.Request.Context(), c.Param("namespace"), c.Param("policy"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusAccepted, v)
}

func (h *Handler) Rollback(c *gin.Context) {
	v, err := h.svc.Rollback(c.Request.Context(), c.Param("namespace"), c.Param("policy"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusAccepted, v)
}
