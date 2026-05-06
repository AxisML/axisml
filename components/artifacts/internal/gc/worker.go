// Package gc implements the artifacts GC worker. MVP scope (design §8.2)
// covers two predicates only:
//
//   - artifacts WHERE status='Uploading' AND created_at < now - UploadingTTL
//     → mark Failed, then GCBackend
//   - artifacts WHERE status='Deleting'
//     → GCBackend, then mark Deleted
//
// Repos in Deleting cascade-fanout to their child artifacts via a single
// UPDATE before each tick, and empty-of-live-artifacts Deleting repos are
// finalized via one more set-based UPDATE at the end of each tick.
//
// Phase 2 will add the Failed > 30d → Deleting predicate.
package gc

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/gorm"

	artmod "github.com/axisml/axisml/components/artifacts/internal/artifact"
	"github.com/axisml/axisml/components/artifacts/internal/artifact/handler"
	"github.com/axisml/axisml/components/artifacts/internal/config"
	"github.com/axisml/axisml/components/artifacts/internal/metrics"
)

// Clock is a minimal abstraction so tests can fast-forward.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// Worker is the leader-elected GC ticker. Implements
// sigs.k8s.io/controller-runtime/pkg/manager.Runnable +
// LeaderElectionRunnable.
type Worker struct {
	cfg   config.Config
	rows  *artmod.Repository
	log   logr.Logger
	clock Clock
}

// New returns a Worker.
func New(cfg config.Config, db *gorm.DB, log logr.Logger) *Worker {
	return &Worker{
		cfg:   cfg,
		rows:  artmod.NewRepository(db),
		log:   log,
		clock: realClock{},
	}
}

// SetClock replaces the clock; tests use it to fast-forward.
func (w *Worker) SetClock(c Clock) { w.clock = c }

// NeedLeaderElection makes the worker leader-only.
func (w *Worker) NeedLeaderElection() bool { return true }

// Start runs the GC loop until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	w.log.Info("gc worker started", "interval", w.cfg.GCInterval)
	metrics.IsLeader.Set(1)
	defer metrics.IsLeader.Set(0)

	t := time.NewTicker(w.cfg.GCInterval)
	defer t.Stop()

	for {
		// Run one pass immediately so manual integration tests don't need to
		// wait for the first tick.
		w.Tick(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// Tick performs one full GC pass. Exported so integration tests can drive
// it directly without waiting for the ticker.
func (w *Worker) Tick(ctx context.Context) {
	w.cascadeRepos(ctx)
	w.processStaleUploading(ctx)
	w.processDeleting(ctx)
	w.finalizeRepos(ctx)
	w.refreshGauges(ctx)
}

// cascadeRepos pushes Deleting-repos' children into Deleting.
func (w *Worker) cascadeRepos(ctx context.Context) {
	n, err := w.rows.CascadeFromDeletingRepos(ctx, nil)
	if err != nil {
		w.log.Error(err, "cascade from deleting repos")
		metrics.GCActions.WithLabelValues("cascade_repo", "error").Inc()
		return
	}
	if n > 0 {
		w.log.Info("cascaded artifacts into Deleting", "count", n)
		metrics.GCActions.WithLabelValues("cascade_repo", "ok").Add(float64(n))
	}
}

// processStaleUploading flips stuck Uploading rows to Failed and cleans up
// any partially uploaded backend state.
func (w *Worker) processStaleUploading(ctx context.Context) {
	cutoff := w.clock.Now().Add(-w.cfg.UploadingTTL)
	stale, err := w.rows.FindStaleUploading(ctx, cutoff)
	if err != nil {
		w.log.Error(err, "find stale uploading")
		metrics.GCActions.WithLabelValues("uploading_ttl", "error").Inc()
		return
	}
	for _, row := range stale {
		if h, ok := handler.Get(row.Kind); ok {
			if err := h.GCBackend(ctx, gcArtifact(row)); err != nil {
				w.log.Error(err, "gc backend (uploading_ttl)", "artifactID", row.ID)
				// continue: prefer flipping Failed even if backend is flaky
			}
		}
		if err := w.rows.Update(ctx, nil, row.ID, map[string]any{
			"status":  artmod.StatusFailed,
			"message": "upload TTL exceeded; cleaned up by GC",
		}); err != nil {
			w.log.Error(err, "mark failed", "artifactID", row.ID)
			metrics.GCActions.WithLabelValues("uploading_ttl", "error").Inc()
			continue
		}
		metrics.GCActions.WithLabelValues("uploading_ttl", "ok").Inc()
	}
}

// processDeleting cleans up backend state for Deleting rows and finalizes
// them as Deleted. Parent repo finalization happens once at the end of the
// tick via a set-based UPDATE in finalizeRepos.
func (w *Worker) processDeleting(ctx context.Context) {
	rows, err := w.rows.FindDeleting(ctx)
	if err != nil {
		w.log.Error(err, "find deleting")
		metrics.GCActions.WithLabelValues("deleting", "error").Inc()
		return
	}
	now := w.clock.Now()
	for _, row := range rows {
		if h, ok := handler.Get(row.Kind); ok {
			if err := h.GCBackend(ctx, gcArtifact(row)); err != nil {
				w.log.Error(err, "gc backend (deleting)", "artifactID", row.ID)
				metrics.GCActions.WithLabelValues("deleting", "error").Inc()
				continue
			}
		}
		if err := w.rows.Update(ctx, nil, row.ID, map[string]any{
			"status":     artmod.StatusDeleted,
			"deleted_at": now,
		}); err != nil {
			w.log.Error(err, "mark deleted", "artifactID", row.ID)
			metrics.GCActions.WithLabelValues("deleting", "error").Inc()
			continue
		}
		metrics.GCActions.WithLabelValues("deleting", "ok").Inc()
	}
}

// finalizeRepos flips Deleting repos with no remaining live artifacts to
// Deleted in a single set-based UPDATE.
func (w *Worker) finalizeRepos(ctx context.Context) {
	n, err := w.rows.FinalizeEmptyDeletingRepos(ctx, nil)
	if err != nil {
		w.log.Error(err, "finalize empty deleting repos")
		metrics.GCActions.WithLabelValues("repo_finalize", "error").Inc()
		return
	}
	if n > 0 {
		metrics.GCActions.WithLabelValues("repo_finalize", "ok").Add(float64(n))
	}
}

// refreshGauges updates the uploading_count gauge from PG state.
func (w *Worker) refreshGauges(ctx context.Context) {
	counts, err := w.rows.CountUploadingByKind(ctx)
	if err != nil {
		w.log.Error(err, "count uploading by kind")
		return
	}
	for _, kind := range handler.Kinds() {
		metrics.UploadingCount.WithLabelValues(kind).Set(float64(counts[kind]))
	}
}

// gcArtifact maps a joined GC row to the handler.Artifact view. Spec is
// not populated here — model.GCBackend (the only Kind in MVP) only reads
// Scope/RepoName/Version/Digest, so unmarshalling spec on every tick would
// be wasted work.
func gcArtifact(row artmod.GCRow) handler.Artifact {
	return handler.Artifact{
		Kind:     row.Kind,
		Scope:    row.Scope(),
		RepoName: row.RepoName,
		Version:  row.Version,
		Digest:   row.Digest,
	}
}
