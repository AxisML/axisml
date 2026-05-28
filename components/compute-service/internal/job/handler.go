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
	g.PATCH("/:job", h.Patch)
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

// PodLog streams a pod's log. The pod must carry
// axisml.io/job-id=<row.id>; otherwise the pod is reachable only via the
// kube-apiserver itself and not through this REST surface.
func (h *Handler) PodLog(c *gin.Context) {
	j, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("job"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	if err := h.kube.VerifyPodHasLabel(c.Request.Context(), j.Namespace, c.Param("pod"),
		mljobv1alpha1.LabelJobID, j.ID.String()); err != nil {
		_ = c.Error(err)
		return
	}
	h.kube.PodLog(c, j.Namespace, c.Param("pod"))
}

// PodEvents lists events whose involvedObject is the pod. Same scoping
// as PodLog: the pod must be tagged with the job's id.
func (h *Handler) PodEvents(c *gin.Context) {
	j, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("job"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	if err := h.kube.VerifyPodHasLabel(c.Request.Context(), j.Namespace, c.Param("pod"),
		mljobv1alpha1.LabelJobID, j.ID.String()); err != nil {
		_ = c.Error(err)
		return
	}
	h.kube.EventsByInvolved(c, j.Namespace,
		kubeproxy.EventTarget{Kind: "Pod", Name: c.Param("pod")})
}

// JobEvents lists events targeting the MLJob CR or its peer scheduling
// primitive (PodGroup) per design §4.3. We resolve the row first so the
// :namespace path parameter is checked against the row, not blindly
// forwarded.
func (h *Handler) JobEvents(c *gin.Context) {
	j, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("job"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.kube.EventsByInvolved(c, j.Namespace,
		kubeproxy.EventTarget{Kind: "MLJob", Name: j.Name},
		kubeproxy.EventTarget{Kind: "PodGroup", Name: j.Name},
	)
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
	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"count":         len(items),
		"total":         total,
		"continueToken": server.EncodeContinue(p.Offset, len(items), total),
	})
}

func (h *Handler) Patch(c *gin.Context) {
	var in PatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), c.Param("job"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
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
	// 202 Accepted per design yaml: cancel is async — the row is now in
	// Canceling and the reconciler will patch suspend=true on the CR.
	c.JSON(http.StatusAccepted, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("job")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
