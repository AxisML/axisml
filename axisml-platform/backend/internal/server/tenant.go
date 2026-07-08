package server

import "time"

// QuotaUnit allocates a quantity of one named resource unit within a pool.
type QuotaUnit struct {
	UnitName string `json:"unitName" binding:"required,dns1123,max=40" desc:"Resource unit (shape) name within the pool."`
	Quantity int    `json:"quantity" binding:"min=0" desc:"Number of units of this shape allocated to the tenant."`
}

// QuotaResources is a pool quota expressed directly as ElasticQuota min/max
// resource quantities (Kubernetes resource.Quantity strings, e.g. "8",
// "256Gi"). Mutually exclusive with the units business form.
type QuotaResources struct {
	Min map[string]string `json:"min,omitempty" desc:"ElasticQuota minimum resources (resource.Quantity strings)."`
	Max map[string]string `json:"max" desc:"ElasticQuota maximum resources (resource.Quantity strings)."`
}

// Quota is a tenant's resource allocation in one pool, in exactly one of two
// forms: the units business form (pool -> resource unit x quantity, from which
// Platform derives the backing ElasticQuota via sum(unit.requests * quantity)),
// or direct min/max resources.
type Quota struct {
	Pool  string          `json:"pool" binding:"dns1123,max=40" desc:"Resource pool the quota allocates from."`
	Units []QuotaUnit     `json:"units,omitempty" desc:"Per-unit allocations within the pool. Mutually exclusive with quota."`
	Quota *QuotaResources `json:"quota,omitempty" desc:"Direct min/max resources for the pool. Mutually exclusive with units."`
}

// QuotaUnitStatus is the live usage of one allocated resource unit.
type QuotaUnitStatus struct {
	UnitName string `json:"unitName" desc:"Resource unit (shape) name."`
	Quantity int    `json:"quantity" desc:"Number of units allocated to the tenant."`
	Used     int    `json:"used,omitempty" desc:"Number of units currently in use."`
}

// QuotaStatus is the live usage of one pool's quota.
type QuotaStatus struct {
	Pool  string            `json:"pool" desc:"Resource pool the status refers to."`
	Units []QuotaUnitStatus `json:"units,omitempty" desc:"Live per-unit usage within the pool."`
}

// QuotaList is a tenant's per-pool quotas plus their live statuses.
type QuotaList struct {
	Items    []Quota       `json:"items" desc:"Per-pool quota allocations for the tenant."`
	Statuses []QuotaStatus `json:"statuses,omitempty" desc:"Live per-pool usage matching the items."`
	Count    int           `json:"count" binding:"min=0" desc:"Number of pool quotas in this list."`
}

// QuotaCreateRequest sets a pool's quota for a tenant, in either the units
// business form or direct min/max mode (exactly one; validated downstream).
type QuotaCreateRequest struct {
	Pool  string          `json:"pool" binding:"required,dns1123,max=40" desc:"Resource pool to set quota for."`
	Units []QuotaUnit     `json:"units,omitempty" desc:"Per-unit allocations to grant in the pool. Mutually exclusive with quota."`
	Quota *QuotaResources `json:"quota,omitempty" desc:"Direct min/max resources for the pool. Mutually exclusive with units."`
}

// QuotaPatchRequest replaces a pool quota's input, in either the units business
// form or direct min/max mode (exactly one; validated downstream).
type QuotaPatchRequest struct {
	Units []QuotaUnit     `json:"units,omitempty" desc:"Replacement per-unit allocations for the pool. Mutually exclusive with quota."`
	Quota *QuotaResources `json:"quota,omitempty" desc:"Replacement direct min/max resources for the pool. Mutually exclusive with units."`
}

// SecretSourceRef references a Secret in another namespace.
type SecretSourceRef struct {
	Namespace string `json:"namespace" desc:"Namespace holding the source Secret."`
	Name      string `json:"name" desc:"Name of the source Secret."`
}

// ConfigMapSourceRef references a ConfigMap in another namespace.
type ConfigMapSourceRef struct {
	Namespace string `json:"namespace" desc:"Namespace holding the source ConfigMap."`
	Name      string `json:"name" desc:"Name of the source ConfigMap."`
}

