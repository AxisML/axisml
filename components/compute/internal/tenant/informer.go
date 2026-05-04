package tenant

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	tenantv1alpha1 "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/quota"
)

// Informer watches Tenant CRs and reflects status into PG. It also drives
// the quota sub-resource state machine via Tenant.status.quotas[].
type Informer struct {
	db     *gorm.DB
	repo   *Repository
	mgr    manager.Manager
	quotas *quota.Service
	log    logr.Logger
}

// NewInformer builds an Informer.
func NewInformer(db *gorm.DB, mgr manager.Manager, quotas *quota.Service, log logr.Logger) *Informer {
	return &Informer{
		db:     db,
		repo:   NewRepository(db),
		mgr:    mgr,
		quotas: quotas,
		log:    log,
	}
}

// NeedLeaderElection: only the leader keeps the work queue authoritative.
func (i *Informer) NeedLeaderElection() bool { return true }

// Start subscribes to Tenant CRs.
func (i *Informer) Start(ctx context.Context) error {
	inf, err := i.mgr.GetCache().GetInformer(ctx, &tenantv1alpha1.Tenant{})
	if err != nil {
		return err
	}
	_, err = inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { i.onChange(ctx, obj) },
		UpdateFunc: func(_, newObj any) { i.onChange(ctx, newObj) },
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
	id, err := lookupTenantID(cr)
	if err != nil {
		return
	}
	t, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	updates := map[string]any{}
	switch cr.Status.Phase {
	case tenantv1alpha1.TenantPhaseActive:
		if t.Status == string(StatusCreating) || t.Status == string(StatusSuspended) {
			updates["status"] = string(StatusActive)
		}
	case tenantv1alpha1.TenantPhaseSuspended, tenantv1alpha1.TenantPhaseFailed:
		if t.Status != string(StatusDeleting) && t.Status != string(StatusDeleted) {
			updates["status"] = string(StatusSuspended)
			updates["message"] = cr.Status.Message
		}
	}
	if len(updates) > 0 {
		_ = i.repo.Update(ctx, nil, id, updates)
	}
	i.syncQuotaStatus(ctx, id, cr)
}

func (i *Informer) onDelete(ctx context.Context, obj any) {
	cr, ok := obj.(*tenantv1alpha1.Tenant)
	if !ok {
		return
	}
	id, err := lookupTenantID(cr)
	if err != nil {
		return
	}
	t, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	if t.Status == string(StatusDeleting) {
		now := time.Now().UTC()
		_ = i.repo.Update(ctx, nil, id, map[string]any{
			"status":     string(StatusDeleted),
			"deleted_at": now,
		})
	}
	// Quota cascade: mark all quotas under this tenant as Deleted.
	if i.quotas != nil {
		_ = i.quotas.SoftDeleteAllByTenant(ctx, nil, id)
	}
}

func (i *Informer) syncQuotaStatus(ctx context.Context, tenantID uuid.UUID, cr *tenantv1alpha1.Tenant) {
	if i.quotas == nil {
		return
	}
	obs := make([]quota.StatusObservation, 0, len(cr.Status.Quotas))
	for _, q := range cr.Status.Quotas {
		obs = append(obs, quota.StatusObservation{
			Pool:  q.Pool,
			Name:  q.Name,
			Ready: q.Ready,
			Used:  q.Used,
		})
	}
	if err := i.quotas.SyncFromTenantStatus(ctx, tenantID, obs); err != nil {
		i.log.Error(err, "sync quota status")
	}
}

func lookupTenantID(cr *tenantv1alpha1.Tenant) (uuid.UUID, error) {
	if v, ok := cr.Labels[tenantv1alpha1.LabelTenantID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing axisml.io/tenant-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
