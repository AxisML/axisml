package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrLastTenantAdmin is returned when a role change would drop a tenant's last
// tenant-admin (self-protection / zero-admin guard, auth.md §4).
var ErrLastTenantAdmin = errors.New("last tenant-admin")

// RoleRepo is the data access for tenant membership bindings (user_roles).
type RoleRepo struct{ db *gorm.DB }

// NewRoleRepo constructs a RoleRepo.
func NewRoleRepo(db *gorm.DB) *RoleRepo { return &RoleRepo{db: db} }

// ListByUser returns all tenant bindings for a user.
func (r *RoleRepo) ListByUser(ctx context.Context, userID string) ([]UserRole, error) {
	var roles []UserRole
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&roles).Error
	return roles, err
}

// ListByTenant returns all member bindings of a tenant.
func (r *RoleRepo) ListByTenant(ctx context.Context, tenant string) ([]UserRole, error) {
	var roles []UserRole
	err := r.db.WithContext(ctx).Where("tenant_name = ?", tenant).Find(&roles).Error
	return roles, err
}

// Get returns a user's binding in a tenant.
func (r *RoleRepo) Get(ctx context.Context, userID, tenant string) (*UserRole, error) {
	var ur UserRole
	err := r.db.WithContext(ctx).First(&ur, "user_id = ? AND tenant_name = ?", userID, tenant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &ur, err
}

// Set replaces a user's binding in a tenant with exactly one role row, in a
// transaction (a (user,tenant) pair holds at most one role), returning the new
// row so callers need no read-back.
func (r *RoleRepo) Set(ctx context.Context, userID, tenant, role string) (*UserRole, error) {
	out := UserRole{UserID: userID, TenantName: tenant, Role: role, CreatedAt: time.Now().UTC()}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND tenant_name = ?", userID, tenant).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		return tx.Create(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GuardedSet changes an existing member's role atomically, refusing to demote
// the last tenant-admin. It locks the tenant's admin rows so two concurrent
// demotions cannot both pass the guard. Returns ErrNotFound when the member has
// no binding and ErrLastTenantAdmin when the change would leave zero admins.
func (r *RoleRepo) GuardedSet(ctx context.Context, userID, tenant, newRole string) (*UserRole, error) {
	out := UserRole{UserID: userID, TenantName: tenant, Role: newRole, CreatedAt: time.Now().UTC()}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := guardLastAdminTx(tx, userID, tenant, newRole); err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND tenant_name = ?", userID, tenant).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		return tx.Create(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GuardedRemove deletes a member's binding atomically, refusing to remove the
// last tenant-admin. Same locking/error semantics as GuardedSet.
func (r *RoleRepo) GuardedRemove(ctx context.Context, userID, tenant string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := guardLastAdminTx(tx, userID, tenant, ""); err != nil {
			return err
		}
		return tx.Where("user_id = ? AND tenant_name = ?", userID, tenant).Delete(&UserRole{}).Error
	})
}

// guardLastAdminTx (inside a transaction) row-locks the tenant's tenant-admin
// bindings, verifies the (userID,tenant) binding exists, and refuses to
// demote/remove it when it is the last remaining admin. newRole=="" means
// removal; any non-"tenant-admin" newRole means demotion.
func guardLastAdminTx(tx *gorm.DB, userID, tenant, newRole string) error {
	var admins []UserRole
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_name = ? AND role = ?", tenant, "tenant-admin").Find(&admins).Error; err != nil {
		return err
	}
	var cur UserRole
	err := tx.First(&cur, "user_id = ? AND tenant_name = ?", userID, tenant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	demotingAdmin := cur.Role == "tenant-admin" && newRole != "tenant-admin"
	if demotingAdmin && len(admins) <= 1 {
		return ErrLastTenantAdmin
	}
	return nil
}

// Delete removes a user's binding in a tenant.
func (r *RoleRepo) Delete(ctx context.Context, userID, tenant string) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND tenant_name = ?", userID, tenant).Delete(&UserRole{}).Error
}

// DeleteByTenant clears all bindings of a tenant (tenant deletion cascade).
func (r *RoleRepo) DeleteByTenant(ctx context.Context, tenant string) error {
	return r.db.WithContext(ctx).Where("tenant_name = ?", tenant).Delete(&UserRole{}).Error
}

// CountByTenant returns the number of members bound to a tenant.
func (r *RoleRepo) CountByTenant(ctx context.Context, tenant string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&UserRole{}).Where("tenant_name = ?", tenant).Count(&n).Error
	return n, err
}
