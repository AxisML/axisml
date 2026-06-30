// Package controllers holds the axisml-scheduler-controller reconcilers.
//
// ElasticQuotaController publishes ElasticQuota.status.used so external readers
// (the AxisML operators) can surface live quota usage. Usage is aggregated by
// the Pod label scheduling.axisml.io/quota — matching the ElasticScheduling
// plugin's label binding — rather than by namespace, which is how upstream
// scheduler-plugins' controller aggregates.
package controllers

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	schedclient "sigs.k8s.io/scheduler-plugins/pkg/generated/clientset/versioned"
	schedinformers "sigs.k8s.io/scheduler-plugins/pkg/generated/informers/externalversions"
	schedlisters "sigs.k8s.io/scheduler-plugins/pkg/generated/listers/scheduling/v1alpha1"

	"github.com/axisml/axisml/axisml-infra/axisml-scheduler/internal/quota"
)

// ElasticQuotaController reconciles ElasticQuota.status.used.
type ElasticQuotaController struct {
	schedClient schedclient.Interface
	podLister   corelisters.PodLister
	eqLister    schedlisters.ElasticQuotaLister
	queue       workqueue.TypedRateLimitingInterface[string]
	hasSynced   []cache.InformerSynced
}

// NewElasticQuotaController wires the informers and event handlers.
func NewElasticQuotaController(
	kubeClient kubernetes.Interface,
	schedClient schedclient.Interface,
	kubeFactory informers.SharedInformerFactory,
	schedFactory schedinformers.SharedInformerFactory,
) *ElasticQuotaController {
	podInformer := kubeFactory.Core().V1().Pods()
	eqInformer := schedFactory.Scheduling().V1alpha1().ElasticQuotas()

	c := &ElasticQuotaController{
		schedClient: schedClient,
		podLister:   podInformer.Lister(),
		eqLister:    eqInformer.Lister(),
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "elasticquota"},
		),
		hasSynced: []cache.InformerSynced{
			podInformer.Informer().HasSynced,
			eqInformer.Informer().HasSynced,
		},
	}

	_, _ = eqInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.enqueueEQ(obj) },
		UpdateFunc: func(_, obj interface{}) { c.enqueueEQ(obj) },
		DeleteFunc: func(obj interface{}) { c.enqueueEQ(obj) },
	})
	_, _ = podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { c.enqueuePod(obj) },
		UpdateFunc: func(_, obj interface{}) { c.enqueuePod(obj) },
		DeleteFunc: func(obj interface{}) { c.enqueuePod(obj) },
	})

	return c
}

func (c *ElasticQuotaController) enqueueEQ(obj interface{}) {
	// MetaNamespaceKeyFunc yields "namespace/name" — the same shape as quota.Key.
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.queue.Add(key)
}

func (c *ElasticQuotaController) enqueuePod(obj interface{}) {
	pod := podFrom(obj)
	if pod == nil {
		return
	}
	if name := pod.Labels[quota.LabelQuota]; name != "" {
		c.queue.Add(quota.Key(pod.Namespace, name))
	}
}

// Run starts the workers and blocks until ctx is cancelled.
func (c *ElasticQuotaController) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()
	logger := klog.FromContext(ctx)
	logger.Info("starting ElasticQuota controller")

	if !cache.WaitForCacheSync(ctx.Done(), c.hasSynced...) {
		return fmt.Errorf("failed to sync informer caches")
	}
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}
	<-ctx.Done()
	logger.Info("shutting down ElasticQuota controller")
	return nil
}

func (c *ElasticQuotaController) runWorker(ctx context.Context) {
	for c.processNext(ctx) {
	}
}

func (c *ElasticQuotaController) processNext(ctx context.Context) bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(key)

	if err := c.reconcile(ctx, key); err != nil {
		klog.FromContext(ctx).Error(err, "reconcile elasticquota", "key", key)
		c.queue.AddRateLimited(key)
		return true
	}
	c.queue.Forget(key)
	return true
}

func (c *ElasticQuotaController) reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil // malformed key; drop
	}

	eq, err := c.eqLister.ElasticQuotas(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Pre-filter by the quota label via the informer index instead of listing the
	// whole namespace and filtering in Go.
	pods, err := c.podLister.Pods(namespace).List(labels.SelectorFromSet(labels.Set{quota.LabelQuota: name}))
	if err != nil {
		return err
	}

	used := v1.ResourceList{}
	for _, pod := range pods {
		if pod.Spec.NodeName == "" || quota.Terminal(pod) {
			continue
		}
		for resName, q := range quota.PodRequest(pod) {
			cur := used[resName]
			cur.Add(q)
			used[resName] = cur
		}
	}
	// Normalize empty to nil so the no-op check matches a never-set status:
	// Semantic.DeepEqual treats a nil map and an empty map as unequal, which
	// would otherwise rewrite status every reconcile for empty quotas.
	if len(used) == 0 {
		used = nil
	}

	if apiequality.Semantic.DeepEqual(eq.Status.Used, used) {
		return nil
	}

	updated := eq.DeepCopy()
	updated.Status.Used = used
	if _, err := c.schedClient.SchedulingV1alpha1().ElasticQuotas(namespace).UpdateStatus(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func podFrom(obj interface{}) *v1.Pod {
	switch t := obj.(type) {
	case *v1.Pod:
		return t
	case cache.DeletedFinalStateUnknown:
		if pod, ok := t.Obj.(*v1.Pod); ok {
			return pod
		}
	}
	return nil
}
