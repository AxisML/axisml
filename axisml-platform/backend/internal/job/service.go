// Package job implements the Jobs tag: name-level Job definitions (Platform PG)
// and their Runs (compute MLRuns) associated live by the compute.axisml.io/job label
// (backend.md §4.2). Run orchestration is shared with Experiments via rundef.
package job

import (
	"context"
	"errors"
	"net/http"

	"gorm.io/gorm"

	"github.com/axisml/axisml/axisml-platform/backend/internal/auth"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/guard"
	"github.com/axisml/axisml/axisml-platform/backend/internal/rundef"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// Service holds Job definition logic; Run ops delegate to the shared Runner.
type Service struct {
	defs   *store.DefinitionRepo
	runner *rundef.Runner
}

// NewService constructs a Job Service (defs bound to the jobs table).
func NewService(defs *store.DefinitionRepo, tenants *store.TenantRepo, compute *computeservice.Client) *Service {
	return &Service{defs: defs, runner: rundef.NewRunner(tenants, compute, LabelJob)}
}

// ---- Job definitions ----

// Create writes a Job definition.
func (s *Service) Create(ctx context.Context, tenant, owner string, in server.JobCreateRequest) (*server.Job, error) {
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
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.Job, error) {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := toView(d)
	return &v, nil
}

// List returns Job definitions visible to the caller.
func (s *Service) List(ctx context.Context, tenants []string, owner, q string, limit, offset int) ([]server.Job, error) {
	defs, err := s.defs.List(ctx, tenants, owner, q, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "list jobs", err)
	}
	out := make([]server.Job, 0, len(defs))
	for i := range defs {
		out = append(out, toView(&defs[i]))
	}
	return out, nil
}

// Update edits a Job template / metadata (affects only later Runs).
func (s *Service) Update(ctx context.Context, id *auth.Identity, tenant, name string, in server.JobPatchRequest) (*server.Job, error) {
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
	active, err := s.runner.HasActiveRuns(ctx, tenant, name)
	if err != nil {
		return err
	}
	if active {
		return conflict("job-has-active-runs", "job has active runs")
	}
	s.runner.CascadeDeleteRuns(ctx, tenant, name)
	if err := s.defs.SoftDelete(ctx, tenant, name); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "delete job", err)
	}
	return nil
}

// ---- Runs (delegated to the shared Runner) ----

// TriggerRun snapshots the Job spec ⊕ overrides into a new MLRun.
func (s *Service) TriggerRun(ctx context.Context, tenant, name, displayName string, ov *server.RunTriggerRequest) (*server.Run, error) {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	var spec server.JobSpec
	if err := jsonUnmarshalSpec(d.Spec, &spec); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "decode job spec", err)
	}
	return s.runner.Trigger(ctx, tenant, name, displayName, spec, ov)
}

func (s *Service) ListRuns(ctx context.Context, tenant, name, phase string) ([]server.Run, error) {
	return s.runner.List(ctx, tenant, name, phase)
}
func (s *Service) GetRun(ctx context.Context, tenant, name, run string) (*server.Run, error) {
	return s.runner.Get(ctx, tenant, name, run)
}
func (s *Service) CancelRun(ctx context.Context, tenant, name, run string) (*server.Run, error) {
	return s.runner.Cancel(ctx, tenant, name, run)
}
func (s *Service) DeleteRun(ctx context.Context, tenant, run string) error {
	return s.runner.Delete(ctx, tenant, run)
}
func (s *Service) RunPods(ctx context.Context, tenant, run string) ([]computeservice.Pod, error) {
	return s.runner.Pods(ctx, tenant, run)
}
func (s *Service) RunEvents(ctx context.Context, tenant, run string) ([]computeservice.Event, error) {
	return s.runner.Events(ctx, tenant, run)
}
func (s *Service) RunPodEvents(ctx context.Context, tenant, run, pod string) ([]computeservice.Event, error) {
	return s.runner.PodEvents(ctx, tenant, run, pod)
}
func (s *Service) RunPodLogs(ctx context.Context, tenant, run, pod string, opt computeservice.LogOptions) (*http.Response, error) {
	return s.runner.PodLogs(ctx, tenant, run, pod, opt)
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

func conflict(reason, msg string) error {
	return apperrors.New(apperrors.ClassConflict, msg).WithReason(reason)
}
