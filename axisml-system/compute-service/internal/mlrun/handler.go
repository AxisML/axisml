package mlrun

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	"github.com/axisml/axisml/components/compute-service/pkg/extensions"
)

// Handler exposes /namespaces/:namespace/mlruns routes. Namespace is the bare
// URL partition key; Compute does no existence / activation check on it.
type Handler struct {
	svc     *Service
	runtime extensions.ComputeRuntime
}

// NewHandler builds a job HTTP handler. runtime may be nil in pure-DB tests;
// the pods / log / events sub-routes are skipped when so.
func NewHandler(svc *Service, runtime extensions.ComputeRuntime) *Handler {
	return &Handler{svc: svc, runtime: runtime}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/mlruns")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:mlrun", h.Get)
	g.PATCH("/:mlrun", h.Patch)
	g.POST("/:mlrun/cancel", h.Cancel)
	g.DELETE("/:mlrun", h.Delete)
	if h.runtime != nil {
		g.GET("/:mlrun/pods", h.ListPods)
		g.GET("/:mlrun/pods/:pod/logs", h.PodLog)
		g.GET("/:mlrun/pods/:pod/events", h.PodEvents)
		g.GET("/:mlrun/events", h.MLRunEvents)
	}
}

// keyFor resolves the row (so the :namespace path param is checked against it)
// and returns the workload key.
func (h *Handler) keyFor(c *gin.Context) (types.NamespacedName, bool) {
	j, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("mlrun"))
	if err != nil {
		_ = c.Error(err)
		return types.NamespacedName{}, false
	}
	return types.NamespacedName{Namespace: j.Namespace, Name: j.Name}, true
}

// ListPods lists the Run's instances.
func (h *Handler) ListPods(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	pods, err := h.runtime.ListMLRunInstances(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	kubeproxy.WritePods(c, pods)
}

// PodLog streams an instance's log. The runtime verifies the instance belongs
// to the addressed Run before streaming.
func (h *Handler) PodLog(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	opts, follow := kubeproxy.PodLogQuery(c)
	kubeproxy.StreamLog(c, follow, func() (io.ReadCloser, error) {
		return h.runtime.GetMLRunInstanceLogs(c.Request.Context(), key, c.Param("pod"), opts)
	})
}

// PodEvents lists events regarding the named instance.
func (h *Handler) PodEvents(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	evs, err := h.runtime.GetMLRunInstanceEvents(c.Request.Context(), key, c.Param("pod"))
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	kubeproxy.WriteEvents(c, evs)
}

// MLRunEvents lists events targeting the MLRun CR or its peer scheduling
// primitive (PodGroup) per design §4.3.
func (h *Handler) MLRunEvents(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	evs, err := h.runtime.GetMLRunEvents(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	kubeproxy.WriteEvents(c, evs)
}

func (h *Handler) Create(c *gin.Context) {
	ns := c.Param("namespace")
	var in server.MLRunCreateRequest
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
	var in server.MLRunPatchRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), c.Param("mlrun"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("mlrun"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Cancel(c *gin.Context) {
	v, err := h.svc.Cancel(c.Request.Context(), c.Param("namespace"), c.Param("mlrun"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	// 202 Accepted per design yaml: cancel is async — the row is now in
	// Canceling and the reconciler will patch suspend=true on the CR.
	c.JSON(http.StatusAccepted, v)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("mlrun")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
