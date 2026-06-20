package app

import (
	"github.com/go-logr/logr"
	"gorm.io/gorm"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/axisml/axisml/components/compute-service/internal/config"
	"github.com/axisml/axisml/components/compute-service/internal/kubeproxy"
	jobmod "github.com/axisml/axisml/components/compute-service/internal/mlrun"
	servicemod "github.com/axisml/axisml/components/compute-service/internal/mlservice"
	"github.com/axisml/axisml/components/compute-service/internal/poolcache"
	"github.com/axisml/axisml/components/compute-service/internal/server"
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

	jobs := jobmod.NewService(gormDB, pools)
	serviceRepo := servicemod.NewRepository(gormDB)
	trafficPolicyRepo := trafficpolicymod.NewRepository(gormDB)
	services := servicemod.NewMLService(gormDB, pools, mgr.GetClient(), trafficPolicyRepo)
	trafficPolicies := trafficpolicymod.NewService(gormDB, serviceRepo, trafficPolicyRepo)

	jobRecon := jobmod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("mlrun-reconciler"), cfg.ReconcileInterval)
	serviceRecon := servicemod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("mlservice-reconciler"), cfg.ReconcileInterval)
	trafficRecon := trafficpolicymod.NewReconciler(gormDB, mgr.GetClient(), log.WithName("traffic-policy-reconciler"), cfg.ReconcileInterval)

	jobInf := jobmod.NewInformer(gormDB, mgr, log.WithName("mlrun-informer"))
	serviceInf := servicemod.NewInformer(gormDB, mgr, log.WithName("mlservice-informer"))
	trafficInf := trafficpolicymod.NewInformer(gormDB, mgr, log.WithName("traffic-policy-informer"))

	kube, err := kubeproxy.New(mgr.GetConfig(), mgr.GetClient())
	if err != nil {
		return nil, nil, err
	}

	modules := []server.Module{
		jobmod.NewHandler(jobs, kube),
		servicemod.NewHandler(services, kube),
		trafficpolicymod.NewHandler(trafficPolicies),
	}
	runnables := []manager.Runnable{
		jobRecon, serviceRecon, trafficRecon,
		jobInf, serviceInf, trafficInf,
	}
	return modules, runnables, nil
}
