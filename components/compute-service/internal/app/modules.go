package app

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	jobmod "github.com/axisml/axisml/components/compute-service/internal/job"
	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	"github.com/axisml/axisml/components/compute-service/internal/poolcache"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/service"
	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
	trafficpolicymod "github.com/axisml/axisml/components/compute-service/internal/trafficpolicy"
)

// BuildModules constructs the full domain wiring (HTTP routes + background
// runnables). Jobs and services partition on bare namespace strings.
// ResourcePool is fed from the K8s Informer cache (controller-runtime
// manager client), not PG.
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	mgr manager.Manager,
	log logr.Logger,
) ([]server.Module, []manager.Runnable, error) {
	pools := poolcache.New(mgr.GetClient())

	tenants := tenantmod.NewService(gormDB)
	jobs := jobmod.NewService(gormDB, pools)
	services := servicemod.NewService(gormDB, pools, mgr.GetClient())
	trafficPolicies := trafficpolicymod.NewService(gormDB, servicemod.NewRepository(gormDB))

	tenantRecon := tenantmod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("tenant-reconciler"), cfg.ReconcileInterval)
	jobRecon := jobmod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("job-reconciler"), cfg.ReconcileInterval)
	serviceRecon := servicemod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("service-reconciler"), cfg.ReconcileInterval)
	trafficRecon := trafficpolicymod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("traffic-policy-reconciler"), cfg.ReconcileInterval)

	tenantInf := tenantmod.NewInformer(gormDB, mgr, log.WithName("tenant-informer"))
	jobInf := jobmod.NewInformer(gormDB, mgr, log.WithName("job-informer"))
	serviceInf := servicemod.NewInformer(gormDB, mgr, log.WithName("service-informer"))
	trafficInf := trafficpolicymod.NewInformer(gormDB, mgr, log.WithName("traffic-policy-informer"))

	kube, err := kubeproxy.New(mgr.GetConfig(), mgr.GetClient())
	if err != nil {
		return nil, nil, err
	}

	modules := []server.Module{
		tenantmod.NewHandler(tenants),
		jobmod.NewHandler(jobs, kube),
		servicemod.NewHandler(services, kube),
		trafficpolicymod.NewHandler(trafficPolicies),
	}
	runnables := []manager.Runnable{
		tenantRecon, jobRecon, serviceRecon, trafficRecon,
		tenantInf, jobInf, serviceInf, trafficInf,
	}
	return modules, runnables, nil
}
