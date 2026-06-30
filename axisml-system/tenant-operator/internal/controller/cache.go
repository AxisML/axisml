package controller

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	schedulingv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/scheduling/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// CacheByObject returns per-type cache restrictions for the resources Tenant
// owns. Without these, the manager would set up cluster-wide informers for
// every Secret/ConfigMap, pulling them all into memory. Source Secret/ConfigMap
// reads bypass the cache via the APIReader.
//
// The filter must NOT be applied as a global default selector — that would
// also filter MLRun/MLService informers (Job, Deployment, PodGroup, HTTPRoute),
// none of which carry the tenant managed-by label.
//
// PodGroup (also in scheduling.x-k8s.io/v1alpha1) is not in the map: it's
// owned by MLRun and not labelled by Tenant.
func CacheByObject() map[client.Object]cache.ByObject {
	managedByOnly := labels.SelectorFromSet(labels.Set{
		tenantv1alpha1.LabelManagedBy: tenantv1alpha1.ManagedByValue,
	})
	return map[client.Object]cache.ByObject{
		&corev1.Secret{}:                   {Label: managedByOnly},
		&corev1.ConfigMap{}:                {Label: managedByOnly},
		&corev1.ServiceAccount{}:           {Label: managedByOnly},
		&rbacv1.Role{}:                     {Label: managedByOnly},
		&rbacv1.RoleBinding{}:              {Label: managedByOnly},
		&schedulingv1alpha1.ElasticQuota{}: {Label: managedByOnly},
	}
}
