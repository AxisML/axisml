package server

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// Tenant is a cluster-scoped CR owned by cluster-manager (REST writer) and
// reconciled by tenant-operator into a Namespace + ElasticQuota + per-tenant
// init resources. cluster-manager holds no PG: the CR in etcd is the only
// store. The tenant's `name` is the single canonical identifier — it is the
// CR metadata.name, the K8s namespace, and the compute/artifacts partition
// string all at once.
//
// Quotas are expressed in the business form (per pool, a list of
// `unit × quantity` selections). cluster-manager owns the ResourcePool /
// ResourceUnit vocabulary, so it folds `Σ(unit.requests × quantity)` /
// `Σ(unit.limits × quantity)` into the ElasticQuota min/max written to the
// CR spec.quotas[]. The original selection round-trips through the
// `axisml.io/quotas` annotation so GET can return the business form.

// TenantDTO is the REST representation of a Tenant CR.
type TenantDTO struct {
	Name            string                        `json:"name"`
	Namespace       tenantv1alpha1.NamespaceSpec  `json:"namespace"`
	Quotas          []QuotaDTO                    `json:"quotas"`
	InitResources   *tenantv1alpha1.InitResources `json:"initResources,omitempty"`
	Labels          map[string]string             `json:"labels,omitempty"`
	Annotations     map[string]string             `json:"annotations,omitempty"`
	ResourceVersion string                        `json:"resourceVersion,omitempty"`
	Phase           string                        `json:"phase,omitempty"`
	Status          *TenantStatusDTO              `json:"status,omitempty"`
	CreatedAt       time.Time                     `json:"createdAt"`
}

// QuotaDTO is the business form of one pool's quota: a list of
// `unit × quantity` selections under that pool.
type QuotaDTO struct {
	Pool  string         `json:"pool"`
	Units []QuotaUnitDTO `json:"units"`
}

// QuotaUnitDTO selects a ResourceUnit and how many of it the tenant is
// granted under the pool.
type QuotaUnitDTO struct {
	UnitName string `json:"unitName"`
	Quantity int    `json:"quantity"`
}

// TenantStatusDTO surfaces the operator-written CR status, read live from
// etcd on every GET (no cache). `used` flows from koord-scheduler through
// the ElasticQuota and is never persisted anywhere but the CR.
type TenantStatusDTO struct {
	ObservedGeneration int64            `json:"observedGeneration,omitempty"`
	Phase              string           `json:"phase,omitempty"`
	Message            string           `json:"message,omitempty"`
	NamespaceReady     bool             `json:"namespaceReady,omitempty"`
	Quotas             []QuotaStatusDTO `json:"quotas,omitempty"`
}

// QuotaStatusDTO is the per-pool quota readiness + live usage.
type QuotaStatusDTO struct {
	Pool  string              `json:"pool"`
	Ready bool                `json:"ready"`
	Used  corev1.ResourceList `json:"used,omitempty"`
}

// CreateTenantRequest is the body for POST /api/v1/tenants.
type CreateTenantRequest struct {
	Name          string                        `json:"name"`
	Namespace     *tenantv1alpha1.NamespaceSpec `json:"namespace,omitempty"`
	Quotas        []QuotaDTO                    `json:"quotas,omitempty"`
	InitResources *tenantv1alpha1.InitResources `json:"initResources,omitempty"`
	Labels        map[string]string             `json:"labels,omitempty"`
	Annotations   map[string]string             `json:"annotations,omitempty"`
}

// PatchTenantRequest covers the tenant-level mutable fields. `name` and
// `spec.namespace.name` are immutable; quotas mutate via the sub-routes.
type PatchTenantRequest struct {
	NamespaceLabels      map[string]string             `json:"namespaceLabels,omitempty"`
	NamespaceAnnotations map[string]string             `json:"namespaceAnnotations,omitempty"`
	InitResources        *tenantv1alpha1.InitResources `json:"initResources,omitempty"`
	Labels               map[string]string             `json:"labels,omitempty"`
	Annotations          map[string]string             `json:"annotations,omitempty"`
}

// SetQuotaRequest is the body for POST /api/v1/tenants/{tenant}/quotas — it
// creates or replaces the quota for one pool.
type SetQuotaRequest struct {
	Pool  string         `json:"pool"`
	Units []QuotaUnitDTO `json:"units"`
}

// PatchQuotaRequest is the body for PATCH .../quotas/{pool}; `pool` is the
// path param.
type PatchQuotaRequest struct {
	Units []QuotaUnitDTO `json:"units"`
}

// TenantList is the LIST response.
type TenantList struct {
	Items         []TenantDTO `json:"items"`
	Count         int         `json:"count"`
	ContinueToken string      `json:"continueToken,omitempty"`
}

// QuotaList is the LIST response for a tenant's quotas.
type QuotaList struct {
	Items []QuotaDTO `json:"items"`
	Count int        `json:"count"`
}

// QuotasAnnotation stores the business-form quota selection (JSON-encoded
// []QuotaDTO) on the Tenant CR so GET can round-trip `unit × quantity`
// after the spec.quotas[] has been folded to ElasticQuota min/max.
const QuotasAnnotation = "axisml.io/quotas"
