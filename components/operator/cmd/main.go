// Command axisml-operator runs the merged Kubernetes operator that
// reconciles Tenant, MLJob, and MLService CRs in a single manager.
// Each controller can be disabled independently via --enable-* flags.
package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	mljobdispatcher "github.com/axisml/axisml/components/operator/internal/mljob/dispatcher"
	"github.com/axisml/axisml/components/operator/internal/mljob/handlers/nativejob"
	"github.com/axisml/axisml/components/operator/internal/mljob/handlers/nativepodgroup"
	mlservicedispatcher "github.com/axisml/axisml/components/operator/internal/mlservice/dispatcher"
	mlservicehandler "github.com/axisml/axisml/components/operator/internal/mlservice/handler"
	"github.com/axisml/axisml/components/operator/internal/setup"
	tenantconfig "github.com/axisml/axisml/components/operator/internal/tenant/config"
	tenantcontroller "github.com/axisml/axisml/components/operator/internal/tenant/controller"
	tenantvalidate "github.com/axisml/axisml/components/operator/internal/tenant/validate"

	// Side-effect import: registers the (native, deployment) MLService handler.
	_ "github.com/axisml/axisml/components/operator/internal/mlservice/handler/nativedeployment"
)

const defaultLeaderElectionID = "axisml-operator.axisml.io"

var scheme = runtime.NewScheme()

func init() {
	setup.AddToScheme(scheme)
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		leaderElectionID     string
		enableTenant         bool
		enableMLJob          bool
		enableMLService      bool
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id",
		defaultLeaderElectionID,
		"Name of the lease used for leader election.")
	flag.BoolVar(&enableTenant, "enable-tenant", true, "Run the Tenant controller.")
	flag.BoolVar(&enableMLJob, "enable-mljob", true, "Run the MLJob controller.")
	flag.BoolVar(&enableMLService, "enable-mlservice", true, "Run the MLService controller.")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))
	setupLog := ctrl.Log.WithName("setup")

	// Tenant config is tenant-only: don't let RESYNC_PERIOD /
	// NAMESPACE_DENYLIST silently steer the manager when tenant is off.
	var tenantCfg tenantconfig.Config
	cacheOpts := cache.Options{}
	if enableTenant {
		tenantCfg = tenantconfig.Load()
		resync := tenantCfg.ResyncPeriod
		cacheOpts.SyncPeriod = &resync
		cacheOpts.ByObject = tenantcontroller.CacheByObject()
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		Cache:                  cacheOpts,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if enableTenant {
		if err := (&tenantcontroller.TenantReconciler{
			Client:    mgr.GetClient(),
			APIReader: mgr.GetAPIReader(),
			Scheme:    mgr.GetScheme(),
			ValidateOpts: tenantvalidate.Options{
				NamespaceDenylist: tenantCfg.NamespaceDenylist,
			},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up Tenant controller")
			os.Exit(1)
		}
	}

	if enableMLJob {
		registry := mljobdispatcher.NewRegistry()
		registry.Register(nativejob.New())
		registry.Register(nativepodgroup.New())
		if err := (&mljobdispatcher.MLJobReconciler{
			Client:   mgr.GetClient(),
			Registry: registry,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to set up MLJob controller")
			os.Exit(1)
		}
	}

	if enableMLService {
		handlersByKey, allHandlers, err := mlservicehandler.Build(mgr)
		if err != nil {
			setupLog.Error(err, "unable to build MLService handler registry")
			os.Exit(1)
		}
		for _, k := range mlservicehandler.Keys() {
			setupLog.Info("registered MLService handler", "key", k.String())
		}
		if err := mlservicedispatcher.NewReconciler(mgr, handlersByKey).SetupWithManager(mgr, allHandlers); err != nil {
			setupLog.Error(err, "unable to set up MLService controller")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to register healthz")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to register readyz")
		os.Exit(1)
	}

	setupLog.Info("starting axisml-operator",
		"tenant", enableTenant,
		"mljob", enableMLJob,
		"mlservice", enableMLService,
	)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager terminated")
		os.Exit(1)
	}
}
