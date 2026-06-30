// Command axisml-scheduler-controller maintains ElasticQuota.status.used so the
// AxisML operators can read live per-quota usage. It aggregates by the Pod label
// scheduling.axisml.io/quota, matching the scheduler's ElasticScheduling plugin.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"

	schedclient "sigs.k8s.io/scheduler-plugins/pkg/generated/clientset/versioned"
	schedinformers "sigs.k8s.io/scheduler-plugins/pkg/generated/informers/externalversions"

	"github.com/axisml/axisml/axisml-infra/axisml-scheduler/internal/controllers"
)

func main() {
	klog.InitFlags(nil)
	var (
		kubeconfig  = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required out-of-cluster.")
		resync      = flag.Duration("resync-period", 30*time.Second, "Informer resync period.")
		workers     = flag.Int("workers", 2, "Number of reconcile workers.")
		leaderElect = flag.Bool("leader-elect", true, "Enable leader election so only one replica writes ElasticQuota status.")
		leaseName   = flag.String("leader-election-id", "axisml-scheduler-controller", "Lease name for leader election.")
		leaseNS     = flag.String("leader-election-namespace", envOr("POD_NAMESPACE", "axisml-infra"), "Namespace holding the leader-election Lease.")
		identity    = flag.String("identity", envOr("POD_NAME", hostname()), "Unique identity for leader election (defaults to pod name / hostname).")
	)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	logger := klog.FromContext(ctx)

	cfg, err := loadConfig(*kubeconfig)
	if err != nil {
		logger.Error(err, "load kube config")
		os.Exit(1)
	}

	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	schedClient := schedclient.NewForConfigOrDie(cfg)

	run := func(ctx context.Context) {
		kubeFactory := informers.NewSharedInformerFactory(kubeClient, *resync)
		schedFactory := schedinformers.NewSharedInformerFactory(schedClient, *resync)
		ctrl := controllers.NewElasticQuotaController(kubeClient, schedClient, kubeFactory, schedFactory)
		kubeFactory.Start(ctx.Done())
		schedFactory.Start(ctx.Done())
		if err := ctrl.Run(ctx, *workers); err != nil {
			logger.Error(err, "controller exited with error")
		}
	}

	if !*leaderElect {
		run(ctx)
		return
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: *leaseName, Namespace: *leaseNS},
		Client:     kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: *identity},
	}
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,
			OnStoppedLeading: func() { logger.Info("lost leadership; stopping") },
		},
	})
}

func loadConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "axisml-scheduler-controller"
	}
	return h
}
