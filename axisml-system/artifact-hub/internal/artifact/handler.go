package artifact

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/auth"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/server"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// Handler exposes routes under /api/v1/namespaces/{ns}/artifacts/... The API
// no longer distinguishes artifact types by URL: a single /artifacts resource
// serves every kind, with kind carried in the Initiate body and recovered from
// the persisted row on every other operation.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register implements server.Module. One sub-tree serves all kinds; artifacts
// are addressed by (namespace, name, version).
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/namespaces/:namespace/artifacts")
	g.GET("", h.ListAll)
	g.POST("/:name", h.Initiate)
	g.GET("/:name", h.List)
	g.GET("/:name/:version", h.Get)
	g.PATCH("/:name/:version", h.Patch)
	g.POST("/:name/:version/complete", h.Complete)
	g.GET("/:name/:version/resolve", h.Resolve)
	g.DELETE("/:name/:version", h.Delete)
}

// ListAll handles GET /api/v1/namespaces/{ns}/artifacts — returns every
// (name, version) in the namespace, optionally narrowed by ?kind=. Exposed for
// browsing all artifacts without specifying a name.
func (h *Handler) ListAll(c *gin.Context) {
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
	// name="" → match every name; kind from ?kind= ("" → every kind).
	rows, total, err := h.svc.List(c.Request.Context(),
		c.Param("namespace"), c.Query("kind"), "", c.Query("status"),
		p.Limit, p.Offset, clause, args)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]server.Artifact, 0, len(rows))
	for i := range rows {
		items = append(items, toView(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"count":         len(items),
		"total":         total,
		"continueToken": server.EncodeContinue(p.Offset, len(items), total),
	})
}

func (h *Handler) Initiate(c *gin.Context) {
	user := auth.User(c.Request.Context())
	var in server.ArtifactInitiateRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	res, err := h.svc.Initiate(c.Request.Context(), c.Param("namespace"), c.Param("name"), user, in)
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
	clause, args, err := server.JSONLabelsSQL("labels", c.Query("labelSelector"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	// A name is unique across kinds, so listing its versions needs no kind.
	rows, total, err := h.svc.List(c.Request.Context(), c.Param("namespace"), "", c.Param("name"), c.Query("status"), p.Limit, p.Offset, clause, args)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]server.Artifact, 0, len(rows))
	for i := range rows {
		items = append(items, toView(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"count":         len(items),
		"total":         total,
		"continueToken": server.EncodeContinue(p.Offset, len(items), total),
	})
}

func (h *Handler) Get(c *gin.Context) {
	row, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("name"), c.Param("version"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

// patchAllowedFields is the closed set of mutable keys per design §6.
// Anything else in the PATCH body is rejected with 400 ImmutableField.
// Keys are camelCase to match the JSON contract.
var patchAllowedFields = map[string]struct{}{
	"displayName": {},
	"description": {},
	"labels":      {},
	"annotations": {},
}

// Patch handles PATCH /artifacts/{name}/{version} — only displayName,
// description, labels, annotations are mutable (design §6). Submitting any
// other key returns 400 ImmutableField so callers can't silently no-op
// (e.g. attempting to change visibility / digest).
func (h *Handler) Patch(c *gin.Context) {
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		_ = c.Error(err)
		return
	}
	for k := range raw {
		if _, ok := patchAllowedFields[k]; !ok {
			_ = c.Error(apperrors.Newf(apperrors.CodeValidation,
				"field %q is immutable; only displayName / description / labels / annotations may be patched", k))
			return
		}
	}
	var in server.ArtifactPatchRequest
	if len(raw) > 0 {
		b, err := json.Marshal(raw)
		if err != nil {
			_ = c.Error(err)
			return
		}
		if err := json.Unmarshal(b, &in); err != nil {
			_ = c.Error(err)
			return
		}
	}
	row, err := h.svc.Patch(c.Request.Context(), c.Param("namespace"), c.Param("name"), c.Param("version"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

func (h *Handler) Complete(c *gin.Context) {
	var in server.ArtifactCompleteRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	row, err := h.svc.Complete(c.Request.Context(), c.Param("namespace"), c.Param("name"), c.Param("version"), in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}

func (h *Handler) Resolve(c *gin.Context) {
	res, err := h.svc.Resolve(c.Request.Context(), c.Param("namespace"), c.Param("name"), c.Param("version"), c.Query("usage"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.svc.MarkDeleting(c.Request.Context(), c.Param("namespace"), c.Param("name"), c.Param("version")); err != nil {
		_ = c.Error(err)
		return
	}
	// Per design yaml: DELETE returns the artifact (now status=Deleting)
	// so the caller has the tombstone row + observable state in one trip.
	row, err := h.svc.Get(c.Request.Context(), c.Param("namespace"), c.Param("name"), c.Param("version"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toView(row))
}
