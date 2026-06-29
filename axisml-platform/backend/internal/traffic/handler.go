package traffic

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/compview"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Handler serves the TrafficPolicy tag.
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the traffic Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts traffic-policy routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/trafficpolicies", h.authn.RequireAuthenticated())
	g.GET("", h.list)

	s := g.Group("", h.authn.RequireActiveTenantRole(auth.RoleUser))
	s.POST("", h.create)
	s.GET("/:name", h.get)
	s.PATCH("/:name", h.update)
	s.DELETE("/:name", h.delete)
	s.POST("/:name/split", h.split)
	s.POST("/:name/promote", h.promote)
	s.POST("/:name/rollback", h.rollback)
	s.GET("/:name/metrics", h.metrics)
	s.GET("/:name/events", h.events)
}

func (h *Handler) create(c *gin.Context) {
	var req server.TrafficPolicyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), auth.ActiveTenant(c), auth.Current(c).Username, req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) list(c *gin.Context) {
	id := auth.Current(c)
	tenant := auth.ActiveTenant(c)
	if tenant == "" {
		server.Fail(c, server.ActiveTenantRequired())
		return
	}
	if !id.HasTenantRole(tenant, auth.RoleUser) {
		server.Fail(c, server.NotFound("tenant not found"))
		return
	}
	items, err := h.svc.List(c.Request.Context(), tenant)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.TrafficPolicyList{Items: items, Count: len(items)})
}

func (h *Handler) get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) update(c *gin.Context) {
	// Display-metadata patch is not yet surfaced by the compute traffic API;
	// echo the current policy.
	h.get(c)
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), auth.ActiveTenant(c), c.Param("name")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) split(c *gin.Context) {
	var req server.TrafficPolicySplitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.Split(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) promote(c *gin.Context) {
	v, err := h.svc.Promote(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) rollback(c *gin.Context) {
	v, err := h.svc.Rollback(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) metrics(c *gin.Context) { server.Fail(c, server.MetricsUnavailable()) }

func (h *Handler) events(c *gin.Context) {
	events, err := h.svc.Events(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Events(events))
}
