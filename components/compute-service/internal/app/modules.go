package app

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	jobmod "github.com/axisml/axisml/components/compute-service/internal/job"
	poolmod "github.com/axisml/axisml/components/compute-service/internal/resourcepool"
	unitmod "github.com/axisml/axisml/components/compute-service/internal/resourceunit"
	"github.com/axisml/axisml/components/compute-service/internal/server"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/service"
)

// BuildModules constructs the full domain wiring (HTTP routes + background
// runnables). Jobs and services partition on bare namespace strings.
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	mgr manager.Manager,
	log logr.Logger,
) ([]server.Module, []manager.Runnable, error) {
	pools := poolmod.NewService(gormDB)
	units := unitmod.NewService(gormDB)
	jobs := jobmod.NewService(gormDB, pools, units)
	services := servicemod.NewService(gormDB, pools, units)

	jobRecon := jobmod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("job-reconciler"), cfg.ReconcileInterval)
	serviceRecon := servicemod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("service-reconciler"), cfg.ReconcileInterval)

	jobInf := jobmod.NewInformer(gormDB, mgr, log.WithName("job-informer"))
	serviceInf := servicemod.NewInformer(gormDB, mgr, log.WithName("service-informer"))

	modules := []server.Module{
		poolmod.NewHandler(pools),
		unitmod.NewHandler(units, pools),
		jobmod.NewHandler(jobs),
		servicemod.NewHandler(services),
	}
	runnables := []manager.Runnable{
		jobRecon, serviceRecon,
		jobInf, serviceInf,
	}
	return modules, runnables, nil
}
