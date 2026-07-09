package mlrun

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/auth"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/resource"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/strutil"
)

// Service is the job business layer. Keyed on bare namespace strings;
// `spec.scheduling.quota` is whatever the caller passes through.
type Service struct {
	repo  *Repository
	db    *gorm.DB
	pools extensions.ResourceResolver
}

// NewService constructs the job service.
func NewService(db *gorm.DB, pools extensions.ResourceResolver) *Service {
	return &Service{
		repo:  NewRepository(db),
		db:    db,
		pools: pools,
	}
}

// Create writes a new job row. CR is reconciled asynchronously.
func (s *Service) Create(ctx context.Context, namespace string, in server.MLRunCreateRequest) (*server.MLRun, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid job name")
	}
	if in.Quota == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "quota is required")
	}
	if existing, err := s.repo.GetByNamespaceName(ctx, namespace, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "job already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	pool, err := s.pools.ResolveResourcePool(ctx, in.PoolName)
	if err != nil {
		return nil, err
	}
	unit, err := s.pools.ResolveResourceUnit(ctx, in.PoolName, in.UnitName)
	if err != nil {
		return nil, err
	}
	expanded := resource.Expand(pool, unit)

	backend := mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"}
	if in.Backend != nil {
		if in.Backend.Name != "" {
			backend.Name = in.Backend.Name
		}
		if in.Backend.Engine != "" {
			backend.Engine = in.Backend.Engine
		}
		backend.Config = in.Backend.Config
	}

	roles := make([]mlrunv1alpha1.RoleSpec, len(in.Roles))
	for i, r := range in.Roles {
		role := r
		role.Template.Resources = resource.BuildResources(expanded.Requests, expanded.Limits)
		roles[i] = role
	}

	runPolicy := mlrunv1alpha1.RunPolicySpec{}
	if in.RunPolicy != nil {
		if in.RunPolicy.Suspend {
			return nil, apperrors.New(apperrors.CodeValidation,
				"runPolicy.suspend=true is not allowed on Create; use POST /mlruns/{name}/cancel after submission")
		}
		runPolicy = *in.RunPolicy
	}

	spec := mlrunv1alpha1.MLRunSpec{
		Backend: backend,
		Scheduling: mlrunv1alpha1.SchedulingSpec{
			Quota:         in.Quota,
			PriorityClass: in.PriorityClass,
			NodeSelector:  expanded.NodeSelector,
			Tolerations:   expanded.Tolerations,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	// Mirror the (poolName, unitName) provenance into PG labels alongside
	// any user-supplied entries — Platform's pre-delete check against
	// active workloads uses labelSelector against resource.axisml.io/pool
	// / resource.axisml.io/unit (compute-service.md §5.4).
	mergedLabels := mergeLabels(in.Labels, map[string]string{
		mlrunv1alpha1.LabelResourcePool: in.PoolName,
		mlrunv1alpha1.LabelResourceUnit: in.UnitName,
	})
	j := &store.MLRun{
		ID:          uuid.New(),
		Namespace:   namespace,
		Name:        in.Name,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Owner:       auth.User(ctx),
		Labels:      mapBytes(mergedLabels),
		Annotations: mapBytes(in.Annotations),
		Spec:        datatypes.JSON(specJSON),
		Phase:       string(StatusCreating),
		StatusJSON:  []byte("{}"),
	}
	if err := s.repo.Create(ctx, j); err != nil {
		return nil, err
	}
	return s.toView(j)
}

func (s *Service) Get(ctx context.Context, namespace, name string) (*server.MLRun, error) {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	return s.toView(j)
}

// Phase returns the run's current lifecycle phase and status detail — a
// lightweight projection for high-frequency polling that skips unmarshalling
// and shipping the spec sub-tree.
func (s *Service) Phase(ctx context.Context, namespace, name string) (*server.MLRunPhase, error) {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	v := phaseView(j)
	return &v, nil
}

// PhasesByNames returns phase projections for the named runs in the namespace.
// Names that don't resolve (deleted / never existed) are simply omitted, so a
// caller diffs the returned set against what it asked for.
func (s *Service) PhasesByNames(ctx context.Context, namespace string, names []string) ([]server.MLRunPhase, error) {
	rows, err := s.repo.ListPhasesByNames(ctx, namespace, names)
	if err != nil {
		return nil, err
	}
	return phaseViews(rows), nil
}

// PhasesBySelector returns phase projections for runs matching the label
// selector, paginated like List.
func (s *Service) PhasesBySelector(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]server.MLRunPhase, int64, error) {
	rows, total, err := s.repo.ListPhasesBySelector(ctx, namespace, limit, offset, labelClause, labelArgs)
	if err != nil {
		return nil, 0, err
	}
	return phaseViews(rows), total, nil
}

// phaseView projects a row onto the lean phase DTO, unmarshalling only the
// status sub-tree (never the spec).
func phaseView(j *store.MLRun) server.MLRunPhase {
	var status server.MLRunStatus
	if len(j.StatusJSON) > 0 {
		_ = json.Unmarshal(j.StatusJSON, &status)
	}
	return server.MLRunPhase{
		Name:       j.Name,
		Phase:      j.Phase,
		Message:    status.Message,
		StartedAt:  status.StartedAt,
		FinishedAt: status.FinishedAt,
	}
}

func phaseViews(rows []store.MLRun) []server.MLRunPhase {
	out := make([]server.MLRunPhase, 0, len(rows))
	for i := range rows {
		out = append(out, phaseView(&rows[i]))
	}
	return out
}

func (s *Service) List(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]server.MLRun, int64, error) {
	rows, total, err := s.repo.ListByNamespace(ctx, namespace, limit, offset, labelClause, labelArgs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]server.MLRun, 0, len(rows))
	for i := range rows {
		v, err := s.toView(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (s *Service) Cancel(ctx context.Context, namespace, name string) (*server.MLRun, error) {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	switch Status(j.Phase) {
	case StatusCreating:
		return nil, apperrors.New(apperrors.CodePrecondition, "job is still being created; use DELETE")
	case StatusCanceling, StatusCancelled, StatusDeleting, StatusDeleted, StatusSucceeded, StatusFailed:
		return nil, apperrors.New(apperrors.CodePrecondition, "job is not cancellable in current state")
	}
	// Merge "user cancelled" into status.message; preserve other status fields.
	var sf server.MLRunStatus
	if len(j.StatusJSON) > 0 {
		_ = json.Unmarshal(j.StatusJSON, &sf)
	}
	sf.Message = "user cancelled"
	statusB, _ := json.Marshal(sf)
	if err := s.repo.Update(ctx, j.ID, map[string]any{
		"phase":  string(StatusCanceling),
		"status": datatypes.JSON(statusB),
	}); err != nil {
		return nil, err
	}
	j.Phase = string(StatusCanceling)
	j.StatusJSON = statusB
	j.UpdatedAt = time.Now().UTC()
	return s.toView(j)
}

// Patch updates the row's display-tier metadata. Pure PG mutation —
// no CR is touched, no generation bump (compute-service.md §4.3).
func (s *Service) Patch(ctx context.Context, namespace, name string, in server.MLRunPatchRequest) (*server.MLRun, error) {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "job not found")
		}
		return nil, err
	}
	updates := map[string]any{}
	if in.DisplayName != nil {
		updates["display_name"] = *in.DisplayName
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.Labels != nil {
		updates["labels"] = mapBytes(in.Labels)
	}
	if in.Annotations != nil {
		updates["annotations"] = mapBytes(in.Annotations)
	}
	if len(updates) == 0 {
		return s.toView(j)
	}
	if err := s.repo.Update(ctx, j.ID, updates); err != nil {
		return nil, err
	}
	fresh, err := s.repo.Get(ctx, j.ID)
	if err != nil {
		return nil, err
	}
	return s.toView(fresh)
}

