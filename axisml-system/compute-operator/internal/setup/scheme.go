// Package setup wires the compute-operator scheme + handler-stub
// registration so the production cmd/main.go and the integration test
// TestMain stay in sync.
package setup

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
	mlservicehandler "github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler"
	mltrafficpolicyhandler "github.com/axisml/axisml/components/compute-operator/internal/mltrafficpolicy/handler"
)

// AddToScheme registers every API group the compute-operator reconciles
// plus the deferred MLService handler stubs. Call once at process start
// before constructing the manager.
//
// clientgoscheme already covers core, apps, rbac, batch, coordination — we
// don't re-register them explicitly.
func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gwapiv1.Install(scheme))
	utilruntime.Must(mlrunv1alpha1.AddToScheme(scheme))
	utilruntime.Must(mlservicev1alpha1.AddToScheme(scheme))
	utilruntime.Must(mltrafficpolicyv1alpha1.AddToScheme(scheme))

	mlservicehandler.RegisterStubs()
	mltrafficpolicyhandler.RegisterStubs()
}
