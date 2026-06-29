// Package setup wires the tenant-operator scheme so the production
// cmd/main.go and the L1 integration TestMain stay in sync.
package setup

import (
	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// AddToScheme registers every API group the tenant-operator reconciles.
// Call once at process start before constructing the manager.
//
// clientgoscheme already covers core, apps, rbac, batch, coordination — we
// don't re-register them explicitly. ElasticQuota comes from the
// scheduler-plugins API in Koordinator's vendored apis tree.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))
}
