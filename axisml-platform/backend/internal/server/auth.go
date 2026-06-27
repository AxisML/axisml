package server

import "time"

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Username string   `json:"username" binding:"required,min=1" desc:"Account username to authenticate."`
	Password Password `json:"password" binding:"required,min=1" desc:"Account password."`
}

// LoginResponse is returned on a successful login.
type LoginResponse struct {
	JWT         string           `json:"jwt" desc:"Signed JWT bearer token for subsequent requests."`
	ExpiresAt   time.Time        `json:"expiresAt,omitempty" desc:"Time the token expires."`
	User        User             `json:"user" desc:"Authenticated user account."`
	TenantRoles []UserTenantRole `json:"tenantRoles" desc:"Tenant/role bindings granted to the user."`
}

// RefreshResponse is returned by POST /auth/refresh.
type RefreshResponse struct {
	JWT       string    `json:"jwt" desc:"Newly issued JWT bearer token."`
	ExpiresAt time.Time `json:"expiresAt,omitempty" desc:"Time the new token expires."`
}

// UserTenantRole binds a user to a role within one tenant.
type UserTenantRole struct {
	TenantName string   `json:"tenantName" desc:"Tenant the role applies within."`
	RoleName   RoleName `json:"roleName" desc:"Role granted to the user in the tenant."`
}

// MeResponse describes the caller's identity and resolved permissions.
type MeResponse struct {
	User          User             `json:"user" desc:"The caller's user account."`
	TenantRoles   []UserTenantRole `json:"tenantRoles" desc:"The caller's tenant/role bindings."`
	Permissions   []string         `json:"permissions" desc:"Flattened permission strings resolved from the caller's roles."`
	IsSystemAdmin bool             `json:"isSystemAdmin" desc:"True if the caller holds the system administrator role."`
}

// User is a platform user account. MustChangePassword surfaces the forced
// first-login password reset so the frontend can gate the session until the
// user changes it (auth.md §2).
type User struct {
	ID                 UUID      `json:"id" desc:"Stable user identifier."`
	Username           string    `json:"username" desc:"Unique account username."`
	DisplayName        string    `json:"displayName,omitempty" desc:"Human-readable display name."`
	Email              Email     `json:"email,omitempty" desc:"Account email address."`
	Disabled           bool      `json:"disabled,omitempty" desc:"True if the account is disabled and cannot sign in."`
	MustChangePassword bool      `json:"mustChangePassword,omitempty" desc:"True if the user must change their password before continuing."`
	CreatedAt          time.Time `json:"createdAt" desc:"Time the account was created."`
	UpdatedAt          time.Time `json:"updatedAt" desc:"Time the account was last updated."`
}

// UserSummary is a compact user projection for list views.
type UserSummary struct {
	ID          UUID   `json:"id" desc:"Stable user identifier."`
	Username    string `json:"username" desc:"Unique account username."`
	DisplayName string `json:"displayName,omitempty" desc:"Human-readable display name."`
	Email       Email  `json:"email,omitempty" desc:"Account email address."`
}

// UserSummaryList is a page of UserSummary.
type UserSummaryList struct {
	Items         []UserSummary `json:"items" desc:"Users in this page."`
	Count         int           `json:"count" binding:"min=0" desc:"Number of users in this page."`
	ContinueToken string        `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
}

// UserCreateRequest is the body of POST /users.
type UserCreateRequest struct {
	Username    string   `json:"username" binding:"required,min=1,max=64" desc:"Unique account username."`
	DisplayName string   `json:"displayName,omitempty" binding:"max=100" desc:"Human-readable display name."`
	Email       Email    `json:"email,omitempty" desc:"Account email address."`
	Password    Password `json:"password" binding:"required,min=8" desc:"Initial account password."`
}

// UserPatchRequest is the body of PATCH /users/{id}. Disabled is a pointer so
// that omitting it leaves the account state untouched (sending a bare profile
// edit must not silently re-enable a disabled user).
type UserPatchRequest struct {
	DisplayName string `json:"displayName,omitempty" binding:"max=100" desc:"Updated human-readable display name."`
	Email       Email  `json:"email,omitempty" desc:"Updated account email address."`
	Disabled    *bool  `json:"disabled,omitempty" desc:"Set true to disable or false to enable the account; omit to leave unchanged."`
}

// SetPasswordRequest is the body of PUT /users/{id}/password.
type SetPasswordRequest struct {
	CurrentPassword Password `json:"currentPassword,omitempty" desc:"Existing password, required when a user changes their own password."`
	NewPassword     Password `json:"newPassword" binding:"required,min=8" desc:"New password to set."`
}
