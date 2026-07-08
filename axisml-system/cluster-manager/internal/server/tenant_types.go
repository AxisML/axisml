package server

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// Tenant is a cluster-scoped CR owned by cluster-manager (REST writer) and
// reconciled by tenant-operator into a Namespace + ElasticQuota + per-tenant
// init resources. cluster-manager holds no PG: the CR in etcd is the only
// store. The tenant's `name` is the single canonical identifier — it is the
// CR metadata.name, the K8s namespace, and the compute/artifacts partition
// string all at once.
//
// Quotas accept either the business form (per pool, a list of `unit × quantity`
// selections) or direct ElasticQuota min/max resources. cluster-manager compiles
// both forms into the single canonical Tenant CR shape: spec.quotas[].min/max.
// Unit selections round-trip through the tenant.axisml.io/quotas annotation so
// GET can return the business form without making it part of the CR contract.

// Tenant is the REST representation of a Tenant CR.
type Tenant struct {
	Name            string                        `json:"name" desc:"Canonical tenant identifier; also the CR name, K8s namespace, and partition string."`
	Namespace       tenantv1alpha1.NamespaceSpec  `json:"namespace" desc:"Backing namespace specification for the tenant."`
	Quotas          []Quota                       `json:"quotas" desc:"Per-pool quotas. Each item is returned either as units (business form) or quota (direct min/max form)."`
	InitResources   *tenantv1alpha1.InitResources `json:"initResources,omitempty" desc:"Per-tenant init resources (Secrets, ConfigMaps, ServiceAccount, RBAC) seeded on provisioning."`
	Labels          map[string]string             `json:"labels,omitempty" desc:"User-defined labels on the tenant."`
	Annotations     map[string]string             `json:"annotations,omitempty" desc:"User-defined annotations on the tenant."`
	ResourceVersion string                        `json:"resourceVersion,omitempty" desc:"Opaque CR resourceVersion for optimistic concurrency."`
	Phase           string                        `json:"phase,omitempty" desc:"High-level provisioning phase of the tenant."`
	Status          *TenantStatus                 `json:"status,omitempty" desc:"Live operator-written status read from the CR."`
	CreatedAt       time.Time                     `json:"createdAt" desc:"Tenant creation timestamp (RFC3339)."`
}

// Quota is one pool quota in exactly one of two input forms:
//   - units: business selection compiled against ResourcePool.spec.units[].
//   - quota: direct ElasticQuota min/max resources.
//
// The Tenant CR always stores the compiled direct form in spec.quotas[].min/max.
type Quota struct {
	Pool  string          `json:"pool" desc:"ResourcePool this quota applies to."`
	Units []QuotaUnit     `json:"units,omitempty" desc:"Unit × quantity selections granted to the tenant under this pool. Mutually exclusive with quota."`
	Quota *QuotaResources `json:"quota,omitempty" desc:"Direct min/max resources for this pool. Mutually exclusive with units."`
}

// MarshalJSON preserves the distinction between an omitted units field (direct
// quota mode) and an explicit empty units array (zero quota in units mode).
func (q Quota) MarshalJSON() ([]byte, error) {
	if q.Units != nil {
		type unitsQuota struct {
			Pool  string      `json:"pool"`
			Units []QuotaUnit `json:"units"`
		}
		return json.Marshal(unitsQuota{Pool: q.Pool, Units: q.Units})
	}
	type directQuota struct {
		Pool  string          `json:"pool"`
		Quota *QuotaResources `json:"quota,omitempty"`
	}
	return json.Marshal(directQuota{Pool: q.Pool, Quota: q.Quota})
}

// QuotaUnit selects a ResourceUnit and how many of it the tenant is
// granted under the pool.
type QuotaUnit struct {
	UnitName string `json:"unitName" desc:"Name of the ResourceUnit being granted."`
	Quantity int    `json:"quantity" desc:"How many of this unit the tenant is granted under the pool."`
}

// QuotaResources is the direct scheduler-facing quota shape. It maps 1:1 to the
// Tenant CR's spec.quotas[].min/max fields.
type QuotaResources struct {
	Min corev1.ResourceList `json:"min,omitempty" desc:"ElasticQuota minimum resources."`
	Max corev1.ResourceList `json:"max" desc:"ElasticQuota maximum resources."`
}

