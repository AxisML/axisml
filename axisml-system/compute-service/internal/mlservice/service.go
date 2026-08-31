package mlservice

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/auth"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/resource"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/serviceadmission"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/strutil"
)

// Module wraps the service business layer. Keyed on bare namespace strings.
type Module struct {
	repo       *Repository
	db         *gorm.DB
	pools      extensions.ResourceResolver
	policyRefs ActivePolicyReferenceChecker
}

// ActivePolicyReferenceChecker prevents deleting an online service while a
// live traffic policy still routes to it.
type ActivePolicyReferenceChecker interface {
	ActiveReferenceName(ctx context.Context, namespace, serviceName string) (string, error)
}

// NewMLService builds the service module wiring. Durable volumes (e.g. the PVC
// backing a kind=workspace service) are pre-provisioned by Platform via
// cluster-manager and referenced as a PVC volume in the role template; compute
// does not provision them.
func NewMLService(
	db *gorm.DB,
	pools extensions.ResourceResolver,
	policyRefs ActivePolicyReferenceChecker,
) *Module {
	return &Module{
		repo:       NewRepository(db),
		db:         db,
		pools:      pools,
		policyRefs: policyRefs,
	}
}

func (m *Module) Create(ctx context.Context, namespace string, in server.MLServiceCreateRequest) (*server.MLService, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid service name")
	}
	kind := in.Kind
	if kind == "" {
		kind = mlservicev1alpha1.ServiceKindService
	}
	if kind != mlservicev1alpha1.ServiceKindService &&
		kind != mlservicev1alpha1.ServiceKindWorkspace &&
		kind != mlservicev1alpha1.ServiceKindTensorBoard {
		return nil, apperrors.Newf(apperrors.CodeValidation,
			"kind must be %q, %q or %q", mlservicev1alpha1.ServiceKindService,
			mlservicev1alpha1.ServiceKindWorkspace, mlservicev1alpha1.ServiceKindTensorBoard)
	}
	if existing, err := m.repo.GetByNamespaceName(ctx, namespace, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "service already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}
	pool, err := m.pools.ResolveResourcePool(ctx, in.PoolName)
	if err != nil {
		return nil, err
	}
	unit, err := m.pools.ResolveResourceUnit(ctx, in.PoolName, in.UnitName)
	if err != nil {
		return nil, err
	}
	expanded := resource.Expand(pool, unit)

	backend := mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"}
	if in.Backend != nil {
		if in.Backend.Name != "" {
			backend.Name = in.Backend.Name
		}
		if in.Backend.Engine != "" {
			backend.Engine = in.Backend.Engine
		}
		backend.Config = in.Backend.Config
	}

	roles := make([]mlservicev1alpha1.RoleSpec, len(in.Roles))
	replicas := int32(0)
	for i, r := range in.Roles {
		role := r
		role.Template.Resources = resource.BuildResources(expanded.Requests, expanded.Limits)
		roles[i] = role
		if i == 0 {
			replicas = role.Replicas
		}
	}

	runPolicy := mlservicev1alpha1.RunPolicy{}
	if in.RunPolicy != nil {
		runPolicy = *in.RunPolicy
	}

	spec := mlservicev1alpha1.MLServiceSpec{
		Backend:    backend,
		ConfigMaps: in.ConfigMaps,
		Scheduling: mlservicev1alpha1.Scheduling{
			Quota:         tenantv1alpha1.ElasticQuotaName(namespace, in.PoolName),
			PriorityClass: in.PriorityClass,
			NodeSelector:  expanded.NodeSelector,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
		Route:     in.Route,
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	// A kind=workspace service's durable volume is just a PVC entry the caller
	// (Platform) declares in roles[0].template.{volumes,volumeMounts}; the volume
	// itself is pre-provisioned by Platform via cluster-manager. compute relays
	// the role template as-is and does not provision storage.

	mergedLabels := mergeSvcLabels(in.Labels, map[string]string{
		mlservicev1alpha1.LabelResourcePool: in.PoolName,
		mlservicev1alpha1.LabelResourceUnit: in.UnitName,
	})
	// status.readyReplicas starts at 0; the desired-replica value lives only
	// in spec.roles[0].replicas (no separate `replicas` column anymore).
	_ = replicas
	row := &store.MLService{
		ID:                 uuid.New(),
		Namespace:          namespace,
		Kind:               kind,
		Name:               in.Name,
		DisplayName:        in.DisplayName,
		Description:        in.Description,
		Owner:              auth.User(ctx),
		Labels:             svcMapBytes(mergedLabels),
		Annotations:        svcMapBytes(in.Annotations),
		Spec:               datatypes.JSON(specJSON),
		AdmittedReplicas:   datatypes.JSON(serviceadmission.Encode(serviceadmission.Zero(spec))),
		DispatchedReplicas: datatypes.JSON(serviceadmission.Encode(serviceadmission.Zero(spec))),
		Generation:         1,
		Phase:              string(StatusQueued),
		StatusJSON:         []byte("{}"),
	}
	if err := m.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) Get(ctx context.Context, namespace, name string) (*server.MLService, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	return m.toView(row)
}

// Phase returns the service's current lifecycle phase, readiness and sync
// signal — a lightweight projection for high-frequency polling that skips
// unmarshalling and shipping the spec sub-tree.
func (m *Module) Phase(ctx context.Context, namespace, name string) (*server.MLServicePhase, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	v := phaseView(row)
	return &v, nil
}

// PhasesByNames returns phase projections for the named services in the
// namespace. Names that don't resolve are simply omitted, so a caller diffs the
// returned set against what it asked for.
func (m *Module) PhasesByNames(ctx context.Context, namespace string, names []string) ([]server.MLServicePhase, error) {
	rows, err := m.repo.ListPhasesByNames(ctx, namespace, names)
	if err != nil {
		return nil, err
	}
	return phaseViews(rows), nil
}

// PhasesBySelector returns phase projections for services matching the kind and
// label selector, paginated like List.
func (m *Module) PhasesBySelector(ctx context.Context, namespace, kind string, limit, offset int, labelClause string, labelArgs []any) ([]server.MLServicePhase, int64, error) {
	rows, total, err := m.repo.ListPhasesBySelector(ctx, namespace, kind, limit, offset, labelClause, labelArgs)
	if err != nil {
		return nil, 0, err
	}
	return phaseViews(rows), total, nil
}

// phaseView projects a row onto the lean phase DTO, unmarshalling only the
// status sub-tree (never the spec).
func phaseView(s *store.MLService) server.MLServicePhase {
	var status server.MLServiceStatus
	if len(s.StatusJSON) > 0 {
		_ = json.Unmarshal(s.StatusJSON, &status)
	}
	status.AdmittedReplicas = admittedPrimary(s)
	return server.MLServicePhase{
		Name:               s.Name,
		Phase:              s.Phase,
		Message:            status.Message,
		AdmissionReason:    status.AdmissionReason,
		AdmissionMessage:   status.AdmissionMessage,
		AdmittedReplicas:   status.AdmittedReplicas,
		ReadyReplicas:      status.ReadyReplicas,
		Generation:         s.Generation,
		ObservedGeneration: s.ObservedGeneration,
	}
}

func phaseViews(rows []store.MLService) []server.MLServicePhase {
	out := make([]server.MLServicePhase, 0, len(rows))
	for i := range rows {
		out = append(out, phaseView(&rows[i]))
	}
	return out
}

func (m *Module) List(ctx context.Context, namespace, kind string, limit, offset int, labelClause string, labelArgs []any) ([]server.MLService, int64, error) {
	rows, total, err := m.repo.ListByNamespace(ctx, namespace, kind, limit, offset, labelClause, labelArgs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]server.MLService, 0, len(rows))
	for i := range rows {
		v, err := m.toView(&rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (m *Module) Scale(ctx context.Context, namespace, name string, in server.MLServiceScaleRequest) (*server.MLService, error) {
	var id uuid.UUID
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row store.MLService
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("namespace = ? AND name = ? AND deleted_at IS NULL", namespace, name).
			First(&row).Error; err != nil {
			return err
		}
		id = row.ID
		var spec mlservicev1alpha1.MLServiceSpec
		if err := json.Unmarshal(row.Spec, &spec); err != nil {
			return err
		}
		if len(spec.Roles) == 0 {
			return apperrors.New(apperrors.CodePrecondition, "service has no roles")
		}
		desiredBefore := serviceadmission.Desired(spec)
		admitted := serviceadmission.Decode(row.AdmittedReplicas, len(spec.Roles), desiredBefore)
		spec.Roles[0].Replicas = in.Replicas
		desiredAfter := serviceadmission.Desired(spec)
		admitted = serviceadmission.ClampToDesired(admitted, desiredAfter)
		specJSON, err := json.Marshal(spec)
		if err != nil {
			return err
		}
		var status server.MLServiceStatus
		_ = json.Unmarshal(row.StatusJSON, &status)
		status.AdmissionReason = ""
		status.AdmissionMessage = ""
		status.AdmittedReplicas = 0 // API reads derive the authoritative value from admitted_replicas.
		statusJSON, _ := json.Marshal(status)
		return tx.Model(&store.MLService{}).Where("id = ?", row.ID).Updates(map[string]any{
			"spec":              datatypes.JSON(specJSON),
			"admitted_replicas": datatypes.JSON(serviceadmission.Encode(admitted)),
			"status":            datatypes.JSON(statusJSON),
			"generation":        gorm.Expr("generation + 1"),
		}).Error
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	// Re-read so the returned view reflects the bumped generation and any
	// immediate scale-down release.
	fresh, err := m.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return m.toView(fresh)
}

// Patch updates the row's display-tier metadata. Pure PG mutation —
// no CR is touched, no generation bump.
func (m *Module) Patch(ctx context.Context, namespace, name string, in server.MLServicePatchRequest) (*server.MLService, error) {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
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
		updates["labels"] = svcMapBytes(in.Labels)
	}
	if in.Annotations != nil {
		updates["annotations"] = svcMapBytes(in.Annotations)
	}
	if len(updates) == 0 {
		return m.toView(row)
	}
	if err := m.repo.Update(ctx, row.ID, updates); err != nil {
		return nil, err
	}
	fresh, err := m.repo.Get(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return m.toView(fresh)
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
	if row.Kind == mlservicev1alpha1.ServiceKindService && m.policyRefs != nil {
		if err := checkActivePolicyReference(ctx, m.policyRefs, namespace, name); err != nil {
			return err
		}
	}
	if Status(row.Phase) == StatusQueued {
		// Queued guarantees that no runtime object exists, so deletion is a
		// database-only terminal transition. Policy references are still checked
		// above because admission state does not weaken referential integrity.
		return m.repo.MarkDeleted(ctx, row.ID)
	}
	// A workspace's durable volume is reclaimed by Platform via cluster-manager,
	// not here — compute only soft-deletes the row and tears down the CR.
	return m.repo.MarkDeleting(ctx, row.ID)
}

func checkActivePolicyReference(
	ctx context.Context,
	checker ActivePolicyReferenceChecker,
	namespace string,
	name string,
) error {
	policyName, err := checker.ActiveReferenceName(ctx, namespace, name)
	if err != nil {
		return err
	}
	if policyName == "" {
		return nil
	}
	return apperrors.Newf(apperrors.CodeConflict,
		"service %q is referenced by active traffic policy %q", name, policyName)
}

func mergeSvcLabels(user, system map[string]string) map[string]string {
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

func svcMapBytes(m map[string]string) datatypes.JSON {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return b
}

func (m *Module) toView(s *store.MLService) (*server.MLService, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if len(s.Spec) > 0 {
		_ = json.Unmarshal(s.Spec, &spec)
	}
	var status server.MLServiceStatus
	if len(s.StatusJSON) > 0 {
		_ = json.Unmarshal(s.StatusJSON, &status)
	}
	status.AdmittedReplicas = admittedPrimary(s)
	return &server.MLService{
		ID:                 s.ID,
		Namespace:          s.Namespace,
		Name:               s.Name,
		Kind:               s.Kind,
		DisplayName:        s.DisplayName,
		Description:        s.Description,
		Owner:              s.Owner,
		Labels:             decodeStringMap(s.Labels),
		Annotations:        decodeStringMap(s.Annotations),
		Generation:         s.Generation,
		ObservedGeneration: s.ObservedGeneration,
		Phase:              s.Phase,
		Spec:               spec,
		Status:             status,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
		DeletedAt:          s.DeletedAt,
	}, nil
}

func admittedPrimary(s *store.MLService) int32 {
	// The lightweight phase query deliberately does not select the desired spec;
	// primary admission is the first element of the independently persisted
	// replica vector.
	return serviceadmission.Primary(serviceadmission.Decode(s.AdmittedReplicas, 1, nil))
}
