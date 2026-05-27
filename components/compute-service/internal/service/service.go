package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	"github.com/axisml/axisml/components/compute-service/internal/spechash"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/strutil"
)

// Module wraps the service business layer. Keyed on bare namespace strings.
type Module struct {
	repo  *Repository
	db    *gorm.DB
	pools *poolcache.Reader
	k8s   client.Client
}

// NewService builds the service module wiring. The k8sClient is used for
// workspace PVC management (kind=workspace) and may be nil in pure-DB tests.
func NewService(db *gorm.DB, pools *poolcache.Reader, k8sClient client.Client) *Module {
	return &Module{
		repo:  NewRepository(db),
		db:    db,
		pools: pools,
		k8s:   k8sClient,
	}
}

// CreateInput is the API request body. Caller selects pool/unit by NAME
// (the ResourcePool CRD lives in K8s; compute reads it via Informer cache).
// `Quota` is the ElasticQuota CR name (cluster-unique string) Compute stamps
// onto Pod labels — opaque, no existence check. `Kind` distinguishes a
// regular online service from a Platform workspace; immutable after create.
// When kind=workspace, `WorkspaceStorage` describes the PVC backing the
// workspace (size required; storageClass optional, falls back to the
// cluster default).
type CreateInput struct {
	Name             string                       `json:"name" binding:"required,axisml_name"`
	Kind             string                       `json:"kind,omitempty"`
	DisplayName      string                       `json:"displayName"`
	Description      string                       `json:"description"`
	PoolName         string                       `json:"poolName" binding:"required"`
	UnitName         string                       `json:"unitName" binding:"required"`
	Quota            string                       `json:"quota" binding:"required"`
	Backend          *mlservicev1alpha1.Backend   `json:"backend"`
	Roles            []mlservicev1alpha1.RoleSpec `json:"roles" binding:"required,min=1"`
	RunPolicy        *mlservicev1alpha1.RunPolicy `json:"runPolicy"`
	Route            *mlservicev1alpha1.Route     `json:"route"`
	WorkspaceStorage *WorkspaceStorageSpec        `json:"workspaceStorage,omitempty"`
}

// WorkspaceStorageSpec carries the PVC sizing for kind=workspace services.
type WorkspaceStorageSpec struct {
	Size         string `json:"size" binding:"required"`
	StorageClass string `json:"storageClass,omitempty"`
}

// ScaleInput is the body for /:scale.
type ScaleInput struct {
	Replicas int32 `json:"replicas" binding:"required,gte=0"`
}

// View is the HTTP response.
type View struct {
	ID            uuid.UUID                       `json:"id"`
	Namespace     string                          `json:"namespace"`
	Name          string                          `json:"name"`
	Kind          string                          `json:"kind"`
	PoolName      string                          `json:"poolName,omitempty"`
	UnitName      string                          `json:"unitName,omitempty"`
	DisplayName   string                          `json:"displayName,omitempty"`
	Description   string                          `json:"description,omitempty"`
	OwnerUser     string                          `json:"ownerUser,omitempty"`
	Replicas      int32                           `json:"replicas"`
	ReadyReplicas int32                           `json:"readyReplicas"`
	Endpoint      string                          `json:"endpoint,omitempty"`
	Status        string                          `json:"status"`
	Message       string                          `json:"message,omitempty"`
	Spec          mlservicev1alpha1.MLServiceSpec `json:"spec"`
	CreatedAt     time.Time                       `json:"createdAt"`
	UpdatedAt     time.Time                       `json:"updatedAt"`
}

func (m *Module) Create(ctx context.Context, namespace string, in CreateInput) (*View, error) {
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
			Quota:        in.Quota,
			NodeSelector: expanded.NodeSelector,
			Tolerations:  expanded.Tolerations,
		},
		Roles:     roles,
		RunPolicy: runPolicy,
		Route:     in.Route,
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	hash, err := spechash.Compute(spec)
	if err != nil {
		return nil, err
	}
	reqJSON, err := json.Marshal(expanded.Requests)
	if err != nil {
		return nil, err
	}

	// For kind=workspace, materialise the backing PVC first; if PVC creation
	// fails we surface the error and never write the DB row. Per design §4.4,
	// PG row + PVC are a logical pair.
	if kind == mlservicev1alpha1.ServiceKindWorkspace {
		if err := m.ensureWorkspacePVC(ctx, namespace, in.Name, in.WorkspaceStorage); err != nil {
			return nil, err
		}
	}

	row := &Service{
		ID:                 uuid.New(),
		Namespace:          namespace,
		Kind:               kind,
		PoolName:           in.PoolName,
		UnitName:           in.UnitName,
		Name:               in.Name,
		DisplayName:        in.DisplayName,
		Description:        in.Description,
		OwnerUser:          auth.User(ctx),
		Spec:               datatypes.JSON(specJSON),
		DesiredSpecHash:    hash,
		RequestedResources: datatypes.JSON(reqJSON),
		Replicas:           replicas,
		Status:             string(StatusCreating),
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
			return nil, apperrors.New(apperrors.CodeNotFound, "service not found")
		}
		return nil, err
	}
	return m.toView(row)
}

func (m *Module) List(ctx context.Context, namespace string, limit, offset int) ([]View, int64, error) {
	rows, total, err := m.repo.ListByNamespace(ctx, namespace, limit, offset)
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

func (m *Module) Scale(ctx context.Context, namespace, name string, in ScaleInput) (*View, error) {
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

	hash, err := spechash.Compute(spec)
	if err != nil {
		return nil, err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	if err := m.repo.Update(ctx, row.ID, map[string]any{
		"spec":              datatypes.JSON(specJSON),
		"desired_spec_hash": hash,
		"replicas":          in.Replicas,
	}); err != nil {
		return nil, err
	}
	row.Spec = datatypes.JSON(specJSON)
	row.DesiredSpecHash = hash
	row.Replicas = in.Replicas
	row.UpdatedAt = time.Now().UTC()
	return m.toView(row)
}

func (m *Module) Delete(ctx context.Context, namespace, name string, deletePVC bool) error {
	row, err := m.repo.GetByNamespaceName(ctx, namespace, name)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	switch Status(row.Status) {
	case StatusDeleting, StatusDeleted:
		return nil
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

// WorkspacePVCName is the deterministic PVC name used for kind=workspace
// services (design §4.4).
func WorkspacePVCName(serviceName string) string {
	return fmt.Sprintf("axisml-ws-%s-data", serviceName)
}

// ensureWorkspacePVC creates the backing PVC for a kind=workspace service.
// Idempotent: AlreadyExists is treated as success so retries don't fail the
// Create. When the K8s client isn't wired (unit / pure-DB tests), the call
// is a no-op.
func (m *Module) ensureWorkspacePVC(ctx context.Context, namespace, name string, spec *WorkspaceStorageSpec) error {
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

func (m *Module) toView(s *Service) (*View, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if len(s.Spec) > 0 {
		_ = json.Unmarshal(s.Spec, &spec)
	}
	return &View{
		ID:            s.ID,
		Namespace:     s.Namespace,
		Name:          s.Name,
		Kind:          s.Kind,
		PoolName:      s.PoolName,
		UnitName:      s.UnitName,
		DisplayName:   s.DisplayName,
		Description:   s.Description,
		OwnerUser:     s.OwnerUser,
		Replicas:      s.Replicas,
		ReadyReplicas: s.ReadyReplicas,
		Endpoint:      s.Endpoint,
		Status:        s.Status,
		Message:       s.Message,
		Spec:          spec,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}, nil
}
