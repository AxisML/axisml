// Package rundef holds the Run orchestration shared by the Jobs and Experiments
// modules. A Runner is parameterised by the grouping label key
// (compute.axisml.io/job | compute.axisml.io/experiment); both modules layer their own
// definition CRUD + type mapping on top.
package rundef

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/guard"
	"github.com/axisml/axisml/axisml-platform/backend/internal/runutil"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

var activePhases = map[string]bool{
	"Creating": true, "Pending": true, "Running": true, "Canceling": true,
}

// Runner orchestrates Runs (compute MLRuns) for one definition kind.
type Runner struct {
	tenants  *store.TenantRepo
	compute  *computeservice.Client
	labelKey string
}

// NewRunner constructs a Runner for the given grouping label key.
func NewRunner(tenants *store.TenantRepo, compute *computeservice.Client, labelKey string) *Runner {
	return &Runner{tenants: tenants, compute: compute, labelKey: labelKey}
}

func (r *Runner) selector(defName string) string { return r.labelKey + "=" + defName }

// Trigger snapshots spec ⊕ overrides into a new MLRun named <def>-<n> with the
// grouping label. The tenant must be active (suspension gate).
func (r *Runner) Trigger(ctx context.Context, tenant, defName, displayName string, spec server.JobSpec, ov *server.RunTriggerRequest) (*server.Run, error) {
	if err := guard.TenantActive(ctx, r.tenants, tenant); err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 5; attempt++ {
		n, err := r.nextRunNumber(ctx, tenant, defName)
		if err != nil {
			return nil, err
		}
		runName := fmt.Sprintf("%s-%d", defName, n)
		labels := map[string]string{r.labelKey: defName}
		for k, v := range ovLabels(ov) {
			labels[k] = v
		}
		input, err := runutil.BuildRunInput(spec, ov, runName, displayName, labels, ovAnnotations(ov))
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ClassInternal, "build run", err)
		}
		run, err := r.compute.CreateMLRun(ctx, tenant, input)
		if err != nil {
			if e, ok := apperrors.As(err); ok && e.Class == apperrors.ClassConflict {
				continue
			}
			return nil, err
		}
		v := runutil.RunToView(run, tenant, defName)
		return &v, nil
	}
	return nil, apperrors.New(apperrors.ClassConflict, "could not allocate a run name").WithReason("run-name-conflict")
}

// List lists a definition's Runs, optionally filtered by phase.
func (r *Runner) List(ctx context.Context, tenant, defName, phase string) ([]server.Run, error) {
	runs, err := r.compute.ListMLRuns(ctx, tenant, r.selector(defName))
	if err != nil {
		return nil, err
	}
	out := make([]server.Run, 0, len(runs))
	for i := range runs {
		if phase != "" && runs[i].Phase != phase {
			continue
		}
		out = append(out, runutil.RunToView(&runs[i], tenant, defName))
	}
	return out, nil
}

// Get returns one Run.
func (r *Runner) Get(ctx context.Context, tenant, defName, run string) (*server.Run, error) {
	v, err := r.compute.GetMLRun(ctx, tenant, run)
	if err != nil {
		return nil, err
	}
	view := runutil.RunToView(v, tenant, defName)
	return &view, nil
}

// Cancel cancels a Run.
func (r *Runner) Cancel(ctx context.Context, tenant, defName, run string) (*server.Run, error) {
	v, err := r.compute.CancelMLRun(ctx, tenant, run)
	if err != nil {
		return nil, err
	}
	view := runutil.RunToView(v, tenant, defName)
	return &view, nil
}

// Delete deletes a Run.
func (r *Runner) Delete(ctx context.Context, tenant, run string) error {
	return r.compute.DeleteMLRun(ctx, tenant, run)
}

// HasActiveRuns reports whether a definition has any non-terminal Run.
func (r *Runner) HasActiveRuns(ctx context.Context, tenant, defName string) (bool, error) {
	runs, err := r.compute.ListMLRuns(ctx, tenant, r.selector(defName))
	if err != nil {
		return false, err
	}
	for i := range runs {
		if activePhases[runs[i].Phase] {
			return true, nil
		}
	}
	return false, nil
}

// CascadeDeleteRuns best-effort deletes all of a definition's Runs.
func (r *Runner) CascadeDeleteRuns(ctx context.Context, tenant, defName string) {
	runs, err := r.compute.ListMLRuns(ctx, tenant, r.selector(defName))
	if err != nil {
		return
	}
	for i := range runs {
		_ = r.compute.DeleteMLRun(ctx, tenant, runs[i].Name)
	}
}

// Pods / Events / PodEvents / PodLogs proxy compute.
func (r *Runner) Pods(ctx context.Context, tenant, run string) ([]computeservice.Pod, error) {
	return r.compute.ListMLRunPods(ctx, tenant, run)
}

// Metrics returns a resource metric time series for a Run (the run name is the
// backing MLRun name).
func (r *Runner) Metrics(ctx context.Context, tenant, run, metric, rng string, step *string) (*computeservice.MetricSeries, error) {
	return r.compute.MLRunMetrics(ctx, tenant, run, metric, rng, step)
}
func (r *Runner) Events(ctx context.Context, tenant, run string) ([]computeservice.Event, error) {
	return r.compute.ListMLRunEvents(ctx, tenant, run)
}
func (r *Runner) PodEvents(ctx context.Context, tenant, run, pod string) ([]computeservice.Event, error) {
	return r.compute.ListMLRunPodEvents(ctx, tenant, run, pod)
}
func (r *Runner) PodLogs(ctx context.Context, tenant, run, pod string, opt computeservice.LogOptions) (*http.Response, error) {
	return r.compute.StreamMLRunPodLogs(ctx, tenant, run, pod, opt)
}

func (r *Runner) nextRunNumber(ctx context.Context, tenant, defName string) (int, error) {
	runs, err := r.compute.ListMLRuns(ctx, tenant, r.selector(defName))
	if err != nil {
		return 0, err
	}
	max := 0
	prefix := defName + "-"
	for i := range runs {
		if !strings.HasPrefix(runs[i].Name, prefix) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(runs[i].Name, prefix)); err == nil && n > max {
			max = n
		}
	}
	return max + 1, nil
}

func ovLabels(ov *server.RunTriggerRequest) map[string]string {
	if ov == nil {
		return nil
	}
	return ov.Labels
}

func ovAnnotations(ov *server.RunTriggerRequest) map[string]string {
	if ov == nil {
		return nil
	}
	return ov.Annotations
}
