package mlservice

import (
	"context"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlservice/v1alpha1"
)

// Informer reflects MLService CR status into PG. It is the Kubernetes status
// reflow; the Lite form uses StatusPoller instead. Both share the same
// writeback helpers (reflectObserved / reflectGone). Writes go to phase
// (high-frequency filter) + status jsonb (message / readyReplicas /
// endpoint / conditions[]).
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
	inf, err := i.mgr.GetCache().GetInformer(ctx, &mlservicev1alpha1.MLService{})
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
	cr, ok := obj.(*mlservicev1alpha1.MLService)
	if !ok {
		return
	}
	id, err := serviceIDFromLabels(cr)
	if err != nil {
		return
	}
	row, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	desired := int32(0)
	if len(cr.Spec.Roles) > 0 {
		desired = cr.Spec.Roles[0].Replicas
	}
	reflectObserved(ctx, i.repo, row, desired, cr.Status)
}

func (i *Informer) onDelete(ctx context.Context, obj any) {
	cr, ok := obj.(*mlservicev1alpha1.MLService)
	if !ok {
		return
	}
	id, err := serviceIDFromLabels(cr)
	if err != nil {
		return
	}
	row, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	reflectGone(ctx, i.repo, row)
}
