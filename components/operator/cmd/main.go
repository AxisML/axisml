// Command axisml-operator runs the merged Kubernetes operator that
// reconciles Tenant, MLJob, and MLService CRs in a single manager.
// Each controller can be disabled independently via --enable-* flags.
package main

import (
	"flag"
	"os"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
	mljobdispatcher "github.com/axisml/axisml/components/operator/internal/mljob/dispatcher"
	"github.com/axisml/axisml/components/operator/internal/mljob/handlers/nativejob"
	"github.com/axisml/axisml/components/operator/internal/mljob/handlers/nativepodgroup"
	mlservicedispatcher "github.com/axisml/axisml/components/operator/internal/mlservice/dispatcher"
	mlservicehandler "github.com/axisml/axisml/components/operator/internal/mlservice/handler"
	tenantconfig "github.com/axisml/axisml/components/operator/internal/tenant/config"
	tenantcontroller "github.com/axisml/axisml/components/operator/internal/tenant/controller"
	tenantvalidate "github.com/axisml/axisml/components/operator/internal/tenant/validate"

	// Side-effect import: registers the (native, deployment) MLService handler.
	_ "github.com/axisml/axisml/components/operator/internal/mlservice/handler/nativedeployment"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	utilruntime.Must(schedulingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(gwapiv1.Install(scheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))
	utilruntime.Must(mljobv1alpha1.AddToScheme(scheme))
	utilruntime.Must(mlservicev1alpha1.AddToScheme(scheme))

	mlservicehandler.RegisterStubs()
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
		enableNativeJob      bool
		enableNativePodGroup bool
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080",
		"The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081",
		"The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id",
		"axisml-operator.axisml.io",
		"Name of the lease used for leader election.")
	flag.BoolVar(&enableTenant, "enable-tenant", true, "Run the Tenant controller.")
	flag.BoolVar(&enableMLJob, "enable-mljob", true, "Run the MLJob controller.")
	flag.BoolVar(&enableMLService, "enable-mlservice", true, "Run the MLService controller.")
	flag.BoolVar(&enableNativeJob, "enable-native-job", true,
		"MLJob: register the (native, job) handler.")
	flag.BoolVar(&enableNativePodGroup, "enable-native-podgroup", true,
		"MLJob: register the (native, podgroup) handler.")

	zapOpts := zap.Options{Development: false}
	zapOpts.BindFlags(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))
	setupLog := ctrl.Log.WithName("setup")

	tenantCfg := tenantconfig.Load()
	resync := tenantCfg.ResyncPeriod

	cacheOpts := cache.Options{SyncPeriod: &resync}
	if enableTenant {
		// Restrict the cache for types Tenant manages to objects carrying our
		// managed-by label. Without this, the default cache would set up a
		// cluster-wide informer for every Secret/ConfigMap, pulling them all
		// into memory. Source Secret/ConfigMap reads bypass the cache via the
		// APIReader (operator design §5: "operator 不为源资源建立 watch").
		// Per-type filter only — do NOT promote to a global default selector,
		// or MLJob/MLService informers (Job, Deployment, PodGroup, HTTPRoute)
		// would also be filtered.
		managedByOnly := labels.SelectorFromSet(labels.Set{
			tenantv1alpha1.LabelManagedBy: tenantv1alpha1.ManagedByValue,
		})
		cacheOpts.ByObject = map[client.Object]cache.ByObject{
			&corev1.Secret{}:                   {Label: managedByOnly},
			&corev1.ConfigMap{}:                {Label: managedByOnly},
			&corev1.ServiceAccount{}:           {Label: managedByOnly},
			&rbacv1.Role{}:                     {Label: managedByOnly},
			&rbacv1.RoleBinding{}:              {Label: managedByOnly},
			&schedulingv1alpha1.ElasticQuota{}: {Label: managedByOnly},
		}
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
		if enableNativeJob {
			registry.Register(nativejob.New())
		}
		if enableNativePodGroup {
			registry.Register(nativepodgroup.New())
		}
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
		"nativeJob", enableNativeJob,
		"nativePodGroup", enableNativePodGroup,
	)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager terminated")
		os.Exit(1)
	}
}
