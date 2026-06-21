// Package tenant implements the Tenants, Quotas and Members tags: the durable
// tenant record (Platform PG) + Tenant CR materialisation/quota folding via
// cluster-manager, suspension gating, and membership with last-tenant-admin
// protection (backend.md §4.1, auth.md §4).
package tenant

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/clients/clustermanager"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// Service holds tenant/quota/member business logic.
type Service struct {
	tenants    *store.TenantRepo
	roles      *store.RoleRepo
	users      *store.UserRepo
	cm         *clustermanager.Client
	invalidate func(ctx context.Context, userID string)
}

// NewService constructs a tenant Service.
func NewService(tenants *store.TenantRepo, roles *store.RoleRepo, users *store.UserRepo, cm *clustermanager.Client) *Service {
	return &Service{tenants: tenants, roles: roles, users: users, cm: cm}
}

// OnIdentityChange registers a hook invoked after any membership change, so the
// affected user's cached identity (which carries their tenant bindings) can be
// busted. No-op when unset.
func (s *Service) OnIdentityChange(f func(ctx context.Context, userID string)) *Service {
	s.invalidate = f
	return s
}

func (s *Service) bust(ctx context.Context, userID string) {
	if s.invalidate != nil {
		s.invalidate(ctx, userID)
	}
}

// CreateInput is the create-tenant payload.
type CreateInput struct {
	Identifier          string
	KubernetesNamespace string
	DisplayName         string
	Description         string
	InitialAdmin        string // email or username
	Labels              map[string]string
	Annotations         map[string]string
	Quotas              []QuotaSpec
}

// Create writes the durable record, materialises the Tenant CR, and binds the
// initial tenant-admin. On CR failure the PG row is rolled back.
func (s *Service) Create(ctx context.Context, in CreateInput, owner string) (*server.Tenant, error) {
	admin, err := s.resolveAccount(ctx, in.InitialAdmin)
	if err != nil {
		return nil, err
	}
	if _, err := s.tenants.GetByIdentifier(ctx, in.Identifier); err == nil {
		return nil, apperrors.New(apperrors.ClassConflict, "tenant already exists").WithReason("tenant-exists")
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup tenant", err)
	}

	row := &store.Tenant{
		Identifier:          in.Identifier,
		KubernetesNamespace: in.KubernetesNamespace,
		DisplayName:         in.DisplayName,
		Description:         in.Description,
		Owner:               owner,
		Labels:              in.Labels,
		Annotations:         in.Annotations,
		LastModifiedBy:      owner,
	}
	if err := s.tenants.Create(ctx, row); err != nil {
		// A racing duplicate that slipped past the pre-check trips the unique
		// index; surface it as 409, not 500.
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, apperrors.New(apperrors.ClassConflict, "tenant already exists").WithReason("tenant-exists")
		}
		return nil, apperrors.Wrap(apperrors.ClassInternal, "create tenant row", err)
	}

	cmIn := clustermanager.CreateTenantInput{
		Name:          in.Identifier,
		NamespaceName: in.KubernetesNamespace,
		DisplayName:   in.DisplayName,
		Labels:        in.Labels,
		Annotations:   in.Annotations,
		Quotas:        toCMQuotas(in.Quotas),
	}
	cr, err := s.cm.CreateTenant(ctx, cmIn)
	if err != nil {
		_ = s.tenants.Delete(ctx, in.Identifier) // rollback durable record
		return nil, err
	}

	if _, err := s.roles.Set(ctx, admin.ID, in.Identifier, string(auth.RoleTenantAdmin)); err != nil {
		_ = s.cm.DeleteTenant(ctx, in.Identifier)
		_ = s.tenants.Delete(ctx, in.Identifier)
		return nil, apperrors.Wrap(apperrors.ClassInternal, "bind initial admin", err)
	}
	s.bust(ctx, admin.ID)
	return buildView(row, cr), nil
}

// Get returns a tenant with live status from cluster-manager.
func (s *Service) Get(ctx context.Context, identifier string) (*server.Tenant, error) {
	row, err := s.getRow(ctx, identifier)
	if err != nil {
		return nil, err
	}
	cr, err := s.cm.GetTenant(ctx, identifier)
	if err != nil {
		// durable record exists but CR read failed: surface the row without live status.
		return buildView(row, nil), nil
	}
	return buildView(row, cr), nil
}

