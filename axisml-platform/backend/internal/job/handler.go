package job

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/compview"
	"github.com/axisml/axisml/components/platform/internal/server"
)

// Handler serves the Jobs tag (Job definitions + their Runs).
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the Job Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts job + run routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/jobs", h.authn.RequireAuthenticated())
	g.GET("", h.list) // list does its own scope resolution (admin fanout)

	scoped := g.Group("", h.authn.RequireActiveTenantRole(auth.RoleUser))
	scoped.POST("", h.create)
	scoped.GET("/:name", h.get)
	scoped.PATCH("/:name", h.update)
	scoped.DELETE("/:name", h.delete)

	scoped.POST("/:name/runs", h.triggerRun)
	scoped.GET("/:name/runs", h.listRuns)
	scoped.GET("/:name/runs/:run", h.getRun)
	scoped.DELETE("/:name/runs/:run", h.deleteRun)
	scoped.POST("/:name/runs/:run/cancel", h.cancelRun)
	scoped.GET("/:name/runs/:run/metrics", h.runMetrics)
	scoped.GET("/:name/runs/:run/pods", h.runPods)
	scoped.GET("/:name/runs/:run/events", h.runEvents)
	scoped.GET("/:name/runs/:run/pods/:pod/logs", h.runPodLogs)
	scoped.GET("/:name/runs/:run/pods/:pod/events", h.runPodEvents)
}

func (h *Handler) create(c *gin.Context) {
	var req server.JobCreateRequest
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
	page := server.ParsePage(c)
	var scope []string
	switch {
	case tenant != "":
		if !id.HasTenantRole(tenant, auth.RoleUser) {
			server.Fail(c, server.NotFound("tenant not found"))
			return
		}
		scope = []string{tenant}
	case id.IsSystemAdmin:
		scope = nil // cross-tenant fan-out (all)
	default:
		server.Fail(c, server.ActiveTenantRequired())
		return
	}
	owner := ""
	if id.IsSystemAdmin {
		owner = c.Query("owner")
	}
	items, err := h.svc.List(c.Request.Context(), scope, owner, c.Query("q"), page.Limit, page.Offset())
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.JobList{
		Items:         items,
		Count:         len(items),
		ContinueToken: server.NextContinue(page.Offset(), page.Limit, len(items)),
	})
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
	var req server.JobPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.Update(c.Request.Context(), auth.Current(c), auth.ActiveTenant(c), c.Param("name"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), auth.Current(c), auth.ActiveTenant(c), c.Param("name")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) triggerRun(c *gin.Context) {
	var req server.RunTriggerRequest
	hasBody := c.Request.ContentLength != 0
	if hasBody {
		if err := c.ShouldBindJSON(&req); err != nil {
			server.Fail(c, err)
			return
		}
	}
	var ov *server.RunTriggerRequest
	if hasBody {
		ov = &req
	}
	v, err := h.svc.TriggerRun(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), req.DisplayName, ov)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) listRuns(c *gin.Context) {
	items, err := h.svc.ListRuns(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), c.Query("phase"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.RunList{Items: items, Count: len(items)})
}

func (h *Handler) getRun(c *gin.Context) {
	v, err := h.svc.GetRun(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), c.Param("run"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) deleteRun(c *gin.Context) {
	if err := h.svc.DeleteRun(c.Request.Context(), auth.ActiveTenant(c), c.Param("run")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) cancelRun(c *gin.Context) {
	v, err := h.svc.CancelRun(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), c.Param("run"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) runMetrics(c *gin.Context) {
	server.Fail(c, server.MetricsUnavailable())
}

func (h *Handler) runPods(c *gin.Context) {
	pods, err := h.svc.RunPods(c.Request.Context(), auth.ActiveTenant(c), c.Param("run"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Pods(pods))
}

func (h *Handler) runEvents(c *gin.Context) {
	events, err := h.svc.RunEvents(c.Request.Context(), auth.ActiveTenant(c), c.Param("run"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Events(events))
}

func (h *Handler) runPodEvents(c *gin.Context) {
	events, err := h.svc.RunPodEvents(c.Request.Context(), auth.ActiveTenant(c), c.Param("run"), c.Param("pod"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, compview.Events(events))
}

func (h *Handler) runPodLogs(c *gin.Context) {
	resp, err := h.svc.RunPodLogs(c.Request.Context(), auth.ActiveTenant(c), c.Param("run"), c.Param("pod"), compview.LogOptions(c))
	if err != nil {
		server.Fail(c, err)
		return
	}
	compview.StreamLogs(c, resp)
}