func (s *Service) Delete(ctx context.Context, namespace, name string) error {
	j, err := s.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	switch Status(j.Phase) {
	case StatusDeleting, StatusDeleted:
		return nil
	}
	return s.repo.MarkDeleting(ctx, j.ID)
}

// mergeLabels combines user-supplied labels with system provenance labels.
// System keys (pool/unit) win over user-supplied entries with the same key
// so callers can't shadow the provenance pointer.
func mergeLabels(user, system map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range user {
		out[k] = v
	}
	for k, v := range system {
		if v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func mapBytes(m map[string]string) datatypes.JSON {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return b
}

func (s *Service) toView(j *store.MLRun) (*server.MLRun, error) {
	var spec mlrunv1alpha1.MLRunSpec
	if len(j.Spec) > 0 {
		_ = json.Unmarshal(j.Spec, &spec)
	}
	var status server.MLRunStatus
	if len(j.StatusJSON) > 0 {
		_ = json.Unmarshal(j.StatusJSON, &status)
	}
	return &server.MLRun{
		ID:          j.ID,
		Namespace:   j.Namespace,
		Name:        j.Name,
		DisplayName: j.DisplayName,
		Description: j.Description,
		Owner:       j.Owner,
		Labels:      decodeMap(j.Labels),
		Annotations: decodeMap(j.Annotations),
		Phase:       j.Phase,
		Spec:        spec,
		Status:      status,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
		DeletedAt:   j.DeletedAt,
	}, nil
}

func decodeMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	m := map[string]string{}
	_ = json.Unmarshal(raw, &m)
	return m
}
