package service

import (
	"net/http"

	"github.com/gin-gonic/gin"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes /namespaces/:namespace/services routes.
type Handler struct {
	svc  *Module
	kube *kubeproxy.Client
}

func NewHandler(svc *Module, kube *kubeproxy.Client) *Handler {
	return &Handler{svc: svc, kube: kube}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/services")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:service", h.Get)
	g.POST("/:service/scale", h.Scale)
	g.DELETE("/:service", h.Delete)
	if h.kube != nil {
		g.GET("/:service/pods", h.ListPods)
		g.GET("/:service/pods/:pod/logs", h.PodLog)
		g.GET("/:service/pods/:pod/events", h.PodEvents)
		g.GET("/:service/events", h.ServiceEvents)
	}
}

func (h *Handler) ListPods(c *gin.Context) {
	s, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("service"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	h.kube.PodsByLabel(c, s.Namespace, mlservicev1alpha1.LabelServiceID, s.ID.String())
}

func (h *Handler) PodLog(c *gin.Context) {
	h.kube.PodLog(c, c.Param("namespace"), c.Param("pod"))
}

func (h *Handler) PodEvents(c *gin.Context) {
	h.kube.EventsByInvolved(c, c.Param("namespace"),
		kubeproxy.EventTarget{Kind: "Pod", Name: c.Param("pod")})
}

// ServiceEvents lists events targeting the MLService CR or its underlying
// workload primitives (Deployment / StatefulSet) and exposed HTTPRoute
// per design §4.4.
func (h *Handler) ServiceEvents(c *gin.Context) {
	svcName := c.Param("service")
	h.kube.EventsByInvolved(c, c.Param("namespace"),
		kubeproxy.EventTarget{Kind: "MLService", Name: svcName},
		kubeproxy.EventTarget{Kind: "Deployment", Name: svcName},
		kubeproxy.EventTarget{Kind: "StatefulSet", Name: svcName},
		kubeproxy.EventTarget{Kind: "HTTPRoute", Name: svcName},
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
	items, total, err := h.svc.List(c.Request.Context(), ns, c.Query("kind"), p.Limit, p.Offset, clause, args)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"total":         total,
		"continueToken": server.EncodeContinue(p.Offset, len(items), total),
	})
}

func (h *Handler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("service"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Scale(c *gin.Context) {
	var in ScaleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Scale(c.Request.Context(), c.Param("namespace"), c.Param("service"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	// 202 Accepted per design yaml: scale is async — generation bumped,
	// reconciler will propagate to the CR.
	c.JSON(http.StatusAccepted, v)
}

func (h *Handler) Delete(c *gin.Context) {
	// ?deletePvc=false opts out of cascading PVC deletion for workspaces;
	// design §4.4 defaults to true.
	deletePVC := true
	if v := c.Query("deletePvc"); v == "false" {
		deletePVC = false
	}
	if err := h.svc.Delete(c.Request.Context(), c.Param("namespace"), c.Param("service"), deletePVC); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
