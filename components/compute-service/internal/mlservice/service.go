package mlservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/auth"
	"github.com/axisml/axisml/components/compute-service/internal/poolcache"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	"github.com/axisml/axisml/components/compute-service/internal/store"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/strutil"
)

// Module wraps the service business layer. Keyed on bare namespace strings.
type Module struct {
	repo       *Repository
	db         *gorm.DB
	pools      *poolcache.Reader
	k8s        client.Client
	policyRefs ActivePolicyReferenceChecker
}

// ActivePolicyReferenceChecker prevents deleting an online service while a
// live traffic policy still routes to it.
type ActivePolicyReferenceChecker interface {
	ActiveReferenceName(ctx context.Context, namespace, serviceName string) (string, error)
}

// NewMLService builds the service module wiring. The k8sClient is used for
// workspace PVC management (kind=workspace) and may be nil in pure-DB tests.
func NewMLService(
	db *gorm.DB,
	pools *poolcache.Reader,
	k8sClient client.Client,
	policyRefs ActivePolicyReferenceChecker,
) *Module {
	return &Module{
		repo:       NewRepository(db),
		db:         db,
		pools:      pools,
		k8s:        k8sClient,
		policyRefs: policyRefs,
	}
}

func (m *Module) Create(ctx context.Context, namespace string, in server.MLServiceCreateRequest) (*server.MLService, error) {
	if !strutil.IsValidName(in.Name) {
		return nil, apperrors.New(apperrors.CodeValidation, "invalid service name")
	}
	if in.Quota == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "quota is required")
	}
	kind := in.Kind
	if kind == "" {
		kind = mlservicev1alpha1.ServiceKindService
	}
	if kind != mlservicev1alpha1.ServiceKindService && kind != mlservicev1alpha1.ServiceKindWorkspace {
		return nil, apperrors.Newf(apperrors.CodeValidation,
			"kind must be %q or %q", mlservicev1alpha1.ServiceKindService, mlservicev1alpha1.ServiceKindWorkspace)
	}
	if existing, err := m.repo.GetByNamespaceName(ctx, namespace, in.Name); err == nil && existing != nil {
		return nil, apperrors.New(apperrors.CodeConflict, "service already exists")
	} else if err != nil && !IsNotFound(err) {
		return nil, err
	}
	expanded, err := m.pools.Resolve(ctx, in.PoolName, in.UnitName)
	if err != nil {
		return nil, err
	}

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
		role.Template.Resources = poolcache.BuildResources(expanded.Requests, expanded.Limits)
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
		Backend: backend,
		Scheduling: mlservicev1alpha1.Scheduling{
			Quota:         in.Quota,
			PriorityClass: in.PriorityClass,
			NodeSelector:  expanded.NodeSelector,
			Tolerations:   expanded.Tolerations,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
		Route:     in.Route,
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}

	// For kind=workspace, materialise the backing PVC first; if PVC creation
	// fails we surface the error and never write the DB row. Per design §4.4,
	// PG row + PVC are a logical pair, and the PVC must be wired into
	// roles[0].template.{volumes,volumeMounts} so the Pod actually mounts it.
	if kind == mlservicev1alpha1.ServiceKindWorkspace {
		if err := m.ensureWorkspacePVC(ctx, namespace, in.Name, in.WorkspaceStorage); err != nil {
			return nil, err
		}
		if len(spec.Roles) > 0 {
			injectWorkspaceVolume(&spec.Roles[0], in.Name)
		}
		// Re-marshal the spec now that we've injected the volume mount.
		if specJSON, err = json.Marshal(spec); err != nil {
			return nil, err
		}
	}

	mergedLabels := mergeSvcLabels(in.Labels, map[string]string{
		mlservicev1alpha1.LabelResourcePool: in.PoolName,
		mlservicev1alpha1.LabelResourceUnit: in.UnitName,
	})
	// status.readyReplicas starts at 0; the desired-replica value lives only
	// in spec.roles[0].replicas (no separate `replicas` column anymore).
	_ = replicas
	row := &store.MLService{
		ID:          uuid.New(),
		Namespace:   namespace,
		Kind:        kind,
		Name:        in.Name,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Owner:       auth.User(ctx),
		Labels:      svcMapBytes(mergedLabels),
		Annotations: svcMapBytes(in.Annotations),
		Spec:        datatypes.JSON(specJSON),
		Generation:  1,
		Phase:       string(StatusCreating),
		StatusJSON:  []byte("{}"),
	}
	if err := m.repo.Create(ctx, row); err != nil {
		// Workspace path created a PVC first; clean it up so we don't leak
		// orphan storage on a unique-violation or other terminal DB error.
		if kind == mlservicev1alpha1.ServiceKindWorkspace && m.k8s != nil {
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      WorkspacePVCName(in.Name),
					Namespace: namespace,
				},
			}
			_ = m.k8s.Delete(ctx, pvc)
		}
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
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	var spec mlservicev1alpha1.MLServiceSpec
	if err := json.Unmarshal(row.Spec, &spec); err != nil {
		return nil, err
	}
	if len(spec.Roles) == 0 {
		return nil, apperrors.New(apperrors.CodePrecondition, "service has no roles")
	}
	spec.Roles[0].Replicas = in.Replicas

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	// Bump generation so the reconciler's `generation <> observed_generation`
	// predicate picks the row up; observed_generation lands when the patch
	// hits the CR (per design §5.2).
	if err := m.repo.Update(ctx, row.ID, map[string]any{
		"spec":       datatypes.JSON(specJSON),
		"generation": gorm.Expr("generation + 1"),
	}); err != nil {
		return nil, err
	}
	// Re-read so the returned view reflects the bumped generation.
	fresh, err := m.repo.Get(ctx, row.ID)
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

