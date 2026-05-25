package job

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
)

// Informer reflects MLJob CR status into PG.
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
	inf, err := i.mgr.GetCache().GetInformer(ctx, &mljobv1alpha1.MLJob{})
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
	cr, ok := obj.(*mljobv1alpha1.MLJob)
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
	updates := map[string]any{}

	switch cr.Status.Phase {
	case mljobv1alpha1.PhasePending:
		if j.Status == string(StatusCreating) {
			updates["status"] = string(StatusPending)
		}
	case mljobv1alpha1.PhaseRunning:
		if Status(j.Status) == StatusCreating || Status(j.Status) == StatusPending {
			updates["status"] = string(StatusRunning)
		}
		if j.StartedAt == nil && cr.Status.StartedAt != nil {
			t := cr.Status.StartedAt.Time
			updates["started_at"] = t
		}
	case mljobv1alpha1.PhaseSucceeded:
		updates["status"] = string(StatusSucceeded)
		updates["finished_at"] = terminalTime(cr)
	case mljobv1alpha1.PhaseFailed:
		updates["status"] = string(StatusFailed)
		updates["finished_at"] = terminalTime(cr)
		if cr.Status.Message != "" {
			updates["message"] = cr.Status.Message
		}
	}

	// Suspended condition with reason=CancelRequested → Cancelled.
	if Status(j.Status) == StatusCanceling && hasCondition(cr.Status.Conditions, mljobv1alpha1.ConditionSuspended, mljobv1alpha1.ReasonCancelRequested) {
		updates["status"] = string(StatusCancelled)
		updates["finished_at"] = time.Now().UTC()
	}

	if len(updates) > 0 {
		_ = i.repo.Update(ctx, id, updates)
	}
}

func (i *Informer) onDelete(ctx context.Context, obj any) {
	cr, ok := obj.(*mljobv1alpha1.MLJob)
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
	switch Status(j.Status) {
	case StatusDeleting:
		now := time.Now().UTC()
		_ = i.repo.Update(ctx, id, map[string]any{"status": string(StatusDeleted), "deleted_at": now})
	case StatusPending, StatusRunning:
		// External delete during run → mark Cancelled per design §5.4.
		_ = i.repo.Update(ctx, id, map[string]any{
			"status":      string(StatusCancelled),
			"finished_at": time.Now().UTC(),
			"message":     "external delete",
		})
	case StatusCanceling:
		// Treat DELETE in Canceling as the natural completion of cancel.
		_ = i.repo.Update(ctx, id, map[string]any{
			"status":      string(StatusCancelled),
			"finished_at": time.Now().UTC(),
		})
	}
}

func jobIDFromLabels(cr *mljobv1alpha1.MLJob) (uuid.UUID, error) {
	if v, ok := cr.Labels[mljobv1alpha1.LabelJobID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

func terminalTime(cr *mljobv1alpha1.MLJob) time.Time {
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

var errMissingID = sentinel("missing axisml.io/job-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
