// Package guard holds the shared authorization/precondition checks used by the
// workload and artifact modules: the tenant suspension gate (§4.1) and the
// owner-or-tenant-admin object check (auth.md §3.1). Centralising these keeps
// every workload-create path from re-implementing (or forgetting) them.
package guard

import (
	"context"
	"errors"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// TenantActive asserts the tenant exists and is not suspended. Workload-create
// entry points call this before triggering compute (suspended → 409).
func TenantActive(ctx context.Context, tenants *store.TenantRepo, identifier string) error {
	t, err := tenants.GetByIdentifier(ctx, identifier)
	if errors.Is(err, store.ErrNotFound) {
		return server.NotFound("tenant not found")
	}
	if err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "lookup tenant", err)
	}
	if t.SuspendedAt != nil {
		return apperrors.New(apperrors.ClassConflict, "tenant is suspended").WithReason("tenant-suspended")
	}
	return nil
}

// OwnerOrTenantAdmin allows the object owner, a tenant-admin of the object's
// tenant, or a system-admin; otherwise 403. owner is the object's creator.
func OwnerOrTenantAdmin(id *auth.Identity, tenant, owner string) error {
	if id.IsSystemAdmin {
		return nil
	}
	if id.HasTenantRole(tenant, auth.RoleTenantAdmin) {
		return nil
	}
	if owner != "" && owner == id.Username {
		return nil
	}
	return server.Forbidden()
}
