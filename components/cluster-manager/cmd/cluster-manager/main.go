// Command axisml-cluster-manager is the admin-tier REST entry point for
// ResourcePool management. Stateless thin shell that translates REST calls
// into K8s API CRUD on the ResourcePool CRD (with embedded spec.units[]).
package main

import (
	"flag"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/axisml/axisml/components/cluster-manager/internal/app"
)

func main() {
	cfg := app.Config{}

	flag.StringVar(&cfg.APIBindAddress, "api-bind-address", ":8080", "REST API listen address.")
	flag.StringVar(&cfg.MetricsBindAddress, "metrics-bind-address", ":8081", "Prometheus metrics listen address.")
	flag.StringVar(&cfg.ProbesBindAddress, "probes-bind-address", ":8082", "Health probe listen address (/healthz, /readyz).")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	ctx := ctrl.SetupSignalHandler()
	if err := app.Run(ctx, cfg); err != nil {
		ctrl.Log.Error(err, "cluster-manager exited with error")
		os.Exit(1)
	}
}
