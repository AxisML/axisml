// Package job implements the Jobs tag: name-level Job definitions (Platform PG)
// and their Runs (compute MLRuns) associated live by the axisml.io/job label
// (backend.md §4.2). Experiments reuse the same Run machinery (runutil).
package job

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/axisml/axisml/components/platform/internal/auth"
	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/guard"
	"github.com/axisml/axisml/components/platform/internal/runutil"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
	apperrors "github.com/axisml/axisml/components/platform/pkg/errors"
)

// activePhases are the Run phases that block Job deletion.
var activePhases = map[string]bool{
	"Creating": true, "Pending": true, "Running": true, "Canceling": true,
}

// Service holds Job definition + Run orchestration logic.
type Service struct {
	defs    *store.DefinitionRepo
	tenants *store.TenantRepo
	compute *computeservice.Client
}

// NewService constructs a Job Service (defs bound to the jobs table).
func NewService(defs *store.DefinitionRepo, tenants *store.TenantRepo, compute *computeservice.Client) *Service {
	return &Service{defs: defs, tenants: tenants, compute: compute}
}

// ---- Job definitions ----

// Create writes a Job definition.
func (s *Service) Create(ctx context.Context, tenant, owner string, in server.JobCreateInput) (*server.JobView, error) {
	d := &store.Definition{
		TenantName:  tenant,
		Name:        in.Name,
		DisplayName: in.DisplayName,
		Description: in.Description,
		OwnerUser:   owner,
		Labels:      store.StrMap(in.Labels),
		Annotations: store.StrMap(in.Annotations),
		Spec:        marshalSpec(in.Spec),
	}
	if err := s.defs.Create(ctx, d); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, conflict("job-exists", "job already exists")
		}
		return nil, apperrors.Wrap(apperrors.ClassInternal, "create job", err)
	}
	v := toView(d)
	return &v, nil
}

// Get returns a Job definition.
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.JobView, error) {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := toView(d)
	return &v, nil
}

// List returns Job definitions visible to the caller.
func (s *Service) List(ctx context.Context, tenants []string, owner, q string, limit, offset int) ([]server.JobView, error) {
	defs, err := s.defs.List(ctx, tenants, owner, q, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "list jobs", err)
	}
	out := make([]server.JobView, 0, len(defs))
	for i := range defs {
		out = append(out, toView(&defs[i]))
	}
	return out, nil
}

// Update edits a Job template / metadata (affects only later Runs).
func (s *Service) Update(ctx context.Context, id *auth.Identity, tenant, name string, in server.JobPatchInput) (*server.JobView, error) {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	if err := guard.OwnerOrTenantAdmin(id, tenant, d.OwnerUser); err != nil {
		return nil, err
	}
	if in.DisplayName != "" {
		d.DisplayName = in.DisplayName
	}
	if in.Description != "" {
		d.Description = in.Description
	}
	if in.Labels != nil {
		d.Labels = store.StrMap(in.Labels)
	}
	if in.Annotations != nil {
		d.Annotations = store.StrMap(in.Annotations)
	}
	if len(in.Spec.Roles) > 0 {
		d.Spec = marshalSpec(in.Spec)
	}
	if err := s.defs.Update(ctx, d); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "update job", err)
	}
	v := toView(d)
	return &v, nil
}

// Delete removes a Job: blocked when it has active Runs, else cascade soft-delete
// of terminal Runs then the definition (best-effort).
func (s *Service) Delete(ctx context.Context, id *auth.Identity, tenant, name string) error {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return err
	}
	if err := guard.OwnerOrTenantAdmin(id, tenant, d.OwnerUser); err != nil {
		return err
	}
	runs, err := s.compute.ListMLRuns(ctx, tenant, LabelJob+"="+name)
	if err != nil {
		return err
	}
	for i := range runs {
		if activePhases[runs[i].Phase] {
			return conflict("job-has-active-runs", "job has active runs")
		}
	}
	for i := range runs {
		_ = s.compute.DeleteMLRun(ctx, tenant, runs[i].Name)
	}
	if err := s.defs.SoftDelete(ctx, tenant, name); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "delete job", err)
	}
	return nil
}

// ---- Runs ----

