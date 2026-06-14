package server

import "time"

// Quota is one ElasticQuota slot within a (tenant, pool).
type Quota struct {
	Pool string      `json:"pool" binding:"dns1123"`
	Name string      `json:"name" binding:"dns1123"`
	Min  ResourceMap `json:"min,omitempty"`
	Max  ResourceMap `json:"max"`
}

// QuotaStatus is the live status of one quota.
type QuotaStatus struct {
	Pool             string      `json:"pool"`
	Name             string      `json:"name"`
	Used             ResourceMap `json:"used,omitempty"`
	ElasticQuotaName string      `json:"elasticQuotaName,omitempty"`
}

// QuotaList is a list of quotas plus their live statuses.
type QuotaList struct {
	Items    []Quota       `json:"items"`
	Statuses []QuotaStatus `json:"statuses,omitempty"`
	Count    int           `json:"count" binding:"min=0"`
}

// QuotaCreateRequest adds a quota to a tenant.
type QuotaCreateRequest struct {
	Pool string      `json:"pool" binding:"required"`
	Name string      `json:"name" binding:"required"`
	Min  ResourceMap `json:"min,omitempty"`
	Max  ResourceMap `json:"max" binding:"required"`
}

// QuotaPatchRequest patches a quota's min/max only.
type QuotaPatchRequest struct {
	Min ResourceMap `json:"min,omitempty"`
	Max ResourceMap `json:"max,omitempty"`
}

// Namespace is a tenant-owned Kubernetes Namespace declaration.
type Namespace struct {
	Name        string    `json:"name" binding:"dns1123,min=1,max=63"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
}

// SecretSourceRef references a Secret in another namespace.
type SecretSourceRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ConfigMapSourceRef references a ConfigMap in another namespace.
type ConfigMapSourceRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ImagePullSecretInit seeds a per-tenant image pull secret.
type ImagePullSecretInit struct {
	Name            string          `json:"name"`
	SourceSecretRef SecretSourceRef `json:"sourceSecretRef"`
}

// SecretInit seeds a per-tenant Secret.
type SecretInit struct {
	Name            string          `json:"name"`
	Type            string          `json:"type,omitempty"`
	SourceSecretRef SecretSourceRef `json:"sourceSecretRef"`
}

// ConfigMapInit seeds a per-tenant ConfigMap.
type ConfigMapInit struct {
	Name               string             `json:"name"`
	SourceConfigMapRef ConfigMapSourceRef `json:"sourceConfigMapRef"`
}

// ServiceAccountInit seeds a per-tenant ServiceAccount.
type ServiceAccountInit struct {
	Name             string         `json:"name"`
	ImagePullSecrets []string       `json:"imagePullSecrets,omitempty"`
	RBAC             map[string]any `json:"rbac,omitempty"`
}

// InitResources is the set of bootstrap objects created in a tenant namespace.
type InitResources struct {
	ImagePullSecrets []ImagePullSecretInit `json:"imagePullSecrets,omitempty"`
	Secrets          []SecretInit          `json:"secrets,omitempty"`
	ConfigMaps       []ConfigMapInit       `json:"configMaps,omitempty"`
	ServiceAccounts  []ServiceAccountInit  `json:"serviceAccounts,omitempty"`
}

// Tenant is the Platform-facing Tenant DTO.
type Tenant struct {
	Name               string       `json:"name"`
	Namespace          string       `json:"namespace" binding:"max=100"`
	DisplayName        string       `json:"displayName"`
	Description        string       `json:"description,omitempty"`
	Owner              string       `json:"owner,omitempty"`
	Labels             StringMap    `json:"labels,omitempty"`
	Annotations        StringMap    `json:"annotations,omitempty"`
	Spec               TenantSpec   `json:"spec"`
	Generation         int64        `json:"generation" binding:"min=1"`
	ObservedGeneration int64        `json:"observedGeneration,omitempty" binding:"min=0"`
	Phase              TenantPhase  `json:"phase"`
	Status             TenantStatus `json:"status,omitempty"`
	ComputeNamespace   string       `json:"computeNamespace,omitempty"`
	ArtifactsNamespace string       `json:"artifactsNamespace,omitempty"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
	DeletedAt          *time.Time   `json:"deletedAt,omitempty"`
}

// TenantSpec is the declarative tenant spec.
type TenantSpec struct {
	Namespace     Namespace     `json:"namespace"`
	Quotas        []Quota       `json:"quotas,omitempty"`
	InitResources InitResources `json:"initResources,omitempty"`
}

// TenantStatus is the tenant status sub-object (phase lives on Tenant).
type TenantStatus struct {
	Message    string        `json:"message,omitempty"`
	Conditions []Condition   `json:"conditions,omitempty"`
	Quotas     []QuotaStatus `json:"quotas,omitempty"`
}

// TenantList is a page of Tenant.
type TenantList struct {
	Items         []Tenant `json:"items"`
	Count         int      `json:"count" binding:"min=0"`
	ContinueToken string   `json:"continueToken,omitempty"`
}

// TenantCreateRequest is the body of POST /tenants.
type TenantCreateRequest struct {
	Name        string     `json:"name" binding:"required,dns1123,min=3,max=40"`
	Namespace   string     `json:"namespace" binding:"required,max=100"`
	DisplayName string     `json:"displayName" binding:"required,min=1,max=100"`
	Description string     `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap  `json:"labels,omitempty"`
	Annotations StringMap  `json:"annotations,omitempty"`
	Spec        TenantSpec `json:"spec" binding:"required"`
}

// TenantPatchRequest is the JSON Merge Patch body of PATCH /tenants/{name}.
type TenantPatchRequest struct {
	Namespace   string    `json:"namespace,omitempty" binding:"max=100"`
	DisplayName string    `json:"displayName,omitempty" binding:"min=1,max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
	Spec        struct {
		Namespace struct {
			Labels      StringMap `json:"labels,omitempty"`
			Annotations StringMap `json:"annotations,omitempty"`
		} `json:"namespace,omitempty"`
		Quotas        []Quota       `json:"quotas,omitempty"`
		InitResources InitResources `json:"initResources,omitempty"`
	} `json:"spec,omitempty"`
}

// Member is a user↔role binding within a tenant.
type Member struct {
	UserID      UUID      `json:"userId"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName,omitempty"`
	Email       Email     `json:"email,omitempty"`
	RoleName    RoleName  `json:"roleName"`
	AddedAt     time.Time `json:"addedAt"`
}

// MemberList is a page of Member.
type MemberList struct {
	Items         []Member `json:"items"`
	Count         int      `json:"count" binding:"min=0"`
	ContinueToken string   `json:"continueToken,omitempty"`
}

// MemberCreateRequest binds a user to a tenant role.
type MemberCreateRequest struct {
	UserID   UUID           `json:"userId" binding:"required"`
	RoleName MemberRoleName `json:"roleName" binding:"required"`
}

// MemberPatchRequest changes a member's role.
type MemberPatchRequest struct {
	RoleName MemberRoleName `json:"roleName" binding:"required"`
}
