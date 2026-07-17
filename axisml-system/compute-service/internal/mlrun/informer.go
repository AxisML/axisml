package mlrun

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
)

// Informer reflects MLRun CR status into PG. It is the Kubernetes status
// reflow; the standalone deployment uses StatusPoller instead. Both share the same
// writeback helpers (reflectObserved / reflectGone).
type Informer struct {
	db   *gorm.DB
	repo *Repository
	mgr  manager.Manager
	log  logr.Logger
}

// NewInformer constructs the job Informer.
func NewInformer(db *gorm.DB, mgr manager.Manager, log logr.Logger) *Informer {
	return &Informer{db: db, repo: NewRepository(db), mgr: mgr, log: log}
}

// NeedLeaderElection ensures only the leader writes back.
func (i *Informer) NeedLeaderElection() bool { return true }

func (i *Informer) Start(ctx context.Context) error {
	inf, err := i.mgr.GetCache().GetInformer(ctx, &mlrunv1alpha1.MLRun{})
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
	cr, ok := obj.(*mlrunv1alpha1.MLRun)
	if !ok {
		return
	}
	id, err := jobIDFromLabels(cr)
	if err != nil {
		return
	}
	j, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	reflectObserved(ctx, i.repo, j, cr.Status, time.Now().UTC())
}

func (i *Informer) onDelete(ctx context.Context, obj any) {
	cr, ok := obj.(*mlrunv1alpha1.MLRun)
	if !ok {
		return
	}
	id, err := jobIDFromLabels(cr)
	if err != nil {
		return
	}
	j, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	reflectGone(ctx, i.repo, j)
}
