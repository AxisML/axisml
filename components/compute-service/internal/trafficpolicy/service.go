package trafficpolicy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/auth"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/service"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/strutil"
)

// Module wraps the traffic-policy business layer. It is the authoritative
// gate for member validation (kind / readiness / occupancy / homogeneous
// backend family) per compute-service.md §4.5.
type Module struct {
	repo    *Repository
	db      *gorm.DB
	members *servicemod.Repository
}

// NewService builds the traffic-policy module. members is the services
// repository, used to validate referenced online services.
func NewService(db *gorm.DB, members *servicemod.Repository) *Module {
	return &Module{repo: NewRepository(db), db: db, members: members}
}

// CreateInput is the API request body. The backend tuple is derived (not
// supplied) from the member services' family.
type CreateInput struct {
	Name        string               `json:"name" binding:"required,axisml_name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description"`
	Labels      map[string]string    `json:"labels,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
	Mode        string               `json:"mode" binding:"required"`
	Endpoint    mltp.Endpoint        `json:"endpoint"`
	Backends    []mltp.BackendMember `json:"backends" binding:"required,min=1"`
}

// SplitInput adjusts per-backend weights. Only listed backends change.
type SplitInput struct {
	Backends []WeightUpdate `json:"backends" binding:"required,min=1"`
}

// WeightUpdate is one (serviceName, weight) pair.
type WeightUpdate struct {
	ServiceName string `json:"serviceName" binding:"required"`
	Weight      int32  `json:"weight"`
}

// PatchInput mutates display-tier metadata only (no CR touch, no generation
// bump).
type PatchInput struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// View is the HTTP response.
type View struct {
	ID                 uuid.UUID                `json:"id"`
	Namespace          string                   `json:"namespace"`
	Name               string                   `json:"name"`
	Mode               string                   `json:"mode"`
	DisplayName        string                   `json:"displayName,omitempty"`
	Description        string                   `json:"description,omitempty"`
	Owner              string                   `json:"owner,omitempty"`
	Labels             map[string]string        `json:"labels,omitempty"`
	Annotations        map[string]string        `json:"annotations,omitempty"`
	Generation         int64                    `json:"generation"`
	ObservedGeneration int64                    `json:"observedGeneration"`
	Phase              string                   `json:"phase"`
	Spec               mltp.MLTrafficPolicySpec `json:"spec"`
	Status             StatusFields             `json:"status"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
	DeletedAt          *time.Time               `json:"deletedAt,omitempty"`
}

