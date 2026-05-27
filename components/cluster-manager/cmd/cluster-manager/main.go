// Command axisml-cluster-manager is the admin-tier REST entry point for
// ResourcePool management. Stateless thin shell that translates REST calls
// into K8s API CRUD on the ResourcePool CRD (with embedded spec.units[]).
//
// Subcommands:
//   - serve     (default) — run the REST server.
//   - bootstrap          — idempotently create the default ResourcePool
//     (parity with the axisml-system Helm chart's
//     post-install hook for clusters that install
//     cluster-manager out-of-band).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	"github.com/axisml/axisml/components/cluster-manager/internal/app"
	"github.com/axisml/axisml/components/cluster-manager/internal/k8sclient"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "bootstrap" {
		runBootstrap()
		return
	}
	runServe()
}

func runServe() {
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

// runBootstrap creates the cluster-wide `default` ResourcePool if absent.
// Idempotent (AlreadyExists is treated as success). Equivalent to the
// post-install Helm hook in axisml-system, exposed as a subcommand so
// air-gapped / non-Helm installs can still seed the pool.
func runBootstrap() {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	poolName := fs.String("pool", "default", "Name of the seed ResourcePool.")
	smallCPU := fs.String("small-cpu", "1", "cpu-small unit CPU request/limit.")
	smallMem := fs.String("small-memory", "2Gi", "cpu-small unit memory request/limit.")
	mediumCPU := fs.String("medium-cpu", "4", "cpu-medium unit CPU request/limit.")
	mediumMem := fs.String("medium-memory", "8Gi", "cpu-medium unit memory request/limit.")
	_ = fs.Parse(os.Args[2:])

	c, err := k8sclient.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build k8s client: %v\n", err)
		os.Exit(1)
	}

	pool := &axismlv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{
			Name: *poolName,
			Annotations: map[string]string{
				"axisml.io/description": "Default cluster-wide pool seeded by cluster-manager bootstrap",
			},
		},
		Spec: axismlv1alpha1.ResourcePoolSpec{
			Units: []axismlv1alpha1.ResourceUnit{
				unit("cpu-small", *smallCPU, *smallMem),
				unit("cpu-medium", *mediumCPU, *mediumMem),
			},
		},
	}

	ctx := context.Background()
	existing := &axismlv1alpha1.ResourcePool{}
	switch err := c.Get(ctx, types.NamespacedName{Name: *poolName}, existing); {
	case err == nil:
		fmt.Fprintf(os.Stderr, "ResourcePool %q already exists; nothing to do\n", *poolName)
		return
	case !apierrors.IsNotFound(err):
		fmt.Fprintf(os.Stderr, "lookup ResourcePool %q: %v\n", *poolName, err)
		os.Exit(1)
	}
	if err := c.Create(ctx, pool); err != nil && !apierrors.IsAlreadyExists(err) {
		fmt.Fprintf(os.Stderr, "create ResourcePool %q: %v\n", *poolName, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "ResourcePool %q created\n", *poolName)
	_ = client.IgnoreNotFound
}

func unit(name, cpu, mem string) axismlv1alpha1.ResourceUnit {
	rl := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(cpu),
		corev1.ResourceMemory: resource.MustParse(mem),
	}
	return axismlv1alpha1.ResourceUnit{
		Name:     name,
		Requests: rl,
		Limits:   rl,
	}
}
