// Package reconcile holds the per-resource reconciliation logic that the
// Tenant controller dispatches to. Each subreconciler returns per-item
// readiness so the top-level controller can aggregate it into Tenant.status.
package reconcile

import (
	"fmt"

	axisml "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
)

// Naming convention prefixes from design §3.1 and §6.2.
const (
	tenantResourcePrefix = "axisml-tenant-"
	elasticQuotaPrefix   = "axisml-"
)

// PerTenantResourceName builds a name like axisml-tenant-<tenant>-<sub>.
// Used for ImagePullSecrets / Secrets / ConfigMaps / ServiceAccounts /
// Roles / RoleBindings (§6.3–§6.6).
func PerTenantResourceName(tenantName, sub string) string {
	return fmt.Sprintf("%s%s-%s", tenantResourcePrefix, tenantName, sub)
}

// ElasticQuotaName builds the cluster-unique ElasticQuota name following
// design §6.2: axisml-<tenant>-<pool>-<quota>.
func ElasticQuotaName(tenantName, pool, quota string) string {
	return fmt.Sprintf("%s%s-%s-%s", elasticQuotaPrefix, tenantName, pool, quota)
}

// TenantLabels returns the common labels every per-tenant resource carries.
func TenantLabels(t *axisml.Tenant) map[string]string {
	return map[string]string{
		axisml.LabelTenantID:  t.Labels[axisml.LabelTenantID],
		axisml.LabelManagedBy: axisml.ManagedByValue,
	}
}

// ApplyTenantLabels merges TenantLabels into an existing label map.
func ApplyTenantLabels(t *axisml.Tenant, m map[string]string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	for k, v := range TenantLabels(t) {
		m[k] = v
	}
	return m
}
