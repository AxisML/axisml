package identity

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
)

// Handler serves the Auth and Users tags.
type Handler struct {
	svc        *Service
	authn      *auth.Authenticator
	loginLimit *auth.RateLimiter
}

// NewHandler constructs the identity Handler.
func NewHandler(svc *Service, authn *auth.Authenticator) *Handler {
	// Throttle /auth/login per client IP to blunt online password brute-force:
	// burst of 10, refilling 0.2/s (~1 attempt every 5s sustained).
	return &Handler{svc: svc, authn: authn, loginLimit: auth.NewRateLimiter(10, 0.2)}
}

// Register mounts auth + users routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	a := rg.Group("/auth")
	a.POST("/login", h.loginLimit.Middleware(auth.ClientIPKey), h.login)
	authed := a.Group("", h.authn.RequireAuthenticated())
	authed.POST("/logout", h.logout)
	authed.POST("/refresh", h.refresh)
	authed.GET("/me", h.me)

	u := rg.Group("/users", h.authn.RequireAuthenticated())
	// User CRUD — including reads — is system-admin only (auth.md §3.1).
	u.GET("", h.authn.RequireSystemAdmin(), h.listUsers)
	u.GET("/:id", h.authn.RequireSystemAdmin(), h.getUser)
	u.POST("", h.authn.RequireSystemAdmin(), h.createUser)
	u.PATCH("/:id", h.authn.RequireSystemAdmin(), h.updateUser)
	u.DELETE("/:id", h.authn.RequireSystemAdmin(), h.deleteUser)
	u.POST("/:id/password", h.setPassword)
}

func (h *Handler) login(c *gin.Context) {
	var req server.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req.Username, string(req.Password))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.LoginResponse{
		JWT:         res.Token,
		ExpiresAt:   res.ExpiresAt,
		User:        toUserView(res.User),
		TenantRoles: toTenantRoles(res.Roles),
	})
}

func (h *Handler) logout(c *gin.Context) {
	jti := auth.JTI(c)
	if err := h.svc.Logout(c.Request.Context(), jti); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) refresh(c *gin.Context) {
	id := auth.Current(c)
	res, err := h.svc.Refresh(c.Request.Context(), id.UserID, auth.JTI(c))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.RefreshResponse{JWT: res.Token, ExpiresAt: res.ExpiresAt})
}

func (h *Handler) me(c *gin.Context) {
	id := auth.Current(c)
	roles, err := h.svc.RolesOf(c.Request.Context(), id.UserID)
	if err != nil {
		server.Fail(c, err)
		return
	}
	u, err := h.svc.GetUser(c.Request.Context(), id.UserID)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, server.MeResponse{
		User:          toUserView(u),
		TenantRoles:   toTenantRoles(roles),
		Permissions:   id.Permissions(),
		IsSystemAdmin: id.IsSystemAdmin,
	})
}

func (h *Handler) listUsers(c *gin.Context) {
	page := server.ParsePage(c)
	users, err := h.svc.ListUsers(c.Request.Context(), c.Query("q"), page.Limit, page.Offset())
	if err != nil {
		server.Fail(c, err)
		return
	}
	items := make([]server.UserSummary, 0, len(users))
	for i := range users {
		items = append(items, toUserSummary(&users[i]))
	}
	c.JSON(http.StatusOK, server.UserSummaryList{
		Items:         items,
		Count:         len(items),
		ContinueToken: server.NextContinue(page.Offset(), page.Limit, len(items)),
	})
}

func (h *Handler) getUser(c *gin.Context) {
	u, err := h.svc.GetUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserView(u))
}

func (h *Handler) createUser(c *gin.Context) {
	var req server.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	u, err := h.svc.CreateUser(c.Request.Context(), req.Username, req.DisplayName, string(req.Email), string(req.Password))
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserView(u))
}

func (h *Handler) updateUser(c *gin.Context) {
	var req server.UserPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	u, err := h.svc.UpdateUser(c.Request.Context(), c.Param("id"), req.DisplayName, string(req.Email), req.Disabled)
	if err != nil {
		server.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserView(u))
}

func (h *Handler) deleteUser(c *gin.Context) {
	if err := h.svc.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) setPassword(c *gin.Context) {
	var req server.SetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.Fail(c, err)
		return
	}
	id := auth.Current(c)
	target := c.Param("id")
	// Only the user themselves or a system-admin may set a password.
	if !id.IsSystemAdmin && id.UserID != target {
		server.Fail(c, server.Forbidden())
		return
	}
	if err := h.svc.SetPassword(c.Request.Context(), target, string(req.CurrentPassword), string(req.NewPassword), id.IsSystemAdmin); err != nil {
		server.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- view mapping -----------------------------------------------------------

func toUserView(u *store.User) server.User {
	return server.User{
		ID:                 server.UUID(u.ID),
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Email:              server.Email(u.Email),
		Disabled:           u.Disabled,
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

func toUserSummary(u *store.User) server.UserSummary {
	return server.UserSummary{
		ID:          server.UUID(u.ID),
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Email:       server.Email(u.Email),
	}
}

func toTenantRoles(roles []store.UserRole) []server.UserTenantRole {
	out := make([]server.UserTenantRole, 0, len(roles))
	for _, r := range roles {
		out = append(out, server.UserTenantRole{TenantName: r.TenantName, RoleName: server.RoleName(r.Role)})
	}
	return out
}
