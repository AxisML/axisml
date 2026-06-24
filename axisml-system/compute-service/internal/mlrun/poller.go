package mlrun

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/components/compute-service/pkg/computeruntime"
)

// StatusPoller is the runtime-Observe status reflow for forms without an
// apiserver informer (Lite, design §4.2 / §9.1). On each tick it observes every
// non-terminal MLRun through the ComputeRuntime and reflects the result onto PG
// via the shared writeback helpers — the same mapping the Kubernetes informer
// uses, so the two forms converge identically.
type StatusPoller struct {
	repo     *Repository
	runtime  computeruntime.ComputeRuntime
	log      logr.Logger
	interval time.Duration
}

// NewStatusPoller constructs the job status poller.
func NewStatusPoller(db *gorm.DB, rt computeruntime.ComputeRuntime, log logr.Logger, interval time.Duration) *StatusPoller {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &StatusPoller{repo: NewRepository(db), runtime: rt, log: log, interval: interval}
}

// NeedLeaderElection mirrors the informer; harmless under Lite's single replica.
func (p *StatusPoller) NeedLeaderElection() bool { return true }

func (p *StatusPoller) Start(ctx context.Context) error {
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			p.runOnce(ctx)
		}
	}
}

func (p *StatusPoller) runOnce(ctx context.Context) {
	rows, err := p.repo.FindObservable(ctx)
	if err != nil {
		p.log.Error(err, "find observable runs")
		return
	}
	now := time.Now().UTC()
	for i := range rows {
		row := &rows[i]
		key := types.NamespacedName{Namespace: row.Namespace, Name: row.Name}
		observed, err := p.runtime.ObserveMLRun(ctx, key)
		if apierrors.IsNotFound(err) {
			reflectGone(ctx, p.repo, row)
			continue
		}
		if err != nil {
			p.log.Error(err, "observe MLRun", "name", row.Name)
			continue
		}
		reflectObserved(ctx, p.repo, row, observed, now)
	}
}
