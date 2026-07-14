package app

import (
	"fmt"

	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/config"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/kuberuntime"
	jobmod "github.com/axisml/axisml/axisml-system/compute-service/internal/mlrun"
	servicemod "github.com/axisml/axisml/axisml-system/compute-service/internal/mlservice"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/poolcache"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/tenantresolver"
	trafficpolicymod "github.com/axisml/axisml/axisml-system/compute-service/internal/trafficpolicy"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
	computemodule "github.com/axisml/axisml/axisml-system/compute-service/pkg/module"
)

// BuildModules is the Kubernetes composition root. It derives the form-neutral
// dependencies from the controller-runtime manager — the ComputeRuntime adapter
// (typed client + clientset) and the ResourcePool informer-cache catalog — hands
// them to the shared pkg/module assembly, then appends the Kubernetes-specific
// status reflow (apiserver informers, design §4.2) as additional runnables.
func BuildModules(
	gormDB *gorm.DB,
	mgr manager.Manager,
	log logr.Logger,
	metrics extensions.MetricsProvider,
	workloadTenantPrefix bool,
) ([]server.Module, []manager.Runnable, server.Capabilities, error) {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, nil, server.Capabilities{}, fmt.Errorf("build clientset: %w", err)
	}

	mod, err := computemodule.New(computemodule.Deps{
		DB: gormDB,
		Runtime: kuberuntime.New(mgr.GetClient(), clientset, kuberuntime.Options{
			NamespaceResolver:    tenantresolver.New(mgr.GetAPIReader()),
			WorkloadTenantPrefix: workloadTenantPrefix,
		}),
		Resolver:          poolcache.New(mgr.GetClient()),
		Metrics:           metrics,
		Log:               log,
		ReconcileInterval: config.ReconcileInterval,
		// Kubernetes composition root: axisml-scheduler admits pods against the
		// tenant ElasticQuota, so quota enforcement is real.
		RuntimeName:          "kubernetes",
		QuotaEnforcement:     true,
		WorkloadTenantPrefix: workloadTenantPrefix,
	})
	if err != nil {
		return nil, nil, server.Capabilities{}, err
	}

	modules := make([]server.Module, 0, len(mod.Routes()))
	for _, r := range mod.Routes() {
		modules = append(modules, r)
	}

	runnables := make([]manager.Runnable, 0, len(mod.Runnables())+3)
	for _, r := range mod.Runnables() {
		runnables = append(runnables, r)
	}
	// Kubernetes status reflow: apiserver informers mirror CR status into PG
	// (leader-only). The Lite form replaces these with a runtime Observe poll.
	runnables = append(runnables,
		jobmod.NewInformer(gormDB, mgr, log.WithName("mlrun-informer")),
		servicemod.NewInformer(gormDB, mgr, log.WithName("mlservice-informer")),
		trafficpolicymod.NewInformer(gormDB, mgr, log.WithName("traffic-policy-informer")),
	)

	return modules, runnables, mod.Capabilities(), nil
}
