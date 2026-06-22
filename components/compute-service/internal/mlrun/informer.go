package mlrun

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/server"
)

// Informer reflects MLRun CR status into PG.
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

	// Build the next phase + status sub-tree by merging CR status into
	// the existing PG row.
	var prevStatus server.MLRunStatus
	if len(j.StatusJSON) > 0 {
		_ = json.Unmarshal(j.StatusJSON, &prevStatus)
	}
	nextStatus := prevStatus
	newPhase := j.Phase

	switch cr.Status.Phase {
	case mlrunv1alpha1.PhasePending:
		if j.Phase == string(StatusCreating) {
			newPhase = string(StatusPending)
		}
	case mlrunv1alpha1.PhaseRunning:
		if Status(j.Phase) == StatusCreating || Status(j.Phase) == StatusPending {
			newPhase = string(StatusRunning)
		}
		if prevStatus.StartedAt == nil && cr.Status.StartedAt != nil {
			t := cr.Status.StartedAt.Time
			nextStatus.StartedAt = &t
		}
	case mlrunv1alpha1.PhaseSucceeded:
		newPhase = string(StatusSucceeded)
		t := terminalTime(cr)
		nextStatus.FinishedAt = &t
	case mlrunv1alpha1.PhaseFailed:
		newPhase = string(StatusFailed)
		t := terminalTime(cr)
		nextStatus.FinishedAt = &t
		if cr.Status.Message != "" {
			nextStatus.Message = cr.Status.Message
		}
	}

	// Suspended condition with reason=CancelRequested → Cancelled.
	if Status(j.Phase) == StatusCanceling && hasCondition(cr.Status.Conditions, mlrunv1alpha1.ConditionSuspended, mlrunv1alpha1.ReasonCancelRequested) {
		newPhase = string(StatusCancelled)
		t := time.Now().UTC()
		nextStatus.FinishedAt = &t
	}

	b, _ := json.Marshal(nextStatus)
	prevB, _ := json.Marshal(prevStatus)
	if newPhase == j.Phase && string(b) == string(prevB) {
		return
	}
	_ = i.repo.Update(ctx, id, map[string]any{
		"phase":  newPhase,
		"status": b,
	})
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
	switch Status(j.Phase) {
	case StatusDeleting:
		now := time.Now().UTC()
		_ = i.repo.Update(ctx, id, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": now,
		})
	case StatusPending, StatusRunning:
		// External delete during run → mark Cancelled per design §5.4.
		now := time.Now().UTC()
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.Message = "external delete"
			s.FinishedAt = &now
		})
		_ = i.repo.Update(ctx, id, map[string]any{
			"phase":  string(StatusCancelled),
			"status": next,
		})
	case StatusCanceling:
		now := time.Now().UTC()
		next := mergeStatusFields(j.StatusJSON, func(s *server.MLRunStatus) {
			s.FinishedAt = &now
		})
		_ = i.repo.Update(ctx, id, map[string]any{
			"phase":  string(StatusCancelled),
			"status": next,
		})
	}
}

func mergeStatusFields(raw []byte, mutate func(*server.MLRunStatus)) []byte {
	var sf server.MLRunStatus
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &sf)
	}
	mutate(&sf)
	b, _ := json.Marshal(sf)
	return b
}

func jobIDFromLabels(cr *mlrunv1alpha1.MLRun) (uuid.UUID, error) {
	if v, ok := cr.Labels[mlrunv1alpha1.LabelRunID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

func terminalTime(cr *mlrunv1alpha1.MLRun) time.Time {
	if cr.Status.FinishedAt != nil {
		return cr.Status.FinishedAt.Time
	}
	return time.Now().UTC()
}

func hasCondition(conds []metav1.Condition, kind, reason string) bool {
	for _, c := range conds {
		if c.Type == kind && c.Reason == reason && c.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

var errMissingID = sentinel("missing axisml.io/run-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