// TriggerRun snapshots the Job spec ⊕ overrides into a new MLRun named <job>-<n>.
func (s *Service) TriggerRun(ctx context.Context, tenant, name, displayName string, ov *server.RunTriggerInput) (*server.RunView, error) {
	if err := guard.TenantActive(ctx, s.tenants, tenant); err != nil {
		return nil, err
	}
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	var spec server.JobSpec
	if err := jsonUnmarshalSpec(d.Spec, &spec); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "decode job spec", err)
	}

	// Bounded retry on run-number collisions.
	for attempt := 0; attempt < 5; attempt++ {
		n, err := s.nextRunNumber(ctx, tenant, name)
		if err != nil {
			return nil, err
		}
		runName := fmt.Sprintf("%s-%d", name, n)
		labels := map[string]string{LabelJob: name}
		for k, v := range ovLabels(ov) {
			labels[k] = v
		}
		input, err := runutil.BuildRunInput(spec, ov, runName, displayName, labels, ovAnnotations(ov))
		if err != nil {
			return nil, apperrors.Wrap(apperrors.ClassInternal, "build run", err)
		}
		run, err := s.compute.CreateMLRun(ctx, tenant, input)
		if err != nil {
			if e, ok := apperrors.As(err); ok && e.Class == apperrors.ClassConflict {
				continue // name raced; recompute n
			}
			return nil, err
		}
		v := runutil.RunToView(run, tenant, name)
		return &v, nil
	}
	return nil, conflict("run-name-conflict", "could not allocate a run name")
}

// ListRuns lists a Job's Runs (live), optionally filtered by phase.
func (s *Service) ListRuns(ctx context.Context, tenant, name, phase string) ([]server.RunView, error) {
	runs, err := s.compute.ListMLRuns(ctx, tenant, LabelJob+"="+name)
	if err != nil {
		return nil, err
	}
	out := make([]server.RunView, 0, len(runs))
	for i := range runs {
		if phase != "" && runs[i].Phase != phase {
			continue
		}
		out = append(out, runutil.RunToView(&runs[i], tenant, name))
	}
	return out, nil
}

// GetRun returns one Run.
func (s *Service) GetRun(ctx context.Context, tenant, name, run string) (*server.RunView, error) {
	r, err := s.compute.GetMLRun(ctx, tenant, run)
	if err != nil {
		return nil, err
	}
	v := runutil.RunToView(r, tenant, name)
	return &v, nil
}

// CancelRun cancels a Run.
func (s *Service) CancelRun(ctx context.Context, tenant, name, run string) (*server.RunView, error) {
	r, err := s.compute.CancelMLRun(ctx, tenant, run)
	if err != nil {
		return nil, err
	}
	v := runutil.RunToView(r, tenant, name)
	return &v, nil
}

// DeleteRun deletes a Run.
func (s *Service) DeleteRun(ctx context.Context, tenant, run string) error {
	return s.compute.DeleteMLRun(ctx, tenant, run)
}

// RunPods / RunEvents / RunPodEvents / RunPodLogs proxy compute.
func (s *Service) RunPods(ctx context.Context, tenant, run string) ([]computeservice.Pod, error) {
	return s.compute.ListMLRunPods(ctx, tenant, run)
}
func (s *Service) RunEvents(ctx context.Context, tenant, run string) ([]computeservice.Event, error) {
	return s.compute.ListMLRunEvents(ctx, tenant, run)
}
func (s *Service) RunPodEvents(ctx context.Context, tenant, run, pod string) ([]computeservice.Event, error) {
	return s.compute.ListMLRunPodEvents(ctx, tenant, run, pod)
}
func (s *Service) RunPodLogs(ctx context.Context, tenant, run, pod string) ([]byte, error) {
	return s.compute.GetMLRunPodLogs(ctx, tenant, run, pod)
}

// ---- helpers ----

func (s *Service) get(ctx context.Context, tenant, name string) (*store.Definition, error) {
	d, err := s.defs.GetByName(ctx, tenant, name)
	if errors.Is(err, store.ErrNotFound) {
		return nil, server.NotFound("job not found")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup job", err)
	}
	return d, nil
}

func (s *Service) nextRunNumber(ctx context.Context, tenant, name string) (int, error) {
	runs, err := s.compute.ListMLRuns(ctx, tenant, LabelJob+"="+name)
	if err != nil {
		return 0, err
	}
	max := 0
	prefix := name + "-"
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

func conflict(reason, msg string) error {
	return apperrors.New(apperrors.ClassConflict, msg).WithReason(reason)
}
