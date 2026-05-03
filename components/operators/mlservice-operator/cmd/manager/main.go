// Command manager is the mlservice-operator entry point. It boots a
// controller-runtime Manager, registers every Handler from the registry, and
// runs the dispatcher Reconciler until SIGTERM.
package main

import (
	"flag"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/mlservice-operator/internal/dispatcher"
	"github.com/axisml/axisml/components/operators/mlservice-operator/internal/handler"

	// Side-effect import: registers the (native, deployment) handler.
	_ "github.com/axisml/axisml/components/operators/mlservice-operator/internal/handler/nativedeployment"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(gwapiv1.Install(scheme))
	utilruntime.Must(axisml.AddToScheme(scheme))

	// Register the deferred (backend, engine) tuples that point at §11.
	handler.RegisterStubs()
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		leaderElectionID     string
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id",
		"mlservice-operator.axisml.io",
		"Name of the resource that leader election will use.")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))
	setupLog := ctrl.Log.WithName("setup")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	handlersByKey, allHandlers, err := handler.Build(mgr)
	if err != nil {
		setupLog.Error(err, "unable to build handler registry")
		os.Exit(1)
	}
	for _, k := range handler.Keys() {
		setupLog.Info("registered handler", "key", k.String())
	}

	r := dispatcher.NewReconciler(mgr, handlersByKey)
	if err := r.SetupWithManager(mgr, allHandlers); err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info(fmt.Sprintf("starting manager (handlers=%d)", len(allHandlers)))
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
