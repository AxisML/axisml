package server

import "time"

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Username string   `json:"username" binding:"required,min=1"`
	Password Password `json:"password" binding:"required,min=1"`
}

// LoginResponse is returned on a successful login.
type LoginResponse struct {
	JWT         string           `json:"jwt"`
	ExpiresAt   time.Time        `json:"expiresAt,omitempty"`
	User        User             `json:"user"`
	TenantRoles []UserTenantRole `json:"tenantRoles"`
}

// RefreshResponse is returned by POST /auth/refresh.
type RefreshResponse struct {
	JWT       string    `json:"jwt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// UserTenantRole binds a user to a role within one tenant.
type UserTenantRole struct {
	TenantName string   `json:"tenantName"`
	RoleName   RoleName `json:"roleName"`
}

// MeResponse describes the caller's identity and resolved permissions.
type MeResponse struct {
	User          User             `json:"user"`
	TenantRoles   []UserTenantRole `json:"tenantRoles"`
	Permissions   []string         `json:"permissions"`
	IsSystemAdmin bool             `json:"isSystemAdmin"`
}

// User is a platform user account. MustChangePassword surfaces the forced
// first-login password reset so the frontend can gate the session until the
// user changes it (auth.md §2).
type User struct {
	ID                 UUID      `json:"id"`
	Username           string    `json:"username"`
	DisplayName        string    `json:"displayName,omitempty"`
	Email              Email     `json:"email,omitempty"`
	Disabled           bool      `json:"disabled,omitempty"`
	MustChangePassword bool      `json:"mustChangePassword,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// UserSummary is a compact user projection for list views.
type UserSummary struct {
	ID          UUID   `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName,omitempty"`
	Email       Email  `json:"email,omitempty"`
}

// UserSummaryList is a page of UserSummary.
type UserSummaryList struct {
	Items         []UserSummary `json:"items"`
	Count         int           `json:"count" binding:"min=0"`
	ContinueToken string        `json:"continueToken,omitempty"`
}

// UserCreateRequest is the body of POST /users.
type UserCreateRequest struct {
	Username    string   `json:"username" binding:"required,min=1,max=64"`
	DisplayName string   `json:"displayName,omitempty" binding:"max=100"`
	Email       Email    `json:"email,omitempty"`
	Password    Password `json:"password" binding:"required,min=8"`
}

// UserPatchRequest is the body of PATCH /users/{id}. Disabled is a pointer so
// that omitting it leaves the account state untouched (sending a bare profile
// edit must not silently re-enable a disabled user).
type UserPatchRequest struct {
	DisplayName string `json:"displayName,omitempty" binding:"max=100"`
	Email       Email  `json:"email,omitempty"`
	Disabled    *bool  `json:"disabled,omitempty"`
}

// SetPasswordRequest is the body of PUT /users/{id}/password.
type SetPasswordRequest struct {
	CurrentPassword Password `json:"currentPassword,omitempty"`
	NewPassword     Password `json:"newPassword" binding:"required,min=8"`
}
