package dashboard

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
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

// Register mounts dashboard routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/dashboard", h.authn.RequireAuthenticated())
	g.GET("/activity", h.activity)
	g.GET("/cluster-usage", h.clusterUsage)
	g.GET("/cluster-metrics", h.clusterMetrics)
}

func (h *Handler) activity(c *gin.Context) {
	tenant, ok := h.tenant(c, c.Query("activeTenant"))
	if !ok {
		return
	}
	list, err := h.svc.Activity(c.Request.Context(), tenant, parseLimit(c.Query("limit"), 50))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) clusterUsage(c *gin.Context) {
	tenant, ok := h.tenant(c, "")
	if !ok {
		return
	}
	usage, err := h.svc.ClusterUsage(c.Request.Context(), tenant, c.Query("pool"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, usage)
}

func (h *Handler) clusterMetrics(c *gin.Context) {
	tenant, ok := h.tenant(c, "")
	if !ok {
		return
	}
	pool := c.Query("pool")
	if pool == "" {
		// N3 is per (tenant, pool); the frontend renders one series per pool.
		server.Fail(c, apperrors.New(apperrors.ClassValidation, "pool query parameter is required").WithReason("pool-required"))
		return
	}
	series, err := h.svc.ClusterMetrics(c.Request.Context(), tenant, pool, c.Query("metric"), c.Query("range"), server.OptStr(c.Query("step")))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, series)
}

// tenant resolves the active tenant (override first, then context) and checks
// the caller has a role in it.
func (h *Handler) tenant(c *gin.Context, override string) (string, bool) {
	tenant := override
	if tenant == "" {
		tenant = auth.ActiveTenant(c)
	}
	if tenant == "" {
		server.Fail(c, server.ActiveTenantRequired())
		return "", false
	}
	if id := auth.Current(c); id == nil || !id.HasTenantRole(tenant, auth.RoleUser) {
		server.Fail(c, server.NotFound("tenant not found"))
		return "", false
	}
	return tenant, true
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
