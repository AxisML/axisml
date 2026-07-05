package dashboard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Handler serves the Dashboard tag.
type Handler struct {
	svc   *Service
	authn *auth.Authenticator
}

// NewHandler constructs the dashboard Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	return &Handler{svc: svc, authn: authn}
}

// Register mounts dashboard routes. cluster-usage / cluster-metrics are wired
// separately once the System-layer per-pool endpoints land.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/dashboard", h.authn.RequireAuthenticated())
	g.GET("/activity", h.activity)
}

func (h *Handler) activity(c *gin.Context) {
	tenant := c.Query("activeTenant")
	if tenant == "" {
		tenant = auth.ActiveTenant(c)
	}
	if tenant == "" {
		server.Fail(c, server.ActiveTenantRequired())
		return
	}
	if id := auth.Current(c); id == nil || !id.HasTenantRole(tenant, auth.RoleUser) {
		server.Fail(c, server.NotFound("tenant not found"))
		return
	}
	list, err := h.svc.Activity(c.Request.Context(), tenant, parseLimit(c.Query("limit"), 50))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func parseLimit(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	if n > 200 {
		return 200
	}
	return n
}
