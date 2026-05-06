// Package server defines request/response types and HTTP handler shapes
// shared between the tenant and quota handlers. The cluster-manager is a
// thin shell over the K8s API, so types here mirror Tenant CR fields.
package server

import "time"

// CreateTenantRequest mirrors Tenant.spec plus a top-level `name` for
// metadata.name. Fields map 1:1 to the CRD; cluster-manager validates
// shape (DNS-1123, length, denylist) before forwarding to the API server.
type CreateTenantRequest struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Namespace   NamespaceSpec     `json:"namespace"`
	Quotas      []QuotaSpec       `json:"quotas,omitempty"`
	Init        *InitResources    `json:"initResources,omitempty"`
	Suspended   bool              `json:"suspended,omitempty"`
}

// PatchTenantRequest covers the mutable fields. Immutable fields
// (spec.namespace.name, spec.quotas[].{pool,name}) are rejected at the
// handler with 400.
type PatchTenantRequest struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Init        *InitResources    `json:"initResources,omitempty"`
}

// NamespaceSpec mirrors Tenant.spec.namespace.
type NamespaceSpec struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// QuotaSpec mirrors one entry of Tenant.spec.quotas[].
type QuotaSpec struct {
	Pool string            `json:"pool"`
	Name string            `json:"name"`
	Min  map[string]string `json:"min,omitempty"`
	Max  map[string]string `json:"max"`
}

// InitResources mirrors Tenant.spec.initResources. Fields are pointers /
// slices so PATCH can omit them safely.
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
	Name             string   `json:"name"`
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
	RBAC             *RBAC    `json:"rbac,omitempty"`
}

type RBAC struct {
	Rules   []map[string]any `json:"rules,omitempty"`
	RoleRef *RoleRef         `json:"roleRef,omitempty"`
}

type RoleRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type ObjectRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// TenantResponse is the shape returned for GET / LIST.
type TenantResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName,omitempty"`
	Namespace   NamespaceSpec  `json:"namespace"`
	Quotas      []QuotaSpec    `json:"quotas,omitempty"`
	Init        *InitResources `json:"initResources,omitempty"`
	Suspended   bool           `json:"suspended"`
	Status      TenantStatus   `json:"status"`
	CreatedAt   time.Time      `json:"createdAt,omitempty"`
}

type TenantStatus struct {
	Phase   string        `json:"phase,omitempty"`
	Message string        `json:"message,omitempty"`
	Quotas  []QuotaStatus `json:"quotas,omitempty"`
}

type QuotaStatus struct {
	Pool    string            `json:"pool"`
	Name    string            `json:"name"`
	Ready   bool              `json:"ready"`
	Used    map[string]string `json:"used,omitempty"`
	Message string            `json:"message,omitempty"`
}

// ListTenantsResponse paginates per the K8s API server's continue token.
type ListTenantsResponse struct {
	Items    []TenantResponse `json:"items"`
	Continue string           `json:"continue,omitempty"`
}

// Problem is the RFC 7807 error body cluster-manager returns on 4xx/5xx.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}