func (m *Module) Delete(ctx context.Context, namespace, name string, deletePVC bool) error {
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
	if err := m.repo.MarkDeleting(ctx, row.ID); err != nil {
		return err
	}
	// For workspaces, default behaviour is to delete the backing PVC alongside
	// the row. Callers that want to preserve the volume must pass
	// deletePVC=false (the ?deletePvc=false query in the HTTP layer).
	if row.Kind == mlservicev1alpha1.ServiceKindWorkspace && deletePVC && m.k8s != nil {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      WorkspacePVCName(name),
				Namespace: namespace,
			},
		}
		if delErr := m.k8s.Delete(ctx, pvc); delErr != nil && !apierrors.IsNotFound(delErr) {
			return apperrors.Wrap(apperrors.CodeUnavailable, "delete workspace pvc", delErr)
		}
	}
	return nil
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

// injectWorkspaceVolume wires the backing PVC into the role's pod template.
// The volume is named `workspace`, mounted at /workspace. If the caller
// already supplied a volume of the same name we leave their choice alone
// (deterministic-PVC name still wins as the claim it points at).
func injectWorkspaceVolume(role *mlservicev1alpha1.RoleSpec, serviceName string) {
	const volumeName = "workspace"
	const mountPath = "/workspace"

	hasVolume := false
	for i := range role.Template.Volumes {
		if role.Template.Volumes[i].Name == volumeName {
			role.Template.Volumes[i].VolumeSource = corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: WorkspacePVCName(serviceName),
				},
			}
			hasVolume = true
			break
		}
	}
	if !hasVolume {
		role.Template.Volumes = append(role.Template.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: WorkspacePVCName(serviceName),
				},
			},
		})
	}
	hasMount := false
	for i := range role.Template.VolumeMounts {
		if role.Template.VolumeMounts[i].Name == volumeName {
			hasMount = true
			break
		}
	}
	if !hasMount {
		role.Template.VolumeMounts = append(role.Template.VolumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPath,
		})
	}
}

// WorkspacePVCName is the deterministic PVC name used for kind=workspace
// mlservices (design §4.4).
func WorkspacePVCName(serviceName string) string {
	return fmt.Sprintf("axisml-ws-%s-data", serviceName)
}

// ensureWorkspacePVC creates the backing PVC for a kind=workspace service.
// Idempotent: AlreadyExists is treated as success so retries don't fail the
// Create. When the K8s client isn't wired (unit / pure-DB tests), the call
// is a no-op.
func (m *Module) ensureWorkspacePVC(ctx context.Context, namespace, name string, spec *server.MLServiceWorkspaceStorageSpec) error {
	if m.k8s == nil {
		return nil
	}
	if spec == nil || spec.Size == "" {
		return apperrors.New(apperrors.CodeValidation,
			"workspaceStorage.size is required when kind=workspace")
	}
	q, err := resource.ParseQuantity(spec.Size)
	if err != nil {
		return apperrors.Newf(apperrors.CodeValidation,
			"workspaceStorage.size %q is not a valid Quantity: %v", spec.Size, err)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WorkspacePVCName(name),
			Namespace: namespace,
			Labels: map[string]string{
				mlservicev1alpha1.LabelServiceKind: mlservicev1alpha1.ServiceKindWorkspace,
				"axisml.io/workspace":              name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: q},
			},
		},
	}
	if spec.StorageClass != "" {
		sc := spec.StorageClass
		pvc.Spec.StorageClassName = &sc
	}
	if err := m.k8s.Create(ctx, pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		return apperrors.Wrap(apperrors.CodeUnavailable, "create workspace pvc", err)
	}
	// Re-read so callers / tests can verify the bound PVC if needed.
	_ = m.k8s.Get(ctx, types.NamespacedName{Namespace: namespace, Name: WorkspacePVCName(name)}, pvc)
	return nil
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