func (m *Module) Create(ctx context.Context, namespace string, in CreateInput) (*View, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid traffic-policy name")
	}
	mode := mltp.TrafficMode(in.Mode)
	switch mode {
	case mltp.TrafficModeWeighted, mltp.TrafficModeCanary, mltp.TrafficModeBlueGreen:
	default:
		return nil, apperrors.Newf(apperrors.CodeValidation, "unknown mode %q", in.Mode)
	}
	// Fail closed: SecurityPolicy derivation for the data-plane gateway is not
	// wired yet, so accepting auth=jwt would create an UNAUTHENTICATED endpoint
	// the caller believes is JWT-protected. Reject until the follow-up lands.
	if in.Endpoint.Auth != nil && in.Endpoint.Auth.Type == mltp.EndpointAuthJWT {
		return nil, apperrors.New(apperrors.CodeValidation,
			"endpoint.auth=jwt is not yet supported (SecurityPolicy derivation pending); omit auth or use type=none")
	}
	if existing, err := m.repo.GetByNamespaceName(ctx, namespace, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "traffic policy already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}

	if err := validateModeShape(mode, in.Backends); err != nil {
		return nil, err
	}
	if err := validateWeights(mode, in.Backends); err != nil {
		return nil, err
	}

	family, engine, err := m.validateMembers(ctx, namespace, mode, in.Backends)
	if err != nil {
		return nil, err
	}

	endpoint := in.Endpoint
	if endpoint.Path == "" {
		endpoint.Path = "/services/" + namespace + "/" + in.Name + "/"
	}

	spec := mltp.MLTrafficPolicySpec{
		Backend:  mltp.Backend{Name: family, Engine: engine},
		Mode:     mode,
		Endpoint: endpoint,
		Backends: in.Backends,
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	row := &TrafficPolicy{
		ID:          uuid.New(),
		Namespace:   namespace,
		Name:        in.Name,
		Mode:        in.Mode,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Owner:       auth.User(ctx),
		Labels:      mapBytes(in.Labels),
		Annotations: mapBytes(in.Annotations),
		Spec:        datatypes.JSON(specJSON),
		Generation:  1,
		Phase:       string(StatusCreating),
		StatusJSON:  []byte("{}"),
	}
	if err := m.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) Get(ctx context.Context, namespace, name string) (*View, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "traffic policy not found")
		}
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) List(ctx context.Context, namespace string, limit, offset int, labelClause string, labelArgs []any) ([]View, int64, error) {
	rows, total, err := m.repo.ListByNamespace(ctx, namespace, limit, offset, labelClause, labelArgs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]View, 0, len(rows))
	for i := range rows {
		v, err := m.toView(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (m *Module) Patch(ctx context.Context, namespace, name string, in PatchInput) (*View, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "traffic policy not found")
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
		return m.toView(row)
	}
	if err := m.repo.Update(ctx, row.ID, updates); err != nil {
		return nil, err
	}
	return m.reread(ctx, row.ID)
}

func (m *Module) Delete(ctx context.Context, namespace, name string) error {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	switch Status(row.Phase) {
	case StatusDeleting, StatusDeleted:
		return nil
	}
	return m.repo.MarkDeleting(ctx, row.ID)
}

