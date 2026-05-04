package app

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/axisml/axisml/components/compute/internal/config"
	jobmod "github.com/axisml/axisml/components/compute/internal/job"
	quotamod "github.com/axisml/axisml/components/compute/internal/quota"
	poolmod "github.com/axisml/axisml/components/compute/internal/resourcepool"
	unitmod "github.com/axisml/axisml/components/compute/internal/resourceunit"
	"github.com/axisml/axisml/components/compute/internal/server"
	servicemod "github.com/axisml/axisml/components/compute/internal/service"
	tenantmod "github.com/axisml/axisml/components/compute/internal/tenant"
)

// BuildModules constructs the full domain wiring (HTTP routes + background
// runnables). Construction order matters because of cross-module deps.
// Exported so envtest harness can reuse the same wiring rather than
// duplicating it (and silently drifting).
func BuildModules(
	cfg config.Config,
	gormDB *gorm.DB,
	mgr manager.Manager,
	log logr.Logger,
) ([]server.Module, []manager.Runnable, error) {
	pools := poolmod.NewService(gormDB)
	units := unitmod.NewService(gormDB)
	quotas := quotamod.NewService(gormDB, pools)
	tenants := tenantmod.NewService(gormDB, quotas, pools)
	jobs := jobmod.NewService(gormDB, tenants, pools, units, quotas)
	services := servicemod.NewService(gormDB, tenants, pools, units, quotas)

	tenantMW := tenantmod.Middleware(tenants)

	tenantRecon := tenantmod.NewReconciler(gormDB, mgr.GetClient(), pools, log.WithName("tenant-reconciler"), cfg.ReconcileInterval)
	tenantRecon.SetQuotas(quotas)
	jobRecon := jobmod.NewReconciler(gormDB, mgr.GetClient(), tenants, log.WithName("job-reconciler"), cfg.ReconcileInterval)
	jobRecon.SetQuotas(quotas)
	serviceRecon := servicemod.NewReconciler(gormDB, mgr.GetClient(), tenants, log.WithName("service-reconciler"), cfg.ReconcileInterval)

	tenantInf := tenantmod.NewInformer(gormDB, mgr, quotas, log.WithName("tenant-informer"))
	jobInf := jobmod.NewInformer(gormDB, mgr, log.WithName("job-informer"))
	serviceInf := servicemod.NewInformer(gormDB, mgr, log.WithName("service-informer"))

	modules := []server.Module{
		tenantmod.NewHandler(tenants),
		poolmod.NewHandler(pools),
		unitmod.NewHandler(units, pools),
		quotamod.NewHandler(quotas, tenantMW),
		jobmod.NewHandler(jobs, tenantMW),
		servicemod.NewHandler(services, tenantMW),
	}
	runnables := []manager.Runnable{
		tenantRecon, jobRecon, serviceRecon,
		tenantInf, jobInf, serviceInf,
	}
	return modules, runnables, nil
}
