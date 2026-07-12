// Package tenant implements the Tenants, Quotas and Members tags: the durable
// tenant record (Platform PG) + Tenant CR materialisation/quota folding via
// cluster-manager, suspension gating, and membership with last-tenant-admin
// protection (backend.md §4.1, auth.md §4).
package tenant

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// Grouping labels carried by a Run's backing MLRun, and the non-terminal run
// phases that make a run "active". Kept local to avoid coupling the tenant
// module to the job/experiment packages.
const (
	labelJob        = "compute.axisml.io/job"
	labelExperiment = "compute.axisml.io/experiment"
)

var activeRunPhases = map[string]bool{
	"Creating": true, "Pending": true, "Running": true, "Canceling": true,
}

// Service holds tenant/quota/member business logic.
type Service struct {
	tenants    *store.TenantRepo
	roles      *store.RoleRepo
	users      *store.UserRepo
	cm         *clustermanager.Client
	compute    *computeservice.Client
	invalidate func(ctx context.Context, userID string)
}

// NewService constructs a tenant Service. compute may be nil (counts are then
// skipped); it is used only for best-effort live workload roll-ups.
func NewService(tenants *store.TenantRepo, roles *store.RoleRepo, users *store.UserRepo, cm *clustermanager.Client, compute *computeservice.Client) *Service {
	return &Service{tenants: tenants, roles: roles, users: users, cm: cm, compute: compute}
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
	Volumes             []VolumeSpec
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
		Volumes:       toCMVolumes(in.Volumes),
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

// Get returns a tenant with live status from cluster-manager and best-effort
// workload roll-ups (member / active-run / online-service counts).
func (s *Service) Get(ctx context.Context, identifier string) (*server.Tenant, error) {
	row, err := s.getRow(ctx, identifier)
	if err != nil {
		return nil, err
	}
	cr, _ := s.cm.GetTenant(ctx, identifier) // CR read is best-effort; nil ⇒ no live status
	view := buildView(row, cr)
	s.enrichCounts(ctx, view)
	return view, nil
}

// List returns tenants visible to the caller, enriched with live status
// (best-effort). When stats is set, each row is also enriched with workload
// roll-up counts (one compute-service call pair per tenant) — callers that only
// need names/scope (e.g. the tenant switcher) leave it off to stay cheap.
// partial reports whether any live enrichment failed.
func (s *Service) List(ctx context.Context, scope []string, q string, stats bool, limit, offset int) (items []*server.Tenant, partial bool, err error) {
	rows, err := s.tenants.List(ctx, q, scope, limit, offset)
	if err != nil {
		return nil, false, apperrors.Wrap(apperrors.ClassInternal, "list tenants", err)
	}
	// Each row's cluster-manager GetTenant (+ optional stats fan-out) is
	// independent, so run them with bounded concurrency: list latency stays flat
	// in the tenant count instead of O(N) sequential remote round-trips.
	out := make([]*server.Tenant, len(rows))
	var partialFlag atomic.Bool
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for i := range rows {
		i := i
		g.Go(func() error {
			cr, cerr := s.cm.GetTenant(gctx, rows[i].Identifier)
			if cerr != nil {
				partialFlag.Store(true)
				cr = nil
			}
			view := buildView(&rows[i], cr)
			if stats && !s.enrichCounts(gctx, view) {
				partialFlag.Store(true)
			}
			out[i] = view
			return nil
		})
	}
	_ = g.Wait() // workers never return an error; a failed source sets partial
	return out, partialFlag.Load(), nil
}

// enrichCounts fills the tenant's best-effort workload roll-ups in place and
// reports whether every source resolved cleanly. Failures are swallowed (the
// counts stay at zero) so a flaky compute-service never breaks the tenant list.
func (s *Service) enrichCounts(ctx context.Context, t *server.Tenant) bool {
	ok := true
	if n, err := s.roles.CountByTenant(ctx, t.Identifier); err == nil {
		t.MemberCount = int(n)
	} else {
		ok = false
	}
	if s.compute == nil {
		return ok
	}
	if runs, err := s.compute.ListMLRuns(ctx, t.Identifier, ""); err == nil {
		for i := range runs {
			if !activeRunPhases[runs[i].Phase] {
				continue
			}
			switch {
			case hasLabel(runs[i].Labels, labelExperiment):
				t.ActiveExperimentRuns++
			case hasLabel(runs[i].Labels, labelJob):
				t.ActiveJobRuns++
			}
		}
	} else {
		ok = false
	}
	if svcs, err := s.compute.ListMLServices(ctx, t.Identifier, ""); err == nil {
		for i := range svcs {
			if svcs[i].Kind == "service" && svcs[i].Status.ReadyReplicas > 0 {
				t.OnlineServices++
			}
		}
	} else {
		ok = false
	}
	return ok
}

func hasLabel(m *map[string]string, key string) bool {
	if m == nil {
		return false
	}
	_, ok := (*m)[key]
	return ok
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

// Delete removes a tenant after verifying it has no members and no live
// workloads. The CR is deleted first, then the durable record and membership
// rows (backend.md §4.1 "活跃业务资源非空 → tenant-in-use").
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
	if err := s.checkNoActiveWorkloads(ctx, identifier); err != nil {
		return err
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

// checkNoActiveWorkloads blocks tenant deletion while the namespace still holds
// active Runs or any online Service / Workspace, so a delete never orphans live
// compute. Unlike enrichCounts, list failures are fatal here: we must not tear a
// tenant down on incomplete information. compute==nil (unconfigured) skips it.
func (s *Service) checkNoActiveWorkloads(ctx context.Context, identifier string) error {
	if s.compute == nil {
		return nil
	}
	runs, err := s.compute.ListMLRuns(ctx, identifier, "")
	if err != nil {
		return err
	}
	for i := range runs {
		if activeRunPhases[runs[i].Phase] {
			return apperrors.New(apperrors.ClassConflict, "tenant still has active runs").WithReason("tenant-in-use")
		}
	}
	svcs, err := s.compute.ListMLServices(ctx, identifier, "")
	if err != nil {
		return err
	}
	for i := range svcs {
		switch svcs[i].Kind {
		case "service", "workspace":
			// Only a service/workspace with live pods blocks deletion; a stopped
			// one (scaled to zero) is dormant, mirroring the active-phase run gate
			// above rather than blocking on mere existence.
			if svcs[i].Status.ReadyReplicas > 0 {
				return apperrors.New(apperrors.ClassConflict, "tenant still has online services or workspaces").WithReason("tenant-in-use")
			}
		}
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
		cmq := clustermanager.Quota{Pool: q.Pool}
		if q.Direct != nil {
			cmq.Quota = toCMResources(q.Direct)
		} else {
			units := toCMUnits(q.Units)
			cmq.Units = &units
		}
		out = append(out, cmq)
	}
	return out
}
