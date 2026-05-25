package artifact

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/artifact-hub/internal/auth"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
)

// Handler exposes routes under /api/v1/namespaces/{ns}/artifacts/...
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace")
	g.GET("/artifacts/:kind", h.ListByKind)
	g.POST("/artifacts/:kind/:name", h.Initiate)
	g.GET("/artifacts/:kind/:name", h.List)
	g.GET("/artifacts/:kind/:name/:version", h.Get)
	g.POST("/artifacts/:kind/:name/:version/complete", h.Complete)
	g.GET("/artifacts/:kind/:name/:version/resolve", h.Resolve)
	g.DELETE("/artifacts/:kind/:name/:version", h.Delete)
}

func (h *Handler) Initiate(c *gin.Context) {
	user := auth.User(c.Request.Context())
	var in InitiateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	res, err := h.svc.Initiate(c.Request.Context(), c.Param("namespace"), c.Param("kind"), c.Param("name"), user, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) List(c *gin.Context) {
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	rows, total, err := h.svc.List(c.Request.Context(), c.Param("namespace"), c.Param("kind"), c.Param("name"), c.Query("status"), p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]View, 0, len(rows))
	for i := range rows {
		items = append(items, toView(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// ListByKind handles GET /artifacts/{kind} — every (name, version) under
// the namespace's kind.
func (h *Handler) ListByKind(c *gin.Context) {
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	rows, total, err := h.svc.List(c.Request.Context(), c.Param("namespace"), c.Param("kind"), "", c.Query("status"), p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]View, 0, len(rows))
	for i := range rows {
		items = append(items, toView(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	row, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("kind"), c.Param("name"), c.Param("version"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

func (h *Handler) Complete(c *gin.Context) {
	var in CompleteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.svc.Complete(c.Request.Context(), c.Param("namespace"), c.Param("kind"), c.Param("name"), c.Param("version"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

func (h *Handler) Resolve(c *gin.Context) {
	res, err := h.svc.Resolve(c.Request.Context(), c.Param("namespace"), c.Param("kind"), c.Param("name"), c.Param("version"), c.Query("usage"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.MarkDeleting(c.Request.Context(), c.Param("namespace"), c.Param("kind"), c.Param("name"), c.Param("version")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusAccepted)
}
