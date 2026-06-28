package datavolume

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/server"
)

// Handler serves the DataVolumes tag. Every operation is system-admin only and
// scoped to the active tenant (carried by the axisml.tenant cookie / header).
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts the data-volume routes. Reads are open to any member of the
// active tenant (so users can pick a volume to mount when launching a workspace /
// job / experiment); writes (create / expand / delete) stay system-admin only.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/datavolumes", h.authn.RequireAuthenticated())
	g.GET("", h.authn.RequireActiveTenantRole(auth.RoleUser), h.list)
	g.GET("/:name", h.authn.RequireActiveTenantRole(auth.RoleUser), h.get)
	g.POST("", h.authn.RequireSystemAdmin(), h.create)
	g.PATCH("/:name", h.authn.RequireSystemAdmin(), h.update)
	g.DELETE("/:name", h.authn.RequireSystemAdmin(), h.delete)

	// StorageClasses are cluster-scoped (not tenant-partitioned); the create form
	// uses them, so they're system-admin only like the volume writes.
	rg.GET("/storageclasses", h.authn.RequireAuthenticated(), h.authn.RequireSystemAdmin(), h.listStorageClasses)
}

func (h *Handler) listStorageClasses(c *gin.Context) {
	items, err := h.svc.ListStorageClasses(c.Request.Context())
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.StorageClassList{Items: items, Count: len(items)})
}

// tenant returns the active tenant scope or fails the request with 400.
func (h *Handler) tenant(c *gin.Context) (string, bool) {
	t := auth.ActiveTenant(c)
	if t == "" {
		server.Fail(c, server.ActiveTenantRequired())
		return "", false
	}
	return t, true
}

func (h *Handler) list(c *gin.Context) {
	tenant, ok := h.tenant(c)
	if !ok {
		return
	}
	items, err := h.svc.List(c.Request.Context(), tenant)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.DataVolumeList{Items: items, Count: len(items)})
}

func (h *Handler) get(c *gin.Context) {
	tenant, ok := h.tenant(c)
	if !ok {
		return
	}
	v, err := h.svc.Get(c.Request.Context(), tenant, c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) create(c *gin.Context) {
	tenant, ok := h.tenant(c)
	if !ok {
		return
	}
	var req server.DataVolumeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), tenant, req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) update(c *gin.Context) {
	tenant, ok := h.tenant(c)
	if !ok {
		return
	}
	var req server.DataVolumePatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	v, err := h.svc.Update(c.Request.Context(), tenant, c.Param("name"), req)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) delete(c *gin.Context) {
	tenant, ok := h.tenant(c)
	if !ok {
		return
	}
	force, _ := strconv.ParseBool(c.Query("force"))
	if err := h.svc.Delete(c.Request.Context(), tenant, c.Param("name"), force); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
