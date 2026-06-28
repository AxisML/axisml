// Command axisml-tenant-operator reconciles Tenant CRs into Namespaces,
// Koordinator ElasticQuotas, and per-tenant Secret/ConfigMap/SA/RBAC
// resources. Single reconciler, no dispatcher.
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

	tenantconfig "github.com/axisml/axisml/components/tenant-operator/internal/config"
	tenantcontroller "github.com/axisml/axisml/components/tenant-operator/internal/controller"
	"github.com/axisml/axisml/components/tenant-operator/internal/setup"
	tenantvalidate "github.com/axisml/axisml/components/tenant-operator/internal/validate"
)

const defaultLeaderElectionID = "axisml-tenant-operator.axisml.io"

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
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8081",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8082",
		"The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id",
		defaultLeaderElectionID,
		"Name of the lease used for leader election.")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))
	setupLog := ctrl.Log.WithName("setup")

	cfg := tenantconfig.Load()
	resync := cfg.ResyncPeriod

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		Cache: cache.Options{
			SyncPeriod: &resync,
			ByObject:   tenantcontroller.CacheByObject(),
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&tenantcontroller.TenantReconciler{
		Client:    mgr.GetClient(),
		APIReader: mgr.GetAPIReader(),
		Scheme:    mgr.GetScheme(),
		ValidateOpts: tenantvalidate.Options{
			NamespaceDenylist: cfg.NamespaceDenylist,
		},
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up Tenant controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to register healthz")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to register readyz")
		os.Exit(1)
	}

	setupLog.Info("starting axisml-tenant-operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager terminated")
		os.Exit(1)
	}
}
