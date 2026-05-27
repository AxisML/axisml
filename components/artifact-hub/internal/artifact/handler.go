package artifact

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/artifact-hub/internal/auth"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
)

// Handler exposes routes under /api/v1/namespaces/{ns}/{kindPlural}/...
// where kindPlural ∈ {models, datasets, images}. The plural URL token is
// the user-facing kind, mapped 1:1 to the singular Kind enum.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// kindPluralToSingular maps the URL token to the persistence enum.
// Centralising it here keeps the routes table small.
var kindPluralToSingular = map[string]string{
	"models":   "model",
	"datasets": "dataset",
	"images":   "image",
}

// Register implements server.Module. Three identical sub-trees per kind so
// each operationId in the OpenAPI maps to a single concrete route — no
// `:kind` path parameter.
func (h *Handler) Register(rg *gin.RouterGroup) {
	for plural := range kindPluralToSingular {
		g := rg.Group("/namespaces/:namespace/" + plural)
		g.POST("/:name", h.bindKind(plural, h.Initiate))
		g.GET("/:name", h.bindKind(plural, h.List))
		g.GET("/:name/:version", h.bindKind(plural, h.Get))
		g.PATCH("/:name/:version", h.bindKind(plural, h.Patch))
		g.POST("/:name/:version/complete", h.bindKind(plural, h.Complete))
		g.GET("/:name/:version/resolve", h.bindKind(plural, h.Resolve))
		g.DELETE("/:name/:version", h.bindKind(plural, h.Delete))
	}
}

// bindKind injects the singular `kind` into the gin context so the per-route
// handler methods can read it via c.GetString("kind").
func (h *Handler) bindKind(plural string, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("kind", kindPluralToSingular[plural])
		next(c)
	}
}

func kindOf(c *gin.Context) string { return c.GetString("kind") }

func (h *Handler) Initiate(c *gin.Context) {
	user := auth.User(c.Request.Context())
	var in InitiateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	res, err := h.svc.Initiate(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), user, in)
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
	rows, total, err := h.svc.List(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), c.Query("status"), p.Limit, p.Offset)
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
	row, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), c.Param("version"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

// Patch handles PATCH /{kindPlural}/{name}/{version} — only displayName,
// description, labels, annotations are mutable (design §6).
func (h *Handler) Patch(c *gin.Context) {
	var in PatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), c.Param("version"), in)
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
	row, err := h.svc.Complete(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), c.Param("version"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

func (h *Handler) Resolve(c *gin.Context) {
	res, err := h.svc.Resolve(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), c.Param("version"), c.Query("usage"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.MarkDeleting(c.Request.Context(), c.Param("namespace"), kindOf(c), c.Param("name"), c.Param("version")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusAccepted)
}
