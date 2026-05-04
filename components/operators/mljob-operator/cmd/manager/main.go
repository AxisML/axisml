// Command manager runs the mljob-operator: it watches MLJob CRs and
// dispatches to backend-specific handlers. Handlers are registered
// explicitly here (not via package init) so build/runtime can choose to
// disable a backend without removing its source.
package main

import (
	"flag"
	"os"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	axisv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/mljob-operator/internal/dispatcher"
	"github.com/axisml/axisml/components/operators/mljob-operator/internal/handlers/nativejob"
	"github.com/axisml/axisml/components/operators/mljob-operator/internal/handlers/nativepodgroup"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(axisv1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		enableNativeJob      bool
		enableNativePodGroup bool
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.BoolVar(&enableNativeJob, "enable-native-job", true, "Register the (native, job) handler.")
	flag.BoolVar(&enableNativePodGroup, "enable-native-podgroup", true, "Register the (native, podgroup) handler.")

	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "mljob-operator.axisml.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	registry := dispatcher.NewRegistry()
	if enableNativeJob {
		registry.Register(nativejob.New())
	}
	if enableNativePodGroup {
		registry.Register(nativepodgroup.New())
	}

	r := &dispatcher.MLJobReconciler{
		Client:   mgr.GetClient(),
		Registry: registry,
	}
	if err := r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up MLJob controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to add healthz check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to add readyz check")
		os.Exit(1)
	}

	setupLog.Info("starting mljob-operator",
		"nativeJob", enableNativeJob,
		"nativePodGroup", enableNativePodGroup)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager terminated")
		os.Exit(1)
	}
}