// Split applies per-backend weight changes, validates the result against the
// mode, and bumps generation so the reconciler patches the CR.
func (m *Module) Split(ctx context.Context, namespace, name string, in SplitInput) (*View, error) {
	row, spec, err := m.load(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	weightByName := make(map[string]int32, len(in.Backends))
	for _, w := range in.Backends {
		weightByName[w.ServiceName] = w.Weight
	}
	for i := range spec.Backends {
		if w, ok := weightByName[spec.Backends[i].ServiceName]; ok {
			spec.Backends[i].Weight = w
		}
	}
	if err := validateWeights(spec.Mode, spec.Backends); err != nil {
		return nil, err
	}
	return m.persistSpec(ctx, row, spec)
}

// Promote (canary only) makes the canary backend the new stable: it swaps the
// stable/canary roles and sets the new stable to 100, old stable to 0. The
// canary baseline is implied by role=stable — there is no separate pointer.
func (m *Module) Promote(ctx context.Context, namespace, name string) (*View, error) {
	row, spec, err := m.load(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if spec.Mode != mltp.TrafficModeCanary {
		return nil, apperrors.New(apperrors.CodePrecondition, "promote is only valid for canary policies")
	}
	si, ci := roleIndex(spec.Backends, mltp.RoleStable), roleIndex(spec.Backends, mltp.RoleCanary)
	if si < 0 || ci < 0 {
		return nil, apperrors.New(apperrors.CodePrecondition, "canary policy must have one stable and one canary backend")
	}
	spec.Backends[ci].Role, spec.Backends[ci].Weight = mltp.RoleStable, 100
	spec.Backends[si].Role, spec.Backends[si].Weight = mltp.RoleCanary, 0
	return m.persistSpec(ctx, row, spec)
}

// Rollback (canary only) returns all traffic to the stable baseline by zeroing
// the canary weight. Roles are unchanged.
func (m *Module) Rollback(ctx context.Context, namespace, name string) (*View, error) {
	row, spec, err := m.load(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if spec.Mode != mltp.TrafficModeCanary {
		return nil, apperrors.New(apperrors.CodePrecondition, "rollback is only valid for canary policies")
	}
	si, ci := roleIndex(spec.Backends, mltp.RoleStable), roleIndex(spec.Backends, mltp.RoleCanary)
	if si < 0 || ci < 0 {
		return nil, apperrors.New(apperrors.CodePrecondition, "canary policy must have one stable and one canary backend")
	}
	spec.Backends[ci].Weight = 0
	spec.Backends[si].Weight = 100
	return m.persistSpec(ctx, row, spec)
}

// load fetches the row + unmarshals its spec.
func (m *Module) load(ctx context.Context, namespace, name string) (*TrafficPolicy, mltp.MLTrafficPolicySpec, error) {
	var spec mltp.MLTrafficPolicySpec
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, spec, apperrors.New(apperrors.CodeNotFound, "traffic policy not found")
		}
		return nil, spec, err
	}
	if err := json.Unmarshal(row.Spec, &spec); err != nil {
		return nil, spec, err
	}
	return row, spec, nil
}

func (m *Module) persistSpec(ctx context.Context, row *TrafficPolicy, spec mltp.MLTrafficPolicySpec) (*View, error) {
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	if err := m.repo.Update(ctx, row.ID, map[string]any{
		"spec":       datatypes.JSON(specJSON),
		"generation": gorm.Expr("generation + 1"),
	}); err != nil {
		return nil, err
	}
	return m.reread(ctx, row.ID)
}

func (m *Module) reread(ctx context.Context, id uuid.UUID) (*View, error) {
	fresh, err := m.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return m.toView(fresh)
}

// validateMembers enforces the authoritative member rules and derives the
// (family, engine) backend tuple.
func (m *Module) validateMembers(ctx context.Context, namespace string, mode mltp.TrafficMode, backends []mltp.BackendMember) (string, string, error) {
	if m.members == nil {
		// Pure-DB tests without a services repo skip member validation.
		return mltp.BackendKindNative, mltp.EngineHTTPRoute, nil
	}
	families := map[string]struct{}{}
	for _, b := range backends {
		if b.ServiceName == "" {
			return "", "", apperrors.New(apperrors.CodeValidation, "backends[].serviceName is required")
		}
		row, err := m.members.GetByNamespaceName(ctx, namespace, b.ServiceName)
		if err != nil {
			if servicemod.IsNotFound(err) {
				return "", "", apperrors.Newf(apperrors.CodeValidation, "member service %q not found", b.ServiceName)
			}
			return "", "", err
		}
		if row.Kind != mlservicev1alpha1.ServiceKindService {
			return "", "", apperrors.Newf(apperrors.CodeValidation, "member %q is a workspace, not a service", b.ServiceName)
		}
		if servicemod.Status(row.Phase) != servicemod.StatusReady {
			return "", "", apperrors.Newf(apperrors.CodePrecondition, "member %q is not Ready (phase=%s)", b.ServiceName, row.Phase)
		}
		refs, err := m.repo.FindActiveReferencing(ctx, namespace, b.ServiceName)
		if err != nil {
			return "", "", err
		}
		if len(refs) > 0 {
			return "", "", apperrors.Newf(apperrors.CodeConflict,
				"member %q is already used by active policy %q", b.ServiceName, refs[0].Name)
		}
		var sp mlservicev1alpha1.MLServiceSpec
		_ = json.Unmarshal(row.Spec, &sp)
		fam := sp.Backend.Name
		if fam == "" {
			fam = mltp.BackendKindNative
		}
		families[fam] = struct{}{}
	}
	if len(families) != 1 {
		return "", "", apperrors.New(apperrors.CodeValidation,
			"all member services must share the same backend family (all native or all kserve)")
	}
	var family string
	for f := range families {
		family = f
	}
	switch family {
	case mltp.BackendKindNative:
		return mltp.BackendKindNative, mltp.EngineHTTPRoute, nil
	case mltp.BackendKindKServe:
		if mode != mltp.TrafficModeCanary || len(backends) != 2 {
			return "", "", apperrors.New(apperrors.CodeValidation,
				"kserve members require mode=canary with exactly 2 backends")
		}
		return mltp.BackendKindKServe, mltp.EngineInference, nil
	default:
		return "", "", apperrors.Newf(apperrors.CodeValidation, "unsupported backend family %q", family)
	}
}

func roleIndex(backends []mltp.BackendMember, role mltp.BackendRole) int {
	for i := range backends {
		if backends[i].Role == role {
			return i
		}
	}
	return -1
}

func validateModeShape(mode mltp.TrafficMode, backends []mltp.BackendMember) error {
	switch mode {
	case mltp.TrafficModeWeighted:
		if len(backends) < 2 {
			return apperrors.New(apperrors.CodeValidation, "weighted mode requires at least 2 backends")
		}
	case mltp.TrafficModeCanary:
		if len(backends) != 2 {
			return apperrors.New(apperrors.CodeValidation, "canary mode requires exactly 2 backends")
		}
		if roleIndex(backends, mltp.RoleStable) < 0 || roleIndex(backends, mltp.RoleCanary) < 0 {
			return apperrors.New(apperrors.CodeValidation,
				"canary mode requires exactly one role=stable and one role=canary backend")
		}
	case mltp.TrafficModeBlueGreen:
		if len(backends) != 2 {
			return apperrors.New(apperrors.CodeValidation, "bluegreen mode requires exactly 2 backends")
		}
	}
	seen := map[string]bool{}
	for _, b := range backends {
		if seen[b.ServiceName] {
			return apperrors.Newf(apperrors.CodeValidation, "duplicate member service %q", b.ServiceName)
		}
		seen[b.ServiceName] = true
	}
	return nil
}

func validateWeights(mode mltp.TrafficMode, backends []mltp.BackendMember) error {
	var sum int32
	for _, b := range backends {
		if b.Weight < 0 || b.Weight > 100 {
			return apperrors.Newf(apperrors.CodeValidation, "backend %q weight must be 0..100", b.ServiceName)
		}
		sum += b.Weight
	}
	switch mode {
	case mltp.TrafficModeWeighted, mltp.TrafficModeCanary:
		if sum != 100 {
			return apperrors.Newf(apperrors.CodeValidation, "backend weights must sum to 100; got %d", sum)
		}
	case mltp.TrafficModeBlueGreen:
		var hundreds, zeros int
		for _, b := range backends {
			switch b.Weight {
			case 100:
				hundreds++
			case 0:
				zeros++
			}
		}
		if hundreds != 1 || zeros != len(backends)-1 {
			return apperrors.New(apperrors.CodeValidation,
				"bluegreen mode requires exactly one backend at weight 100 and the rest at 0")
		}
	}
	return nil
}

func mapBytes(m map[string]string) datatypes.JSON {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return b
}

func (m *Module) toView(p *TrafficPolicy) (*View, error) {
	var spec mltp.MLTrafficPolicySpec
	if len(p.Spec) > 0 {
		_ = json.Unmarshal(p.Spec, &spec)
	}
	var status StatusFields
	if len(p.StatusJSON) > 0 {
		_ = json.Unmarshal(p.StatusJSON, &status)
	}
	return &View{
		ID:                 p.ID,
		Namespace:          p.Namespace,
		Name:               p.Name,
		Mode:               p.Mode,
		DisplayName:        p.DisplayName,
		Description:        p.Description,
		Owner:              p.Owner,
		Labels:             decodeStringMap(p.Labels),
		Annotations:        decodeStringMap(p.Annotations),
		Generation:         p.Generation,
		ObservedGeneration: p.ObservedGeneration,
		Phase:              p.Phase,
		Spec:               spec,
		Status:             status,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		DeletedAt:          p.DeletedAt,
	}, nil
}
