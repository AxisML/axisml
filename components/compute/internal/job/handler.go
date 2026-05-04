package job

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/axisml/axisml/components/compute/internal/server"
	apperrors "github.com/axisml/axisml/components/compute/pkg/errors"
)

// Handler exposes /tenants/:tenant/jobs routes.
type Handler struct {
	svc        *Service
	middleware []gin.HandlerFunc
}

// NewHandler builds a job HTTP handler. middleware is applied to the route
// group so the tenant resolver runs before each handler.
func NewHandler(svc *Service, middleware ...gin.HandlerFunc) *Handler {
	return &Handler{svc: svc, middleware: middleware}
}

// Register implements server.Module.
func (h *Handler) Register(rg *gin.RouterGroup) {
	g := rg.Group("/tenants/:tenant/jobs")
	for _, m := range h.middleware {
		g.Use(m)
	}
	g.Use(populateTenantName)
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:job", h.Get)
	g.POST("/:job/cancel", h.Cancel)
	g.DELETE("/:job", h.Delete)
}

// populateTenantName takes the tenant URL segment (already validated by the
// tenant middleware) and stashes it on the request context so downstream
// services can render the ElasticQuota name without an extra DB lookup.
func populateTenantName(c *gin.Context) {
	name := c.Param("tenant")
	c.Request = c.Request.WithContext(WithTenantName(c.Request.Context(), name))
	c.Next()
}

func tenantID(c *gin.Context) (uuid.UUID, error) {
	v, ok := c.Get("tenantID")
	if !ok {
		return uuid.Nil, apperrors.New(apperrors.CodeInternal, "tenant resolver not configured")
	}
	id, _ := v.(uuid.UUID)
	return id, nil
}

func (h *Handler) Create(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	var in CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		_ = c.Error(err)
		return
	}
	v, err := h.svc.Create(c.Request.Context(), id, in)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) List(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	p, err := server.ParsePagination(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), id, p.Limit, p.Offset)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	name := strings.TrimSuffix(c.Param("job"), ":cancel")
	v, err := h.svc.Get(c.Request.Context(), id, name)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Cancel(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	name := strings.TrimSuffix(c.Param("job"), ":cancel")
	v, err := h.svc.Cancel(c.Request.Context(), id, name)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := tenantID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, c.Param("job")); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
