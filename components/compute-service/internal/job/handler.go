package job

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Handler exposes /namespaces/:namespace/jobs routes. Namespace is the bare
// URL partition key; Compute does no existence / activation check on it.
type Handler struct {
	svc *Service
}

// NewHandler builds a job HTTP handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/jobs")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:job", h.Get)
	g.POST("/:job/cancel", h.Cancel)
	g.DELETE("/:job", h.Delete)
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
	items, total, err := h.svc.List(c.Request.Context(), ns, p.Limit, p.Offset)
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
