// Package setup wires shared scheme + handler-stub registration so the
// production cmd/main.go and the merged envtest TestMain stay in sync.
// Adding a new API group should require touching this file only.
package setup

import (
	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
	mlservicehandler "github.com/axisml/axisml/components/operator/internal/mlservice/handler"
)

// AddToScheme registers every API group the merged operator reconciles plus
// the deferred MLService handler stubs. Call once at process start before
// constructing the manager.
//
// clientgoscheme already covers core, apps, rbac, batch, coordination — we
// don't re-register them explicitly.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(gwapiv1.Install(scheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))
	utilruntime.Must(mljobv1alpha1.AddToScheme(scheme))
	utilruntime.Must(mlservicev1alpha1.AddToScheme(scheme))

	mlservicehandler.RegisterStubs()
}