// List returns tenants visible to the caller, enriched with live status
// (best-effort). partial reports whether any live enrichment failed.
func (s *Service) List(ctx context.Context, scope []string, q string, limit, offset int) (items []*server.Tenant, partial bool, err error) {
	rows, err := s.tenants.List(ctx, q, scope, limit, offset)
	if err != nil {
		return nil, false, apperrors.Wrap(apperrors.ClassInternal, "list tenants", err)
	}
	out := make([]*server.Tenant, 0, len(rows))
	for i := range rows {
		cr, cerr := s.cm.GetTenant(ctx, rows[i].Identifier)
		if cerr != nil {
			partial = true
			cr = nil
		}
		out = append(out, buildView(&rows[i], cr))
	}
	return out, partial, nil
}

// UpdateMeta edits display metadata and syncs labels/annotations to the CR.
func (s *Service) UpdateMeta(ctx context.Context, identifier, displayName, description string, labels, annotations map[string]string, user string) (*server.Tenant, error) {
	row, err := s.getRow(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if displayName != "" {
		row.DisplayName = displayName
	}
	if description != "" {
		row.Description = description
	}
	if labels != nil {
		row.Labels = labels
	}
	if annotations != nil {
		row.Annotations = annotations
	}
	row.LastModifiedBy = user
	if err := s.tenants.UpdateMeta(ctx, row); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "update tenant", err)
	}
	cr, _ := s.cm.UpdateTenant(ctx, identifier, row.Labels, row.Annotations)
	return buildView(row, cr), nil
}

// Delete removes a tenant after verifying it has no members. The CR is deleted
// first, then the durable record and membership rows.
//
// TODO(workloads): also block on active Runs/Services/Workspaces once the
// compute client wrapper lands (backend.md §4.1 "活跃业务资源非空 → tenant-in-use").
func (s *Service) Delete(ctx context.Context, identifier string) error {
	if _, err := s.getRow(ctx, identifier); err != nil {
		return err
	}
	n, err := s.roles.CountByTenant(ctx, identifier)
	if err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "count members", err)
	}
	if n > 0 {
		return apperrors.New(apperrors.ClassConflict, "tenant still has members").WithReason("tenant-in-use")
	}
	if err := s.cm.DeleteTenant(ctx, identifier); err != nil {
		return err
	}
	if err := s.roles.DeleteByTenant(ctx, identifier); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "clear members", err)
	}
	if err := s.tenants.Delete(ctx, identifier); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "delete tenant", err)
	}
	return nil
}

// SetSuspended sets or clears the suspension gate (Platform PG only).
func (s *Service) SetSuspended(ctx context.Context, identifier string, suspend bool) (*server.Tenant, error) {
	row, err := s.getRow(ctx, identifier)
	if err != nil {
		return nil, err
	}
	var at *time.Time
	if suspend {
		now := time.Now().UTC()
		at = &now
	}
	if err := s.tenants.SetSuspended(ctx, identifier, at); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "set suspension", err)
	}
	row.SuspendedAt = at
	cr, _ := s.cm.GetTenant(ctx, identifier)
	return buildView(row, cr), nil
}

func (s *Service) getRow(ctx context.Context, identifier string) (*store.Tenant, error) {
	row, err := s.tenants.GetByIdentifier(ctx, identifier)
	if errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.New(apperrors.ClassNotFound, "tenant not found").WithReason("not-found")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup tenant", err)
	}
	return row, nil
}

func (s *Service) resolveAccount(ctx context.Context, account string) (*store.User, error) {
	account = strings.TrimSpace(account)
	var (
		u   *store.User
		err error
	)
	if strings.Contains(account, "@") {
		u, err = s.users.GetByEmail(ctx, account)
	} else {
		u, err = s.users.GetByUsername(ctx, account)
	}
	if errors.Is(err, store.ErrNotFound) {
		return nil, apperrors.Newf(apperrors.ClassUnprocessable, "user %q not found", account).WithReason("user-not-found")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "resolve account", err)
	}
	return u, nil
}

func toCMQuotas(qs []QuotaSpec) []clustermanager.Quota {
	out := make([]clustermanager.Quota, 0, len(qs))
	for _, q := range qs {
		units := make([]clustermanager.QuotaUnit, 0, len(q.Units))
		for _, u := range q.Units {
			units = append(units, clustermanager.QuotaUnit{UnitName: u.UnitName, Quantity: u.Quantity})
		}
		out = append(out, clustermanager.Quota{Pool: q.Pool, Units: units})
	}
	return out
}
