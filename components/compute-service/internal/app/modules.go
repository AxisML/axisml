package app

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	jobmod "github.com/axisml/axisml/components/compute-service/internal/job"
	"github.com/axisml/axisml/components/compute-service/internal/poolcache"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/service"
	tenantmod "github.com/axisml/axisml/components/compute-service/internal/tenant"
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
	services := servicemod.NewService(gormDB, pools)

	jobRecon := jobmod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("job-reconciler"), cfg.ReconcileInterval)
	serviceRecon := servicemod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("service-reconciler"), cfg.ReconcileInterval)

	jobInf := jobmod.NewInformer(gormDB, mgr, log.WithName("job-informer"))
	serviceInf := servicemod.NewInformer(gormDB, mgr, log.WithName("service-informer"))

	modules := []server.Module{
		tenantmod.NewHandler(tenants),
		jobmod.NewHandler(jobs),
		servicemod.NewHandler(services),
	}
	runnables := []manager.Runnable{
		jobRecon, serviceRecon,
		jobInf, serviceInf,
	}
	return modules, runnables, nil
}
