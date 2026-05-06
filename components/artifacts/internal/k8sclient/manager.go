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

	"github.com/axisml/axisml/components/artifacts/internal/config"
)

// Scheme returns a runtime.Scheme. Artifacts watches no CRDs in MVP, so
// only the core client-go scheme is registered (needed for the leader
// election Lease in `coordination.k8s.io`).
func Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	return scheme
}

// NewManager builds a controller-runtime Manager configured for artifacts.
// The Manager owns the leader election lease, probe endpoints, and the
// /metrics server. The HTTP server and GC worker are added to the Manager
// as Runnables.
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
