package job

import (
	"net/http"

	"github.com/gin-gonic/gin"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes /namespaces/:namespace/jobs routes. Namespace is the bare
// URL partition key; Compute does no existence / activation check on it.
type Handler struct {
	svc  *Service
	kube *kubeproxy.Client
}

// NewHandler builds a job HTTP handler. kube may be nil in pure-DB tests;
// the pods / log / events sub-routes are skipped when so.
func NewHandler(svc *Service, kube *kubeproxy.Client) *Handler {
	return &Handler{svc: svc, kube: kube}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/jobs")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:job", h.Get)
	g.POST("/:job/cancel", h.Cancel)
	g.DELETE("/:job", h.Delete)
	if h.kube != nil {
		g.GET("/:job/pods", h.ListPods)
		g.GET("/:job/pods/:pod/logs", h.PodLog)
		g.GET("/:job/pods/:pod/events", h.PodEvents)
		g.GET("/:job/events", h.JobEvents)
	}
}

// ListPods lists Pods labeled axisml.io/job-id=<row.id>.
func (h *Handler) ListPods(c *gin.Context) {
	j, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("job"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.kube.PodsByLabel(c, j.Namespace, mljobv1alpha1.LabelJobID, j.ID.String())
}

// PodLog streams a pod's log.
func (h *Handler) PodLog(c *gin.Context) {
	h.kube.PodLog(c, c.Param("namespace"), c.Param("pod"))
}

// PodEvents lists events whose involvedObject is the pod.
func (h *Handler) PodEvents(c *gin.Context) {
	h.kube.EventsByInvolved(c, c.Param("namespace"), "Pod", c.Param("pod"))
}

// JobEvents lists events whose involvedObject is the MLJob CR.
func (h *Handler) JobEvents(c *gin.Context) {
	h.kube.EventsByInvolved(c, c.Param("namespace"), "MLJob", c.Param("job"))
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
	clause, args, err := server.JSONLabelsSQL("labels", c.Query("labelSelector"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), ns, p.Limit, p.Offset, clause, args)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("job"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Cancel(c *gin.Context) {
	v, err := h.svc.Cancel(c.Request.Context(), c.Param("namespace"), c.Param("job"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("job")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
