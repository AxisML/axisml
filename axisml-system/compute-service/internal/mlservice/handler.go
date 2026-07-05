package mlservice

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/metricsquery"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// Handler exposes /namespaces/:namespace/mlservices routes.
type Handler struct {
	svc     *Module
	runtime extensions.ComputeRuntime
	metrics *metricsquery.Querier
}

func NewHandler(svc *Module, runtime extensions.ComputeRuntime, metrics *metricsquery.Querier) *Handler {
	return &Handler{svc: svc, runtime: runtime, metrics: metrics}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/mlservices")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:mlservice", h.Get)
	g.PATCH("/:mlservice", h.Patch)
	g.POST("/:mlservice/scale", h.Scale)
	g.DELETE("/:mlservice", h.Delete)
	if h.runtime != nil {
		g.GET("/:mlservice/pods", h.ListPods)
		g.GET("/:mlservice/pods/:pod/logs", h.PodLog)
		g.GET("/:mlservice/pods/:pod/events", h.PodEvents)
		g.GET("/:mlservice/events", h.MLServiceEvents)
		g.GET("/:mlservice/metrics", h.Metrics)
	}
}

// Metrics returns a resource metric time series for the service, sampled from
// Prometheus over the pods currently backing it.
func (h *Handler) Metrics(c *gin.Context) {
	if h.metrics == nil || !h.metrics.Enabled() {
		_ = c.Error(apperrors.New(apperrors.CodeUnavailable, "workload metrics are unavailable"))
		return
	}
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	pods, err := h.runtime.ListMLServiceInstances(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	series, err := h.metrics.Series(c.Request.Context(), key.Namespace, metricsquery.PodNames(pods),
		c.Query("metric"), c.Query("range"), c.Query("step"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, series)
}

// keyFor resolves the row (checking the :namespace path param against it) and
// returns the workload key.
func (h *Handler) keyFor(c *gin.Context) (types.NamespacedName, bool) {
	s, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("mlservice"))
	if err != nil {
		_ = c.Error(err)
		return types.NamespacedName{}, false
	}
	return types.NamespacedName{Namespace: s.Namespace, Name: s.Name}, true
}

func (h *Handler) ListPods(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	pods, err := h.runtime.ListMLServiceInstances(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	kubeproxy.WritePods(c, pods)
}

func (h *Handler) Patch(c *gin.Context) {
	var in server.MLServicePatchRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), c.Param("mlservice"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

// PodLog streams an instance's log. The runtime verifies the instance belongs
// to the addressed Service before streaming.
func (h *Handler) PodLog(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	opts, follow := kubeproxy.PodLogQuery(c)
	kubeproxy.StreamLog(c, follow, func() (io.ReadCloser, error) {
		return h.runtime.GetMLServiceInstanceLogs(c.Request.Context(), key, c.Param("pod"), opts)
	})
}

func (h *Handler) PodEvents(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	evs, err := h.runtime.GetMLServiceInstanceEvents(c.Request.Context(), key, c.Param("pod"))
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	kubeproxy.WriteEvents(c, evs)
}

// MLServiceEvents lists events targeting the MLService CR or its underlying
// workload primitives (Deployment / StatefulSet) and exposed HTTPRoute
// per design §4.4.
func (h *Handler) MLServiceEvents(c *gin.Context) {
	key, ok := h.keyFor(c)
	if !ok {
		return
	}
	evs, err := h.runtime.GetMLServiceEvents(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(kubeproxy.MapErr(err))
		return
	}
	kubeproxy.WriteEvents(c, evs)
}

func (h *Handler) Create(c *gin.Context) {
	ns := c.Param("namespace")
	var in server.MLServiceCreateRequest
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
	items, total, err := h.svc.List(c.Request.Context(), ns, c.Query("kind"), p.Limit, p.Offset, clause, args)
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
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("mlservice"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Scale(c *gin.Context) {
	var in server.MLServiceScaleRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Scale(c.Request.Context(), c.Param("namespace"), c.Param("mlservice"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	// 202 Accepted per design yaml: scale is async — generation bumped,
	// reconciler will propagate to the CR.
	c.JSON(http.StatusAccepted, v)
}

func (h *Handler) Delete(c *gin.Context) {
	// A workspace's durable volume is reclaimed by Platform via cluster-manager,
	// not here, so there is no PVC-cascade option on this endpoint.
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("mlservice")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
