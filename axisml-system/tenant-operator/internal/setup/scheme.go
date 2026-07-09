// Package setup wires the tenant-operator scheme so the production
// cmd/main.go and the L1 integration TestMain stay in sync.
package setup

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	schedulingv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/scheduling/v1alpha1"
)

// AddToScheme registers every API group the tenant-operator reconciles.
// Call once at process start before constructing the manager.
//
// clientgoscheme already covers core, apps, rbac, batch, coordination — we
// don't re-register them explicitly. ElasticQuota comes from the in-repo
// scheduler-plugins API copy (group scheduling.x-k8s.io).
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))
}
