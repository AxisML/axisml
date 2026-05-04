package main

import (
	"flag"
	"fmt"
	"os"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	axisml "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/tenant-operator/internal/config"
	"github.com/axisml/axisml/components/operators/tenant-operator/internal/controller"
	"github.com/axisml/axisml/components/operators/tenant-operator/internal/validate"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(schedv1alpha1.AddToScheme(scheme))
	utilruntime.Must(axisml.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var leaderElect bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "address for the metrics endpoint")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "address for /healthz and /readyz")
	flag.BoolVar(&leaderElect, "leader-elect", false,
		"enable leader election (run a single active replica)")
	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	cfg := config.Load()

	resync := cfg.ResyncPeriod
	// Restrict the cache for types we manage to objects carrying our
	// managed-by label. Without this, the default cache would set up a
	// cluster-wide informer for every Secret/ConfigMap (driven by both Owns()
	// in SetupWithManager and per-tenant Get calls), pulling every Secret in
	// the cluster into memory. Source Secret/ConfigMap reads bypass the cache
	// via the APIReader (design §5: "operator 不为源资源建立 watch").
	managedByOnly := labels.SelectorFromSet(labels.Set{
		axisml.LabelManagedBy: axisml.ManagedByValue,
	})
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         leaderElect,
		LeaderElectionID:       "tenant-operator.axisml.io",
		Cache: cache.Options{
			SyncPeriod: &resync,
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Secret{}:              {Label: managedByOnly},
				&corev1.ConfigMap{}:           {Label: managedByOnly},
				&corev1.ServiceAccount{}:      {Label: managedByOnly},
				&rbacv1.Role{}:                {Label: managedByOnly},
				&rbacv1.RoleBinding{}:         {Label: managedByOnly},
				&schedv1alpha1.ElasticQuota{}: {Label: managedByOnly},
			},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start manager: %v\n", err)
		os.Exit(1)
	}

	if err := (&controller.TenantReconciler{
		Client:    mgr.GetClient(),
		APIReader: mgr.GetAPIReader(),
		Scheme:    mgr.GetScheme(),
		ValidateOpts: validate.Options{
			NamespaceDenylist: cfg.NamespaceDenylist,
		},
	}).SetupWithManager(mgr); err != nil {
		fmt.Fprintf(os.Stderr, "unable to set up tenant controller: %v\n", err)
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		fmt.Fprintf(os.Stderr, "unable to register healthz: %v\n", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		fmt.Fprintf(os.Stderr, "unable to register readyz: %v\n", err)
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Fprintf(os.Stderr, "manager exited with error: %v\n", err)
		os.Exit(1)
	}
}
