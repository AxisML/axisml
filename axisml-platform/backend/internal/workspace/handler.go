package workspace

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/compview"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Handler serves the Workspaces tag.
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the Workspace Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts workspace routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/workspaces", h.authn.RequireAuthenticated())
	g.GET("", h.list)

	s := g.Group("", h.authn.RequireActiveTenantRole(auth.RoleUser))
	s.POST("", h.create)
	s.GET("/:name", h.get)
	s.PATCH("/:name", h.update)
	s.DELETE("/:name", h.delete)
	s.POST("/:name/start", h.start)
	s.POST("/:name/stop", h.stop)
	s.GET("/:name/events", h.events)
	s.GET("/:name/pods", h.pods)
	s.GET("/:name/pods/:pod/logs", h.podLogs)
	s.GET("/:name/pods/:pod/events", h.podEvents)
}

func (h *Handler) create(c *gin.Context) {
	var req server.WorkspaceCreateRequest
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
	c.JSON(http.StatusOK, server.WorkspaceList{Items: items, Count: len(items)})
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
	var req server.WorkspacePatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.UpdateMeta(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), req.DisplayName, req.Description)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) delete(c *gin.Context) {
	var req server.WorkspaceDeleteRequest
	var deletePVC *bool
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err == nil {
			deletePVC = &req.DeletePVC
		}
	}
	if err := h.svc.Delete(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), deletePVC); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) start(c *gin.Context) {
	v, err := h.svc.Start(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) stop(c *gin.Context) {
	v, err := h.svc.Stop(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) events(c *gin.Context) {
	events, err := h.svc.Events(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Events(events))
}

func (h *Handler) pods(c *gin.Context) {
	pods, err := h.svc.Pods(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Pods(pods))
}

func (h *Handler) podEvents(c *gin.Context) {
	events, err := h.svc.PodEvents(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), c.Param("pod"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Events(events))
}

func (h *Handler) podLogs(c *gin.Context) {
	resp, err := h.svc.PodLogs(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), c.Param("pod"), compview.LogOptions(c))
	if err != nil {
		server.Fail(c, err)
		return
	}
	compview.StreamLogs(c, resp)
}