// TenantStatus surfaces the operator-written CR status, read live from
// etcd on every GET (no cache). `used` flows from axisml-scheduler through
// the ElasticQuota and is never persisted anywhere but the CR.
type TenantStatus struct {
	ObservedGeneration int64         `json:"observedGeneration,omitempty" desc:"Generation of the spec the operator last reconciled."`
	Phase              string        `json:"phase,omitempty" desc:"Current reconciliation phase reported by the operator."`
	Message            string        `json:"message,omitempty" desc:"Human-readable detail about the current phase."`
	NamespaceReady     bool          `json:"namespaceReady,omitempty" desc:"Whether the tenant's namespace has been provisioned."`
	Quotas             []QuotaStatus `json:"quotas,omitempty" desc:"Per-pool quota readiness and live usage."`
}

// QuotaStatus is the per-pool quota readiness + live usage.
type QuotaStatus struct {
	Pool  string              `json:"pool" desc:"ResourcePool this quota status applies to."`
	Ready bool                `json:"ready" desc:"Whether the ElasticQuota for this pool is provisioned and ready."`
	Used  corev1.ResourceList `json:"used,omitempty" desc:"Live resource usage from axisml-scheduler via the ElasticQuota."`
}

// CreateTenantRequest is the body for POST /api/v1/tenants.
type CreateTenantRequest struct {
	Name          string                        `json:"name" desc:"Tenant identifier to create; becomes the CR name, namespace, and partition string."`
	Namespace     *tenantv1alpha1.NamespaceSpec `json:"namespace,omitempty" desc:"Optional namespace specification; defaults are derived from the tenant name when omitted."`
	Quotas        []Quota                       `json:"quotas,omitempty" desc:"Initial per-pool quotas to grant the tenant. Each item must use either units or quota."`
	InitResources *tenantv1alpha1.InitResources `json:"initResources,omitempty" desc:"Per-tenant init resources to seed on provisioning."`
	Labels        map[string]string             `json:"labels,omitempty" desc:"User-defined labels to set on the tenant."`
	Annotations   map[string]string             `json:"annotations,omitempty" desc:"User-defined annotations to set on the tenant."`
}

// PatchTenantRequest covers the tenant-level mutable fields. `name` and
// `spec.namespace.name` are immutable; quotas mutate via the sub-routes.
type PatchTenantRequest struct {
	NamespaceLabels      map[string]string             `json:"namespaceLabels,omitempty" desc:"Replacement labels applied to the tenant's namespace."`
	NamespaceAnnotations map[string]string             `json:"namespaceAnnotations,omitempty" desc:"Replacement annotations applied to the tenant's namespace."`
	InitResources        *tenantv1alpha1.InitResources `json:"initResources,omitempty" desc:"Replacement per-tenant init resources."`
	Labels               map[string]string             `json:"labels,omitempty" desc:"Replacement labels for the tenant."`
	Annotations          map[string]string             `json:"annotations,omitempty" desc:"Replacement annotations for the tenant."`
}

// SetQuotaRequest is the body for POST /api/v1/tenants/{tenant}/quotas — it
// creates or replaces the quota for one pool.
type SetQuotaRequest struct {
	Pool  string          `json:"pool" desc:"ResourcePool to create or replace the quota for."`
	Units []QuotaUnit     `json:"units,omitempty" desc:"Unit × quantity selections that make up the pool quota. Mutually exclusive with quota."`
	Quota *QuotaResources `json:"quota,omitempty" desc:"Direct min/max resources for the pool quota. Mutually exclusive with units."`
}

// PatchQuotaRequest is the body for PATCH .../quotas/{pool}; `pool` is the
// path param.
type PatchQuotaRequest struct {
	Units []QuotaUnit     `json:"units,omitempty" desc:"Replacement unit × quantity selections for the pool quota. Mutually exclusive with quota."`
	Quota *QuotaResources `json:"quota,omitempty" desc:"Replacement direct min/max resources for the pool quota. Mutually exclusive with units."`
}

// TenantList is the LIST response.
type TenantList struct {
	Items         []Tenant `json:"items" desc:"Page of tenants."`
	Count         int      `json:"count" desc:"Number of tenants in this page."`
	ContinueToken string   `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page; empty when no more pages."`
}

// QuotaList is the LIST response for a tenant's quotas.
type QuotaList struct {
	Items []Quota `json:"items" desc:"The tenant's per-pool quotas."`
	Count int     `json:"count" desc:"Number of quotas returned."`
}

// QuotasAnnotation stores the business-form quota selection (JSON-encoded
// []Quota) on the Tenant CR so GET can round-trip `unit × quantity`
// after the spec.quotas[] has been folded to ElasticQuota min/max.
const QuotasAnnotation = "tenant.axisml.io/quotas"
