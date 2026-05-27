package tenant

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// Informer reflects Tenant CR status into the PG `tenants` row. Status
// jsonb stores the post-strip `{message, namespaceReady, conditions[],
// quotas[].{pool, name, ready}}` tree; quotas[].used is deliberately
// stripped here — that field is ephemeral and lives only in tenant-
// operator's CR cache (compute reads it on demand, never persists).
type Informer struct {
	db   *gorm.DB
	repo *Repository
	mgr  manager.Manager
	log  logr.Logger
}

func NewInformer(db *gorm.DB, mgr manager.Manager, log logr.Logger) *Informer {
	return &Informer{db: db, repo: NewRepository(db), mgr: mgr, log: log}
}

func (i *Informer) NeedLeaderElection() bool { return true }

func (i *Informer) Start(ctx context.Context) error {
	inf, err := i.mgr.GetCache().GetInformer(ctx, &tenantv1alpha1.Tenant{})
	if err != nil {
		return err
	}
	_, err = inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { i.onChange(ctx, obj) },
		UpdateFunc: func(_, n any) { i.onChange(ctx, n) },
		DeleteFunc: func(obj any) { i.onDelete(ctx, obj) },
	})
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

func (i *Informer) onChange(ctx context.Context, obj any) {
	cr, ok := obj.(*tenantv1alpha1.Tenant)
	if !ok {
		return
	}
	id, err := tenantIDFromLabels(cr)
	if err != nil {
		return
	}
	row, err := i.repo.Get(ctx, id)
	if err != nil {
		// Reverse-orphan: CR exists but the PG row doesn't. Per design §5.5,
		// the default response is to delete the CR + record an audit event,
		// trusting that PG is the authority. Helm-seeded tenants (e.g. the
		// `axisml-system` bootstrap entry) land here legitimately on a
		// fresh cluster — admins can flip `seed.tenant.skipInfomerGC=true`
		// or hand-create the matching PG row before the first reconcile;
		// for now we log + audit but DO NOT delete, since seed tenants are
		// load-bearing for visibility=public artifacts.
		if IsNotFound(err) {
			i.log.Info("reverse-orphan Tenant CR observed (no PG row); leaving in place — admin must reconcile",
				"name", cr.Name, "tenant-id", cr.Labels[tenantv1alpha1.LabelTenantID])
		}
		return
	}

	updates := map[string]any{}

	switch cr.Status.Phase {
	case tenantv1alpha1.TenantPhaseActive:
		if row.Phase == PhaseCreating || row.Phase == PhaseFailed {
			updates["phase"] = PhaseActive
		}
	case tenantv1alpha1.TenantPhaseFailed:
		if row.Phase != PhaseDeleting && row.Phase != PhaseDeleted {
			updates["phase"] = PhaseFailed
		}
	}

	// Status jsonb: strip quotas[].used (ephemeral) and persist the rest.
	status := strippedStatus(cr)
	if b, err := json.Marshal(status); err == nil {
		updates["status"] = b
	}

	if len(updates) > 0 {
		_ = i.repo.Update(ctx, id, updates)
	}
}

func (i *Informer) onDelete(ctx context.Context, obj any) {
	cr, ok := obj.(*tenantv1alpha1.Tenant)
	if !ok {
		return
	}
	id, err := tenantIDFromLabels(cr)
	if err != nil {
		return
	}
	row, err := i.repo.Get(ctx, id)
	if err != nil {
		// Row already deleted or out of sync — nothing to do.
		return
	}
	if row.Phase == PhaseDeleting {
		_ = i.repo.Update(ctx, id, map[string]any{
			"phase": PhaseDeleted,
		})
		return
	}
	// External delete (someone kubectl'd the CR while compute thought the
	// tenant was Active) → re-create on the next reconciler tick (Reconciler
	// patch path catches generation != observed_generation; we nudge here).
	_ = i.repo.Update(ctx, id, map[string]any{
		"observed_generation": int64(0),
	})
}

// strippedStatus mirrors design §5.3: persist everything from the CR's
// .status except quotas[].used. We don't pull the full Status struct
// into PG; just the subset needed for GET responses.
func strippedStatus(cr *tenantv1alpha1.Tenant) map[string]any {
	out := map[string]any{
		"message":        cr.Status.Message,
		"namespaceReady": cr.Status.NamespaceReady,
		"conditions":     cr.Status.Conditions,
	}
	type quotaRow struct {
		Pool  string `json:"pool"`
		Name  string `json:"name"`
		Ready bool   `json:"ready"`
	}
	qs := make([]quotaRow, 0, len(cr.Status.Quotas))
	for _, q := range cr.Status.Quotas {
		qs = append(qs, quotaRow{Pool: q.Pool, Name: q.Name, Ready: q.Ready})
	}
	out["quotas"] = qs
	out["observedAt"] = time.Now().UTC().Format(time.RFC3339)
	return out
}

func tenantIDFromLabels(cr *tenantv1alpha1.Tenant) (uuid.UUID, error) {
	if v, ok := cr.Labels[tenantv1alpha1.LabelTenantID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing axisml.io/tenant-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
