package server

import "time"

// QuotaUnit allocates a quantity of one named resource unit within a pool.
type QuotaUnit struct {
	UnitName string `json:"unitName" binding:"required,dns1123,max=40"`
	Quantity int    `json:"quantity" binding:"min=0"`
}

// Quota is a tenant's resource allocation in one pool, expressed as counts of
// named resource units (product model: pool -> resource unit x quantity).
// Platform derives the backing Koordinator ElasticQuota from
// sum(unit.requests * quantity).
type Quota struct {
	Pool  string      `json:"pool" binding:"dns1123,max=40"`
	Units []QuotaUnit `json:"units"`
}

// QuotaUnitStatus is the live usage of one allocated resource unit.
type QuotaUnitStatus struct {
	UnitName string `json:"unitName"`
	Quantity int    `json:"quantity"`
	Used     int    `json:"used,omitempty"`
}

// QuotaStatus is the live usage of one pool's quota.
type QuotaStatus struct {
	Pool  string            `json:"pool"`
	Units []QuotaUnitStatus `json:"units,omitempty"`
}

// QuotaList is a tenant's per-pool quotas plus their live statuses.
type QuotaList struct {
	Items    []Quota       `json:"items"`
	Statuses []QuotaStatus `json:"statuses,omitempty"`
	Count    int           `json:"count" binding:"min=0"`
}

// QuotaCreateRequest sets a pool's quota for a tenant.
type QuotaCreateRequest struct {
	Pool  string      `json:"pool" binding:"required,dns1123,max=40"`
	Units []QuotaUnit `json:"units" binding:"required"`
}

// QuotaPatchRequest replaces a pool quota's unit allocations.
type QuotaPatchRequest struct {
	Units []QuotaUnit `json:"units" binding:"required"`
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

// Tenant is the Platform-facing Tenant DTO. Platform owns the durable tenant
// record (this row); the live Kubernetes materialization — Namespace,
// ElasticQuota, per-tenant init resources — is owned by cluster-manager via
// the Tenant CR, which Platform drives over REST and never touches directly.
//
// Identifier is the stable logical tenant scope used by Platform, compute and
// artifacts. KubernetesNamespace is the physical namespace selected for
// Tenant-owned Kubernetes resources and may be shared by multiple tenants.
// Phase / Status are read live from cluster-manager (not cached); Suspended is
// Platform-owned and enforced at the workload-create entry point.
type Tenant struct {
	Identifier          string        `json:"identifier"`
	KubernetesNamespace string        `json:"kubernetesNamespace"`
	DisplayName         string        `json:"displayName"`
	Description         string        `json:"description,omitempty"`
	Owner               string        `json:"owner,omitempty"`
	Labels              StringMap     `json:"labels,omitempty"`
	Annotations         StringMap     `json:"annotations,omitempty"`
	Quotas              []Quota       `json:"quotas,omitempty"`
	InitResources       InitResources `json:"initResources,omitempty"`
	Phase               TenantPhase   `json:"phase"`
	Status              TenantStatus  `json:"status,omitempty"`
	Suspended           bool          `json:"suspended"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
}

// TenantStatus is the tenant status sub-object (phase lives on Tenant).
type TenantStatus struct {
	Message    string        `json:"message,omitempty"`
	Conditions []Condition   `json:"conditions,omitempty"`
	Quotas     []QuotaStatus `json:"quotas,omitempty"`
}

// TenantList is a page of Tenant. Partial is set when a cross-tenant list could
// not enrich every row with live status (§5.3).
type TenantList struct {
	Items         []Tenant `json:"items"`
	Count         int      `json:"count" binding:"min=0"`
	ContinueToken string   `json:"continueToken,omitempty"`
	Partial       bool     `json:"partial,omitempty"`
}

// TenantCreateRequest is the body of POST /tenants. Identifier becomes the
// cluster-manager Tenant CR name and logical tenant scope. KubernetesNamespace
// selects the physical namespace and may be shared. InitialAdmin seeds the
// first tenant-admin member, by email or username.
type TenantCreateRequest struct {
	Identifier          string        `json:"identifier" binding:"required,dns1123,min=3,max=40"`
	KubernetesNamespace string        `json:"kubernetesNamespace" binding:"required,dns1123,max=63"`
	DisplayName         string        `json:"displayName" binding:"required,min=1,max=100"`
	Description         string        `json:"description,omitempty" binding:"max=1000"`
	InitialAdmin        string        `json:"initialAdmin" binding:"required"`
	Labels              StringMap     `json:"labels,omitempty"`
	Annotations         StringMap     `json:"annotations,omitempty"`
	Quotas              []Quota       `json:"quotas,omitempty"`
	InitResources       InitResources `json:"initResources,omitempty"`
}

// TenantPatchRequest is the JSON Merge Patch body of PATCH /tenants/{name}.
// Only display metadata is editable here; quotas are managed via the quota
// sub-resource endpoints.
type TenantPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"min=1,max=100"`
	Description string    `json:"description,omitempty" binding:"max=1000"`
	Labels      StringMap `json:"labels,omitempty"`
	Annotations StringMap `json:"annotations,omitempty"`
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

// MemberCreateRequest binds a user to a tenant role. Account is the invitee's
// email or username (invites an existing platform user).
type MemberCreateRequest struct {
	Account  string         `json:"account" binding:"required"`
	RoleName MemberRoleName `json:"roleName" binding:"required"`
}

// MemberPatchRequest changes a member's role.
type MemberPatchRequest struct {
	RoleName MemberRoleName `json:"roleName" binding:"required"`
}
