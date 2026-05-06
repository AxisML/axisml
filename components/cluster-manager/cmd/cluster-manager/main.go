// Command axisml-cluster-manager is the admin-tier REST entry point
// for tenant + quota management. Stateless thin shell that translates
// REST calls into Tenant CR CRUD on the K8s API server.
package main

import (
	"flag"
	"os"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/axisml/axisml/components/cluster-manager/internal/app"
)

func main() {
	cfg := app.Config{}
	var denylistRaw string

	flag.StringVar(&cfg.APIBindAddress, "api-bind-address", ":8082", "REST API listen address.")
	flag.StringVar(&cfg.MetricsBindAddress, "metrics-bind-address", ":8080", "Prometheus metrics listen address.")
	flag.StringVar(&cfg.ProbesBindAddress, "probes-bind-address", ":8081", "Health probe listen address (/healthz, /readyz).")
	flag.StringVar(&denylistRaw, "namespace-denylist",
		"kube-system,kube-public,kube-node-lease,default,axisml-system,axisml-infra",
		"Comma-separated list of namespaces a Tenant CR may NOT target.")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))

	for _, n := range strings.Split(denylistRaw, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			cfg.NamespaceDenylist = append(cfg.NamespaceDenylist, n)
		}
	}

	ctx := ctrl.SetupSignalHandler()
	if err := app.Run(ctx, cfg); err != nil {
		ctrl.Log.Error(err, "cluster-manager exited with error")
		os.Exit(1)
	}
}