// ImagePullSecretInit seeds a per-tenant image pull secret.
type ImagePullSecretInit struct {
	Name            string          `json:"name" desc:"Name of the image pull Secret to create in the tenant namespace."`
	SourceSecretRef SecretSourceRef `json:"sourceSecretRef" desc:"Source Secret to copy the pull credentials from."`
}

// SecretInit seeds a per-tenant Secret.
type SecretInit struct {
	Name            string          `json:"name" desc:"Name of the Secret to create in the tenant namespace."`
	Type            string          `json:"type,omitempty" desc:"Kubernetes Secret type (e.g. Opaque)."`
	SourceSecretRef SecretSourceRef `json:"sourceSecretRef" desc:"Source Secret to copy the data from."`
}

// ConfigMapInit seeds a per-tenant ConfigMap.
type ConfigMapInit struct {
	Name               string             `json:"name" desc:"Name of the ConfigMap to create in the tenant namespace."`
	SourceConfigMapRef ConfigMapSourceRef `json:"sourceConfigMapRef" desc:"Source ConfigMap to copy the data from."`
}

// ServiceAccountInit seeds a per-tenant ServiceAccount.
type ServiceAccountInit struct {
	Name             string         `json:"name" desc:"Name of the ServiceAccount to create in the tenant namespace."`
	ImagePullSecrets []string       `json:"imagePullSecrets,omitempty" desc:"Image pull Secret names attached to the ServiceAccount."`
	RBAC             map[string]any `json:"rbac,omitempty" desc:"RBAC rules to bind to the ServiceAccount."`
}

// InitResources is the set of bootstrap objects created in a tenant namespace.
type InitResources struct {
	ImagePullSecrets []ImagePullSecretInit `json:"imagePullSecrets,omitempty" desc:"Image pull Secrets to seed in the tenant namespace."`
	Secrets          []SecretInit          `json:"secrets,omitempty" desc:"Secrets to seed in the tenant namespace."`
	ConfigMaps       []ConfigMapInit       `json:"configMaps,omitempty" desc:"ConfigMaps to seed in the tenant namespace."`
	ServiceAccounts  []ServiceAccountInit  `json:"serviceAccounts,omitempty" desc:"ServiceAccounts to seed in the tenant namespace."`
}

// Tenant is the Platform-facing Tenant type. Platform owns the durable tenant
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
	Identifier          string        `json:"identifier" desc:"Stable logical tenant scope used across Platform, compute and artifacts."`
	KubernetesNamespace string        `json:"kubernetesNamespace" desc:"Physical Kubernetes namespace backing the tenant (may be shared)."`
	DisplayName         string        `json:"displayName" desc:"Human-readable tenant name."`
	Description         string        `json:"description,omitempty" desc:"Free-text tenant description."`
	Owner               string        `json:"owner,omitempty" desc:"Username of the tenant owner."`
	Labels              StringMap     `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations         StringMap     `json:"annotations,omitempty" desc:"User-defined annotations."`
	Quotas              []Quota       `json:"quotas,omitempty" desc:"Per-pool resource quota allocations."`
	InitResources       InitResources `json:"initResources,omitempty" desc:"Bootstrap objects seeded into the tenant namespace."`
	Phase               TenantPhase   `json:"phase" desc:"Current tenant lifecycle phase (read live from cluster-manager)."`
	Status              TenantStatus  `json:"status,omitempty" desc:"Live tenant status detail."`
	Suspended           bool          `json:"suspended" desc:"Whether new workloads are blocked (Platform-enforced)."`
	// Live workload roll-ups, best-effort. Populated on getTenant and on
	// listTenants?stats=true; left at zero when the compute-service enrichment
	// is skipped (e.g. the tenant-switcher list) or temporarily unavailable.
	// ActiveJobRuns + ActiveExperimentRuns is the tenant's "active task" count.
	MemberCount          int       `json:"memberCount" desc:"Number of members bound to the tenant."`
	ActiveJobRuns        int       `json:"activeJobRuns" desc:"Number of active job runs in the tenant."`
	ActiveExperimentRuns int       `json:"activeExperimentRuns" desc:"Number of active experiment runs in the tenant."`
	OnlineServices       int       `json:"onlineServices" desc:"Number of online ML services in the tenant."`
	CreatedAt            time.Time `json:"createdAt" desc:"Time the tenant was created."`
	UpdatedAt            time.Time `json:"updatedAt" desc:"Time the tenant was last updated."`
}

// TenantStatus is the tenant status sub-object (phase lives on Tenant).
type TenantStatus struct {
	Message    string        `json:"message,omitempty" desc:"Human-readable status detail for the tenant."`
	Conditions []Condition   `json:"conditions,omitempty" desc:"Live status conditions reported by cluster-manager."`
	Quotas     []QuotaStatus `json:"quotas,omitempty" desc:"Live per-pool quota usage."`
}

// TenantList is a page of Tenant. Partial is set when a cross-tenant list could
// not enrich every row with live status (§5.3).
type TenantList struct {
	Items         []Tenant `json:"items" desc:"Tenants in this page."`
	Count         int      `json:"count" binding:"min=0" desc:"Number of tenants in this page."`
	ContinueToken string   `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
	Partial       bool     `json:"partial,omitempty" desc:"True if some rows could not be enriched with live status."`
}

