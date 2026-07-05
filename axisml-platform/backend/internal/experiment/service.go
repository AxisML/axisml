// Package experiment implements the Experiments tag: training-specialized Job
// definitions whose Runs carry compute.axisml.io/experiment, plus on-demand TensorBoard
// (backend.md §4.9–4.10). Run orchestration is shared via rundef; the spec is
// isomorphic to a Job's.
package experiment

import (
	"context"
	"encoding/json"
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

// Service holds Experiment definition + TensorBoard logic; Runs delegate to the
// shared Runner.
type Service struct {
	defs    *store.DefinitionRepo
	runner  *rundef.Runner
	compute *computeservice.Client
	tenants *store.TenantRepo
}

// NewService constructs an Experiment Service (defs bound to the experiments table).
func NewService(defs *store.DefinitionRepo, tenants *store.TenantRepo, compute *computeservice.Client) *Service {
	return &Service{
		defs:    defs,
		runner:  rundef.NewRunner(tenants, compute, LabelExperiment),
		compute: compute,
		tenants: tenants,
	}
}

// ---- Experiment definitions ----

// Create writes an Experiment definition.
func (s *Service) Create(ctx context.Context, tenant, owner string, in server.ExperimentCreateRequest) (*server.Experiment, error) {
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
			return nil, conflict("experiment-exists", "experiment already exists")
		}
		return nil, apperrors.Wrap(apperrors.ClassInternal, "create experiment", err)
	}
	v := toView(d)
	return &v, nil
}

// Get returns an Experiment definition.
func (s *Service) Get(ctx context.Context, tenant, name string) (*server.Experiment, error) {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	v := toView(d)
	return &v, nil
}

// List returns Experiment definitions visible to the caller.
func (s *Service) List(ctx context.Context, tenants []string, owner, q string, limit, offset int) ([]server.Experiment, error) {
	defs, err := s.defs.List(ctx, tenants, owner, q, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "list experiments", err)
	}
	out := make([]server.Experiment, 0, len(defs))
	for i := range defs {
		out = append(out, toView(&defs[i]))
	}
	return out, nil
}

// Update edits an Experiment template / metadata.
func (s *Service) Update(ctx context.Context, id *auth.Identity, tenant, name string, in server.ExperimentPatchRequest) (*server.Experiment, error) {
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
		return nil, apperrors.Wrap(apperrors.ClassInternal, "update experiment", err)
	}
	v := toView(d)
	return &v, nil
}

// Delete removes an Experiment (blocked on active Runs, else cascade soft-delete).
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
		return conflict("experiment-has-active-runs", "experiment has active runs")
	}
	s.runner.CascadeDeleteRuns(ctx, tenant, name)
	if err := s.defs.SoftDelete(ctx, tenant, name); err != nil {
		return apperrors.Wrap(apperrors.ClassInternal, "delete experiment", err)
	}
	return nil
}

// ---- Runs (delegated) ----

// TriggerRun snapshots the Experiment spec ⊕ overrides into a new MLRun.
func (s *Service) TriggerRun(ctx context.Context, tenant, name, displayName string, ov *server.RunTriggerRequest) (*server.Run, error) {
	d, err := s.get(ctx, tenant, name)
	if err != nil {
		return nil, err
	}
	var spec server.JobSpec
	if err := jsonUnmarshalSpec(d.Spec, &spec); err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "decode experiment spec", err)
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
func (s *Service) RunMetrics(ctx context.Context, tenant, run, metric, rng string, step *string) (*computeservice.MetricSeries, error) {
	return s.runner.Metrics(ctx, tenant, run, metric, rng, step)
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

// ---- TensorBoard (§4.10) ----

func tbName(exp string) string {
	n := "tb-" + exp
	if len(n) > 40 {
		n = n[:40]
	}
	return n
}

// GetTensorBoard returns the experiment's TensorBoard instance (404 if none).
func (s *Service) GetTensorBoard(ctx context.Context, tenant, exp string) (*server.TensorBoard, error) {
	if _, err := s.get(ctx, tenant, exp); err != nil {
		return nil, err
	}
	svc, err := s.compute.GetMLService(ctx, tenant, tbName(exp))
	if err != nil {
		return nil, err
	}
	v := tbToView(svc)
	return &v, nil
}

// StartTensorBoard starts (or reuses) a TensorBoard for the experiment. The
// logdir/object-store wiring is rendered compute-side (kind=tensorboard); this
// path only requests the instance.
func (s *Service) StartTensorBoard(ctx context.Context, tenant, exp string, runs []string) (*server.TensorBoard, error) {
	if err := guard.TenantActive(ctx, s.tenants, tenant); err != nil {
		return nil, err
	}
	d, err := s.get(ctx, tenant, exp)
	if err != nil {
		return nil, err
	}
	// Idempotent: reuse a live instance.
	if svc, err := s.compute.GetMLService(ctx, tenant, tbName(exp)); err == nil {
		v := tbToView(svc)
		return &v, nil
	}
	var spec server.JobSpec
	_ = jsonUnmarshalSpec(d.Spec, &spec)
	input, err := buildTensorBoardInput(tbName(exp), exp, spec, runs)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "build tensorboard", err)
	}
	svc, err := s.compute.CreateMLService(ctx, tenant, input)
	if err != nil {
		return nil, err
	}
	v := tbToView(svc)
	return &v, nil
}

// StopTensorBoard tears down the experiment's TensorBoard.
func (s *Service) StopTensorBoard(ctx context.Context, tenant, exp string) error {
	if _, err := s.get(ctx, tenant, exp); err != nil {
		return err
	}
	return s.compute.DeleteMLService(ctx, tenant, tbName(exp))
}

// buildTensorBoardInput assembles a kind=tensorboard MLService request. The
// experiment + selected runs are passed as annotations for compute to resolve
// the object-store logdir prefix; the route stays disabled (no external access
// until the SecurityPolicy lands).
func buildTensorBoardInput(name, exp string, spec server.JobSpec, runs []string) (computeservice.MLServiceCreate, error) {
	annos := map[string]any{"compute.axisml.io/experiment": exp}
	if len(runs) > 0 {
		annos["platform.axisml.io/tensorboard-runs"] = runs
	}
	input := map[string]any{
		"name":        name,
		"kind":        "tensorboard",
		"poolName":    spec.PoolName,
		"unitName":    spec.UnitName,
		"quota":       spec.Quota,
		"annotations": annos,
		"roles": []map[string]any{{
			"name":     "tensorboard",
			"replicas": 1,
			"template": map[string]any{"image": "tensorboard"},
		}},
		"route": map[string]any{"enabled": false},
	}
	var out computeservice.MLServiceCreate
	b, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(b, &out)
}

// ---- helpers ----

func (s *Service) get(ctx context.Context, tenant, name string) (*store.Definition, error) {
	d, err := s.defs.GetByName(ctx, tenant, name)
	if errors.Is(err, store.ErrNotFound) {
		return nil, server.NotFound("experiment not found")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ClassInternal, "lookup experiment", err)
	}
	return d, nil
}

func conflict(reason, msg string) error {
	return apperrors.New(apperrors.ClassConflict, msg).WithReason(reason)
}
