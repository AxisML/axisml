package artifactdef

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/server"
)

// Handler serves the Models or Images tag for one kind (tuple-addressed).
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
	seg   string // URL segment: "models" | "images"
}

// NewHandler constructs a Handler for the given kind-plural segment.
func NewHandler(svc *Service, authn *auth.Authenticator, seg string) *Handler {
	return &Handler{svc: svc, authn: authn, seg: seg}
}

// Register mounts definition + version routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/"+h.seg, h.authn.RequireAuthenticated())
	g.GET("", h.listDefs) // header-scoped list with admin fan-out

	t := g.Group("/:tenant", h.authn.RequireTenantRole(auth.RoleUser, "tenant"))
	t.POST("/:name", h.createDef)
	t.GET("/:name", h.getDef)
	t.PATCH("/:name", h.updateDef)
	t.DELETE("/:name", h.deleteDef)
	t.GET("/:name/versions", h.listVersions)
	t.POST("/:name/versions", h.initiate)
	t.GET("/:name/versions/:version", h.getVersion)
	t.PATCH("/:name/versions/:version", h.updateVersion)
	t.DELETE("/:name/versions/:version", h.deleteVersion)
	t.POST("/:name/versions/:version/complete", h.complete)
	t.GET("/:name/versions/:version/resolve", h.resolve)
}

func (h *Handler) listDefs(c *gin.Context) {
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
	items, err := h.svc.ListDefs(c.Request.Context(), scope, c.Query("q"), page.Limit, page.Offset())
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.ArtifactDefinitionList{
		Items:         items,
		Count:         len(items),
		ContinueToken: server.NextContinue(page.Offset(), page.Limit, len(items)),
	})
}

func (h *Handler) createDef(c *gin.Context) {
	var req server.ArtifactDefinitionCreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.CreateDef(c.Request.Context(), c.Param("tenant"), c.Param("name"), auth.Current(c).Username, req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) getDef(c *gin.Context) {
	v, err := h.svc.GetDef(c.Request.Context(), c.Param("tenant"), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) updateDef(c *gin.Context) {
	var req server.ArtifactDefinitionPatchInput
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.UpdateDef(c.Request.Context(), c.Param("tenant"), c.Param("name"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) deleteDef(c *gin.Context) {
	if err := h.svc.DeleteDef(c.Request.Context(), c.Param("tenant"), c.Param("name")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listVersions(c *gin.Context) {
	items, err := h.svc.ListVersions(c.Request.Context(), c.Param("tenant"), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.ModelList{Items: items, Count: len(items)})
}

type initiateBody struct {
	Version     string         `json:"version"`
	DisplayName string         `json:"displayName"`
	Description string         `json:"description"`
	Source      string         `json:"source"`
	Spec        map[string]any `json:"spec"`
}

func (h *Handler) initiate(c *gin.Context) {
	var req initiateBody
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	res, err := h.svc.InitiateVersion(c.Request.Context(), c.Param("tenant"), c.Param("name"),
		req.Version, req.DisplayName, req.Description, req.Source, req.Spec)
	if err != nil {
		server.Fail(c, err)
		return
	}
	resp := server.ModelInitiateResponse{ID: res.View.ID}
	if v, ok := res.Upload["uri"].(string); ok {
		resp.URI = v
	}
	if v, ok := res.Upload["storageKind"].(string); ok {
		resp.StorageKind = server.StorageKind(v)
	}
	if v, ok := res.Upload["credentials"].(map[string]any); ok {
		resp.UploadCredentials = v
	}
	c.JSON(http.StatusAccepted, resp)
}

func (h *Handler) complete(c *gin.Context) {
	var req server.ModelCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.CompleteVersion(c.Request.Context(), c.Param("tenant"), c.Param("name"), c.Param("version"), req.Digest)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) getVersion(c *gin.Context) {
	v, err := h.svc.GetVersion(c.Request.Context(), c.Param("tenant"), c.Param("name"), c.Param("version"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) updateVersion(c *gin.Context) {
	var req server.ArtifactUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.UpdateVersion(c.Request.Context(), c.Param("tenant"), c.Param("name"), c.Param("version"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) deleteVersion(c *gin.Context) {
	if err := h.svc.DeleteVersion(c.Request.Context(), c.Param("tenant"), c.Param("name"), c.Param("version")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) resolve(c *gin.Context) {
	v, err := h.svc.Resolve(c.Request.Context(), c.Param("tenant"), c.Param("name"), c.Param("version"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}