// TenantCreateRequest is the body of POST /tenants. Identifier becomes the
// cluster-manager Tenant CR name and logical tenant scope. KubernetesNamespace
// selects the physical namespace and may be shared. InitialAdmin seeds the
// first tenant-admin member, by email or username.
type TenantCreateRequest struct {
	Identifier          string        `json:"identifier" binding:"required,dns1123,min=3,max=40" desc:"Stable logical tenant scope (becomes the Tenant CR name)."`
	KubernetesNamespace string        `json:"kubernetesNamespace" binding:"required,dns1123,max=63" desc:"Physical Kubernetes namespace to back the tenant (may be shared)."`
	DisplayName         string        `json:"displayName" binding:"required,min=1,max=100" desc:"Human-readable tenant name."`
	Description         string        `json:"description,omitempty" binding:"max=1000" desc:"Free-text tenant description."`
	InitialAdmin        string        `json:"initialAdmin" binding:"required" desc:"Email or username of the first tenant-admin member."`
	Labels              StringMap     `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations         StringMap     `json:"annotations,omitempty" desc:"User-defined annotations."`
	Quotas              []Quota       `json:"quotas,omitempty" desc:"Initial per-pool resource quota allocations."`
	InitResources       InitResources `json:"initResources,omitempty" desc:"Bootstrap objects to seed into the tenant namespace."`
}

// TenantPatchRequest is the JSON Merge Patch body of PATCH /tenants/{name}.
// Only display metadata is editable here; quotas are managed via the quota
// sub-resource endpoints.
type TenantPatchRequest struct {
	DisplayName string    `json:"displayName,omitempty" binding:"min=1,max=100" desc:"Updated human-readable tenant name."`
	Description string    `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text tenant description."`
	Labels      StringMap `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations StringMap `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// Member is a user↔role binding within a tenant.
type Member struct {
	UserID      UUID      `json:"userId" desc:"User ID of the member."`
	Username    string    `json:"username" desc:"Username of the member."`
	DisplayName string    `json:"displayName,omitempty" desc:"Human-readable name of the member."`
	Email       Email     `json:"email,omitempty" desc:"Email address of the member."`
	RoleName    RoleName  `json:"roleName" desc:"Role granted to the member within the tenant."`
	AddedAt     time.Time `json:"addedAt" desc:"Time the member was added to the tenant."`
}

// MemberList is a page of Member.
type MemberList struct {
	Items         []Member `json:"items" desc:"Members in this page."`
	Count         int      `json:"count" binding:"min=0" desc:"Number of members in this page."`
	ContinueToken string   `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
}

// MemberCreateRequest binds a user to a tenant role. Account is the invitee's
// email or username (invites an existing platform user).
type MemberCreateRequest struct {
	Account  string         `json:"account" binding:"required" desc:"Email or username of the existing platform user to add."`
	RoleName MemberRoleName `json:"roleName" binding:"required" desc:"Role to grant the member (tenant-admin or user)."`
}

// MemberPatchRequest changes a member's role.
type MemberPatchRequest struct {
	RoleName MemberRoleName `json:"roleName" binding:"required" desc:"New role for the member (tenant-admin or user)."`
}
