package mlservice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// Informer reflects MLService CR status into PG. Writes go to phase
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
	// Don't override Deleting/Deleted.
	if Status(row.Phase) == StatusDeleting || Status(row.Phase) == StatusDeleted {
		return
	}

	desired := int32(0)
	if len(cr.Spec.Roles) > 0 {
		desired = cr.Spec.Roles[0].Replicas
	}
	newPhase := row.Phase
	switch {
	case desired == 0:
		newPhase = string(StatusPending)
	case cr.Status.ReadyReplicas == 0 && cr.Status.Phase == mlservicev1alpha1.PhasePending:
		newPhase = string(StatusPending)
	case cr.Status.ReadyReplicas == desired:
		newPhase = string(StatusReady)
	case cr.Status.ReadyReplicas > 0 && cr.Status.ReadyReplicas < desired:
		newPhase = string(StatusDegraded)
	case cr.Status.ReadyReplicas == 0 && cr.Status.Phase == mlservicev1alpha1.PhaseFailed:
		newPhase = string(StatusFailed)
	}

	var sf StatusFields
	if len(row.StatusJSON) > 0 {
		_ = json.Unmarshal(row.StatusJSON, &sf)
	}
	sf.ReadyReplicas = cr.Status.ReadyReplicas
	sf.Endpoint = cr.Status.Endpoint
	if cr.Status.Phase == mlservicev1alpha1.PhaseFailed && cr.Status.Message != "" {
		sf.Message = cr.Status.Message
	}
	b, _ := json.Marshal(sf)

	_ = i.repo.Update(ctx, id, map[string]any{
		"phase":  newPhase,
		"status": b,
	})
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
	switch Status(row.Phase) {
	case StatusDeleting:
		now := time.Now().UTC()
		_ = i.repo.Update(ctx, id, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
	case StatusPending, StatusReady, StatusDegraded, StatusFailed:
		// External delete during run → mark Deleting per design §5.4.
		var sf StatusFields
		if len(row.StatusJSON) > 0 {
			_ = json.Unmarshal(row.StatusJSON, &sf)
		}
		sf.Message = "external delete"
		b, _ := json.Marshal(sf)
		_ = i.repo.Update(ctx, id, map[string]any{
			"phase":      string(StatusDeleting),
			"deleted_at": time.Now().UTC(),
			"status":     b,
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
