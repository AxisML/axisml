package store

import "time"

// Tenant is the durable tenant record (hard-deleted, no deleted_at).
type Tenant struct {
	ID                  string `gorm:"primaryKey"`
	Identifier          string
	KubernetesNamespace string
	DisplayName         string
	Description         string
	Owner               string
	Labels              StrMap
	Annotations         StrMap
	SuspendedAt         *time.Time
	LastModifiedBy      string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TableName pins the table.
func (Tenant) TableName() string { return "tenants" }

// User is a Platform user account.
type User struct {
	ID                 string `gorm:"primaryKey"`
	Username           string
	PasswordHash       string
	MustChangePassword bool
	Email              string
	DisplayName        string
	Disabled           bool
	IsSystemAdmin      bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TableName pins the table.
func (User) TableName() string { return "users" }

// UserRole binds a user to a role within one tenant (tenant-admin | user).
type UserRole struct {
	UserID     string `gorm:"primaryKey"`
	TenantName string `gorm:"primaryKey"`
	Role       string `gorm:"primaryKey"`
	CreatedAt  time.Time
}

// TableName pins the table.
func (UserRole) TableName() string { return "user_roles" }

// Session is a JWT session row keyed by jti (blacklist on revoke / expiry).
type Session struct {
	JTI       string `gorm:"primaryKey;column:jti"`
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
}

// TableName pins the table.
func (Session) TableName() string { return "sessions" }

// Definition is the shared row shape for the four name-level definition tables
// (jobs / experiments / models / images). The concrete table is selected by the
// repository, not by a TableName method.
type Definition struct {
	ID          string `gorm:"primaryKey"`
	TenantName  string
	Name        string
	DisplayName string
	Description string
	OwnerUser   string
	Labels      StrMap
	Annotations StrMap
	Spec        JSONB
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
