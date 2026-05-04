package service

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mlservicev1alpha1 "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
)

// Informer reflects MLService CR status into PG.
type Informer struct {
	db   *gorm.DB
	repo *Repository
	mgr  manager.Manager
	log  logr.Logger
}

// NewInformer constructs the service Informer.
func NewInformer(db *gorm.DB, mgr manager.Manager, log logr.Logger) *Informer {
	return &Informer{db: db, repo: NewRepository(db), mgr: mgr, log: log}
}

// NeedLeaderElection ensures only the leader writes back.
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
	updates := map[string]any{
		"ready_replicas": cr.Status.ReadyReplicas,
		"endpoint":       cr.Status.Endpoint,
	}
	desired := int32(0)
	if len(cr.Spec.Roles) > 0 {
		desired = cr.Spec.Roles[0].Replicas
	}
	switch {
	case desired == 0:
		updates["status"] = string(StatusPending)
	case cr.Status.ReadyReplicas == 0 && cr.Status.Phase == mlservicev1alpha1.PhasePending:
		updates["status"] = string(StatusPending)
	case cr.Status.ReadyReplicas == desired:
		updates["status"] = string(StatusReady)
	case cr.Status.ReadyReplicas > 0 && cr.Status.ReadyReplicas < desired:
		updates["status"] = string(StatusDegraded)
	case cr.Status.ReadyReplicas == 0 && cr.Status.Phase == mlservicev1alpha1.PhaseFailed:
		updates["status"] = string(StatusFailed)
		if cr.Status.Message != "" {
			updates["message"] = cr.Status.Message
		}
	}
	// Don't override Deleting/Deleted.
	if Status(row.Status) == StatusDeleting || Status(row.Status) == StatusDeleted {
		return
	}
	_ = i.repo.Update(ctx, id, updates)
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
	switch Status(row.Status) {
	case StatusDeleting:
		now := time.Now().UTC()
		_ = i.repo.Update(ctx, id, map[string]any{"status": string(StatusDeleted), "deleted_at": now})
	case StatusPending, StatusReady, StatusDegraded, StatusFailed:
		_ = i.repo.Update(ctx, id, map[string]any{
			"status":     string(StatusDeleting),
			"deleted_at": time.Now().UTC(),
			"message":    "external delete",
		})
	}
}

func serviceIDFromLabels(cr *mlservicev1alpha1.MLService) (uuid.UUID, error) {
	if v, ok := cr.Labels[mlservicev1alpha1.LabelServiceID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing axisml.io/service-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
