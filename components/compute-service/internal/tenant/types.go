package tenant

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SpecJSON is the structured shape persisted to tenants.spec.
type SpecJSON struct {
	Namespace     NamespaceSpec  `json:"namespace"`
	Quotas        []QuotaSpec    `json:"quotas,omitempty"`
	InitResources *InitResources `json:"initResources,omitempty"`
}

type NamespaceSpec struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type QuotaSpec struct {
	Pool string            `json:"pool"`
	Name string            `json:"name"`
	Min  map[string]string `json:"min,omitempty"`
	Max  map[string]string `json:"max"`
}

type InitResources struct {
	ImagePullSecrets []ImagePullSecretSpec `json:"imagePullSecrets,omitempty"`
	Secrets          []SecretSpec          `json:"secrets,omitempty"`
	ConfigMaps       []ConfigMapSpec       `json:"configMaps,omitempty"`
	ServiceAccounts  []ServiceAccountSpec  `json:"serviceAccounts,omitempty"`
}

type ImagePullSecretSpec struct {
	Name            string    `json:"name"`
	SourceSecretRef ObjectRef `json:"sourceSecretRef"`
}

type SecretSpec struct {
	Name            string    `json:"name"`
	Type            string    `json:"type,omitempty"`
	SourceSecretRef ObjectRef `json:"sourceSecretRef"`
}

type ConfigMapSpec struct {
	Name               string    `json:"name"`
	SourceConfigMapRef ObjectRef `json:"sourceConfigMapRef"`
}

type ServiceAccountSpec struct {
	Name             string    `json:"name"`
	ImagePullSecrets []string  `json:"imagePullSecrets,omitempty"`
	RBAC             *RBACSpec `json:"rbac,omitempty"`
}

// RBACSpec mirrors tenant-operator's RBAC spec — `rules` builds a Role,
// or `roleRef` references an existing ClusterRole. Per design §4.1.3.
type RBACSpec struct {
	Rules   []RBACRule   `json:"rules,omitempty"`
	RoleRef *RBACRoleRef `json:"roleRef,omitempty"`
}

type RBACRule struct {
	APIGroups       []string `json:"apiGroups,omitempty"`
	Resources       []string `json:"resources,omitempty"`
	Verbs           []string `json:"verbs,omitempty"`
	ResourceNames   []string `json:"resourceNames,omitempty"`
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
}

type RBACRoleRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type ObjectRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// CreateInput is the body for POST /api/v1/namespaces.
type CreateInput struct {
	Name          string            `json:"name" binding:"required,axisml_name"`
	DisplayName   string            `json:"displayName,omitempty"`
	Description   string            `json:"description,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Namespace     NamespaceSpec     `json:"namespace" binding:"required"`
	Quotas        []QuotaSpec       `json:"quotas,omitempty"`
	InitResources *InitResources    `json:"initResources,omitempty"`
}

// PatchInput is the body for PATCH /api/v1/namespaces/{name}. The wire
// shape mirrors the design yaml: top-level display fields plus a nested
// `spec` block carrying the mutable spec entries.
//
// Per design §4.1, `spec.namespace.{labels,annotations}` mutations rewrite
// the CR field but DO NOT bump generation (tenant-operator doesn't recreate
// the K8s namespace — see tenant-operator.md §4.1.1). Quotas / initResources
// mutations DO bump generation.
type PatchInput struct {
	DisplayName *string             `json:"displayName,omitempty"`
	Description *string             `json:"description,omitempty"`
	Labels      map[string]string   `json:"labels,omitempty"`
	Annotations map[string]string   `json:"annotations,omitempty"`
	Spec        *PatchInputSpecPart `json:"spec,omitempty"`
}

// PatchInputSpecPart carries the design-allowed mutable spec entries.
type PatchInputSpecPart struct {
	Namespace     *PatchInputNamespacePart `json:"namespace,omitempty"`
	Quotas        *[]QuotaSpec             `json:"quotas,omitempty"`
	InitResources *InitResources           `json:"initResources,omitempty"`
}

// PatchInputNamespacePart wraps the two PG-only display fields. Labels /
// annotations are patched as a whole-map replace.
type PatchInputNamespacePart struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Response is the JSON shape returned by GET / LIST / Create / Patch.
type Response struct {
	ID                 uuid.UUID         `json:"id"`
	Name               string            `json:"name"`
	DisplayName        string            `json:"displayName,omitempty"`
	Description        string            `json:"description,omitempty"`
	Owner              string            `json:"owner,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
	Namespace          NamespaceSpec     `json:"namespace"`
	Quotas             []QuotaSpec       `json:"quotas,omitempty"`
	InitResources      *InitResources    `json:"initResources,omitempty"`
	Phase              string            `json:"phase"`
	Status             json.RawMessage   `json:"status,omitempty"`
	Generation         int64             `json:"generation"`
	ObservedGeneration int64             `json:"observedGeneration"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	DeletedAt          *time.Time        `json:"deletedAt,omitempty"`
	LastModifiedBy     string            `json:"lastModifiedBy,omitempty"`
}

// ListResponse paginates the LIST endpoint. `count` is the page size
// (== len(items)) per the design yaml; `total` is the unfiltered row
// count kept for client convenience. `continueToken` is a K8s-style
// opaque cursor for the next page; absent on the last page.
type ListResponse struct {
	Items         []Response `json:"items"`
	Count         int        `json:"count"`
	Total         int64      `json:"total"`
	ContinueToken string     `json:"continueToken,omitempty"`
}

func toResponse(t *Tenant) (Response, error) {
	var spec SpecJSON
	if len(t.Spec) > 0 {
		if err := json.Unmarshal(t.Spec, &spec); err != nil {
			return Response{}, err
		}
	}
	var labels, annotations map[string]string
	_ = json.Unmarshal(t.Labels, &labels)
	_ = json.Unmarshal(t.Annotations, &annotations)

	return Response{
		ID:                 t.ID,
		Name:               t.Name,
		DisplayName:        t.DisplayName,
		Description:        t.Description,
		Owner:              t.Owner,
		Labels:             labels,
		Annotations:        annotations,
		Namespace:          spec.Namespace,
		Quotas:             spec.Quotas,
		InitResources:      spec.InitResources,
		Phase:              t.Phase,
		Status:             json.RawMessage(t.Status),
		Generation:         t.Generation,
		ObservedGeneration: t.ObservedGeneration,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
		DeletedAt:          t.DeletedAt,
		LastModifiedBy:     t.LastModifiedBy,
	}, nil
}

func toSpecBytes(in CreateInput) (datatypes.JSON, error) {
	s := SpecJSON{
		Namespace:     in.Namespace,
		Quotas:        in.Quotas,
		InitResources: in.InitResources,
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func mapToJSON(m map[string]string) datatypes.JSON {
	if m == nil {
		m = map[string]string{}
	}
	b, _ := json.Marshal(m)
	return b
}
