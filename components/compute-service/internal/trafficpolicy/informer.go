package trafficpolicy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Informer reflects MLTrafficPolicy CR status into PG. Writes go to phase
// (high-frequency filter) + status jsonb (message / endpoint / backends[] /
// conditions[]).
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
	inf, err := i.mgr.GetCache().GetInformer(ctx, &mltp.MLTrafficPolicy{})
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
	cr, ok := obj.(*mltp.MLTrafficPolicy)
	if !ok {
		return
	}
	id, err := policyIDFromLabels(cr)
	if err != nil {
		return
	}
	row, err := i.repo.Get(ctx, id)
	if err != nil {
		return
	}
	if Status(row.Phase) == StatusDeleting || Status(row.Phase) == StatusDeleted {
		return
	}

	// CR phase (Pending/Ready/Degraded/Failed) maps 1:1 onto the PG phase.
	newPhase := row.Phase
	switch cr.Status.Phase {
	case mltp.PhasePending:
		newPhase = string(StatusPending)
	case mltp.PhaseReady:
		newPhase = string(StatusReady)
	case mltp.PhaseDegraded:
		newPhase = string(StatusDegraded)
	case mltp.PhaseFailed:
		newPhase = string(StatusFailed)
	}

	var sf server.TrafficPolicyStatus
	if len(row.StatusJSON) > 0 {
		_ = json.Unmarshal(row.StatusJSON, &sf)
	}
	sf.Endpoint = cr.Status.Endpoint
	sf.Message = cr.Status.Message
	sf.Backends = sf.Backends[:0]
	for _, b := range cr.Status.Backends {
		sf.Backends = append(sf.Backends, server.TrafficPolicyBackendStatus{
			ServiceName: b.ServiceName,
			Weight:      b.Weight,
			Ready:       b.Ready,
		})
	}
	b, _ := json.Marshal(sf)

	_ = i.repo.Update(ctx, id, map[string]any{
		"phase":  newPhase,
		"status": b,
	})
}

func (i *Informer) onDelete(ctx context.Context, obj any) {
	cr, ok := obj.(*mltp.MLTrafficPolicy)
	if !ok {
		return
	}
	id, err := policyIDFromLabels(cr)
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
		// External delete while active: a traffic policy is pure declarative
		// routing and PG is authoritative, so the reconciler rebuilds the CR
		// (it stays Creating-eligible via generation). We do not converge to
		// Deleted here (compute-service.md §5.5). Bump generation so the
		// spec-sync predicate re-emits the CR.
		_ = i.repo.Update(ctx, id, map[string]any{
			"observed_generation": row.Generation - 1,
		})
	}
}

func policyIDFromLabels(cr *mltp.MLTrafficPolicy) (uuid.UUID, error) {
	if v, ok := cr.Labels[mltp.LabelTrafficPolicyID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing axisml.io/traffic-policy-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
