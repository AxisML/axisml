package k8sclient

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/axisml/axisml/components/compute-service/internal/config"

	resourcepoolv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// Scheme returns a runtime.Scheme pre-loaded with all CRDs that compute
// needs to interact with.
func Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mlrunv1alpha1.AddToScheme(scheme))
	utilruntime.Must(mlservicev1alpha1.AddToScheme(scheme))
	utilruntime.Must(mltrafficpolicyv1alpha1.AddToScheme(scheme))
	// compute resolves ResourcePool/ResourceUnit (owned by cluster-manager) when
	// materializing Job/Service workloads, so its type must be in the scheme.
	utilruntime.Must(resourcepoolv1alpha1.AddToScheme(scheme))
	return scheme
}

// NewManager builds a controller-runtime Manager configured for compute.
// The Manager owns the shared cache, leader election lease, and probe HTTP
// endpoints that the API binary exposes alongside its own gin server.
func NewManager(cfg config.Config) (manager.Manager, error) {
	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("kube rest config: %w", err)
	}
	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme:                     Scheme(),
		Metrics:                    metricsserver.Options{BindAddress: cfg.MetricsBindAddress},
		HealthProbeBindAddress:     cfg.ProbesBindAddress,
		LeaderElection:             cfg.LeaderElect,
		LeaderElectionID:           cfg.LeaderElectionID,
		LeaderElectionNamespace:    cfg.LeaderResourceNS,
		LeaderElectionResourceLock: cfg.LeaderResourceLock,
		LeaseDuration:              durationPtr(cfg.LeaderLeaseDuration),
		RenewDeadline:              durationPtr(cfg.LeaderRenewDeadline),
		RetryPeriod:                durationPtr(cfg.LeaderRetryPeriod),
	})
	if err != nil {
		return nil, fmt.Errorf("manager: %w", err)
	}
	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return nil, err
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		return nil, err
	}
	return mgr, nil
}

func durationPtr(d time.Duration) *time.Duration {
	if d == 0 {
		return nil
	}
	return &d
}
