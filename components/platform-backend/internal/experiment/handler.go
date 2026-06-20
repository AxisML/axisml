package experiment

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/compview"
	"github.com/axisml/axisml/components/platform/internal/server"
)

// Handler serves the Experiments tag (definitions + Runs + TensorBoard).
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the Experiment Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts experiment routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/experiments", h.authn.RequireAuthenticated())
	g.GET("", h.list)

	s := g.Group("", h.authn.RequireActiveTenantRole(auth.RoleUser))
	s.POST("", h.create)
	s.GET("/:name", h.get)
	s.PATCH("/:name", h.update)
	s.DELETE("/:name", h.delete)

	s.POST("/:name/runs", h.triggerRun)
	s.GET("/:name/runs", h.listRuns)
	s.GET("/:name/runs/:run", h.getRun)
	s.DELETE("/:name/runs/:run", h.deleteRun)
	s.POST("/:name/runs/:run/cancel", h.cancelRun)
	s.GET("/:name/runs/:run/metrics", h.runMetrics)
	s.GET("/:name/runs/:run/pods", h.runPods)
	s.GET("/:name/runs/:run/events", h.runEvents)
	s.GET("/:name/runs/:run/pods/:pod/logs", h.runPodLogs)
	s.GET("/:name/runs/:run/pods/:pod/events", h.runPodEvents)

	s.GET("/:name/tensorboard", h.getTB)
	s.POST("/:name/tensorboard", h.startTB)
	s.DELETE("/:name/tensorboard", h.stopTB)
}

func (h *Handler) create(c *gin.Context) {
	var req server.ExperimentCreateRequest
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
		scope = nil
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
	c.JSON(http.StatusOK, server.ExperimentList{
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
	var req server.ExperimentPatchRequest
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

func (h *Handler) runMetrics(c *gin.Context) { server.Fail(c, server.MetricsUnavailable()) }

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

func (h *Handler) getTB(c *gin.Context) {
	v, err := h.svc.GetTensorBoard(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) startTB(c *gin.Context) {
	var req server.TensorBoardRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			server.Fail(c, err)
			return
		}
	}
	v, err := h.svc.StartTensorBoard(c.Request.Context(), auth.ActiveTenant(c), c.Param("name"), req.Runs)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) stopTB(c *gin.Context) {
	if err := h.svc.StopTensorBoard(c.Request.Context(), auth.ActiveTenant(c), c.Param("name")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
