package tenant

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Handler serves the Tenants, Quotas and Members tags.
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the tenant Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts tenant / quota / member routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	t := rg.Group("/tenants", h.authn.RequireAuthenticated())
	t.GET("", h.listTenants)
	t.POST("", h.authn.RequireSystemAdmin(), h.createTenant)
	t.GET("/:name", h.authn.RequireTenantRole(auth.RoleUser, "name"), h.getTenant)
	t.PATCH("/:name", h.authn.RequireTenantRole(auth.RoleTenantAdmin, "name"), h.updateTenant)
	t.DELETE("/:name", h.authn.RequireSystemAdmin(), h.deleteTenant)
	t.POST("/:name/suspend", h.authn.RequireSystemAdmin(), h.suspendTenant)
	t.POST("/:name/resume", h.authn.RequireSystemAdmin(), h.resumeTenant)

	t.GET("/:name/quotas", h.authn.RequireTenantRole(auth.RoleUser, "name"), h.listQuotas)
	t.POST("/:name/quotas", h.authn.RequireSystemAdmin(), h.createQuota)
	t.PATCH("/:name/quotas/:pool", h.authn.RequireSystemAdmin(), h.updateQuota)
	t.DELETE("/:name/quotas/:pool", h.authn.RequireSystemAdmin(), h.deleteQuota)

	t.GET("/:name/members", h.authn.RequireTenantRole(auth.RoleUser, "name"), h.listMembers)
	t.POST("/:name/members", h.authn.RequireTenantRole(auth.RoleTenantAdmin, "name"), h.addMember)
	t.PATCH("/:name/members/:userId", h.authn.RequireTenantRole(auth.RoleTenantAdmin, "name"), h.updateMember)
	t.DELETE("/:name/members/:userId", h.authn.RequireTenantRole(auth.RoleTenantAdmin, "name"), h.removeMember)
}

// ---- Tenants ----

func (h *Handler) createTenant(c *gin.Context) {
	var req server.TenantCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	in := CreateInput{
		Identifier:          req.Identifier,
		KubernetesNamespace: req.KubernetesNamespace,
		DisplayName:         req.DisplayName,
		Description:         req.Description,
		InitialAdmin:        req.InitialAdmin,
		Labels:              req.Labels,
		Annotations:         req.Annotations,
		Quotas:              fromContractQuotas(req.Quotas),
		Volumes:             fromContractVolumes(req.Volumes),
	}
	view, err := h.svc.Create(c.Request.Context(), in, auth.Current(c).Username)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *Handler) listTenants(c *gin.Context) {
	id := auth.Current(c)
	var scope []string
	if !id.IsSystemAdmin {
		scope = id.VisibleTenants()
	}
	page := server.ParsePage(c)
	stats := c.Query("stats") == "true"
	items, partial, err := h.svc.List(c.Request.Context(), scope, c.Query("q"), stats, page.Limit, page.Offset())
	if err != nil {
		server.Fail(c, err)
		return
	}
	out := make([]server.Tenant, 0, len(items))
	for _, it := range items {
		out = append(out, *it)
	}
	c.JSON(http.StatusOK, server.TenantList{
		Items:         out,
		Count:         len(out),
		ContinueToken: server.NextContinue(page.Offset(), page.Limit, len(out)),
		Partial:       partial,
	})
}

func (h *Handler) getTenant(c *gin.Context) {
	view, err := h.svc.Get(c.Request.Context(), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) updateTenant(c *gin.Context) {
	var req server.TenantPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	view, err := h.svc.UpdateMeta(c.Request.Context(), c.Param("name"), req.DisplayName, req.Description, req.Labels, req.Annotations, auth.Current(c).Username)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) deleteTenant(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("name")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) suspendTenant(c *gin.Context) { h.setSuspended(c, true) }
func (h *Handler) resumeTenant(c *gin.Context)  { h.setSuspended(c, false) }

func (h *Handler) setSuspended(c *gin.Context, suspend bool) {
	view, err := h.svc.SetSuspended(c.Request.Context(), c.Param("name"), suspend)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

// ---- Quotas ----

func (h *Handler) listQuotas(c *gin.Context) {
	list, err := h.svc.ListQuotas(c.Request.Context(), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) createQuota(c *gin.Context) {
	var req server.QuotaCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	view, err := h.svc.SetQuota(c.Request.Context(), c.Param("name"), makeQuotaSpec(req.Pool, req.Units, req.Quota))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *Handler) updateQuota(c *gin.Context) {
	var req server.QuotaPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	view, err := h.svc.UpdateQuota(c.Request.Context(), c.Param("name"), makeQuotaSpec(c.Param("pool"), req.Units, req.Quota))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) deleteQuota(c *gin.Context) {
	if err := h.svc.DeleteQuota(c.Request.Context(), c.Param("name"), c.Param("pool")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Members ----

func (h *Handler) listMembers(c *gin.Context) {
	members, err := h.svc.ListMembers(c.Request.Context(), c.Param("name"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.MemberList{Items: members, Count: len(members)})
}

func (h *Handler) addMember(c *gin.Context) {
	var req server.MemberCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	m, err := h.svc.AddMember(c.Request.Context(), c.Param("name"), req.Account, string(req.RoleName))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *Handler) updateMember(c *gin.Context) {
	var req server.MemberPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	m, err := h.svc.UpdateMember(c.Request.Context(), c.Param("name"), c.Param("userId"), string(req.RoleName))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *Handler) removeMember(c *gin.Context) {
	if err := h.svc.RemoveMember(c.Request.Context(), c.Param("name"), c.Param("userId")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- request mapping ----

func fromContractQuotas(qs []server.Quota) []QuotaSpec {
	out := make([]QuotaSpec, 0, len(qs))
	for _, q := range qs {
		out = append(out, makeQuotaSpec(q.Pool, q.Units, q.Quota))
	}
	return out
}

// makeQuotaSpec resolves one pool's contract input into the internal QuotaSpec.
// Direct min/max wins when present; otherwise the units business form is used.
// Mode exclusivity (both/neither) is enforced downstream by cluster-manager.
func makeQuotaSpec(pool string, units []server.QuotaUnit, direct *server.QuotaResources) QuotaSpec {
	if direct != nil {
		return QuotaSpec{Pool: pool, Direct: &QuotaResourcesSpec{Min: direct.Min, Max: direct.Max}}
	}
	return QuotaSpec{Pool: pool, Units: fromContractUnits(units)}
}

func fromContractUnits(units []server.QuotaUnit) []QuotaUnitSpec {
	out := make([]QuotaUnitSpec, 0, len(units))
	for _, u := range units {
		out = append(out, QuotaUnitSpec{UnitName: u.UnitName, Quantity: u.Quantity})
	}
	return out
}
