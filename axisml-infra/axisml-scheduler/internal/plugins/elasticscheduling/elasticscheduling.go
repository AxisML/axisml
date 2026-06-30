// Package elasticscheduling implements the AxisML ElasticScheduling scheduler
// plugin: a label-bound ElasticQuota enforcer.
//
// It is the AxisML counterpart of upstream scheduler-plugins CapacityScheduling,
// differing in one deliberate way: a Pod is associated to an ElasticQuota by the
// label scheduling.axisml.io/quota (so one namespace can hold many quotas),
// instead of by namespace. PreFilter rejects a Pod whose admission would push
// its quota's usage past spec.max; Reserve/Unreserve keep in-flight accounting
// consistent across concurrent scheduling cycles.
//
// It fails closed: a Pod that selects this scheduler but lacks the quota label,
// or whose ElasticQuota does not (yet) exist, is left Unschedulable rather than
// admitted unaccounted — the scheduler itself upholds the system-level "no
// quota-bypass" invariant instead of trusting callers to always stamp the label.
//
// Min-based borrowing reclaim via preemption is not yet implemented (see the
// design doc §4.3 / §10); spec.max enforcement is the load-bearing
// "超 max 拒绝调度" invariant and is implemented here. The companion controller
// publishes ElasticQuota.status.used for external read-back; this plugin computes
// usage independently for scheduling decisions.
package elasticscheduling

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/scheduler-plugins/apis/scheduling"
	"sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	schedutil "sigs.k8s.io/scheduler-plugins/pkg/util"

	"github.com/axisml/axisml/axisml-infra/axisml-scheduler/internal/quota"
)

// Name is the plugin name used in Registry and KubeSchedulerConfiguration.
const Name = "ElasticScheduling"

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

// ElasticScheduling enforces per-ElasticQuota spec.max, bound by Pod label.
type ElasticScheduling struct {
	logger    klog.Logger
	client    client.Client
	podLister corelisters.PodLister
	usage     *quota.Cache
}

var (
	_ framework.PreFilterPlugin   = &ElasticScheduling{}
	_ framework.ReservePlugin     = &ElasticScheduling{}
	_ framework.EnqueueExtensions = &ElasticScheduling{}
)

// Name returns the plugin name.
func (e *ElasticScheduling) Name() string { return Name }

// New constructs the plugin: a cached ElasticQuota reader plus a Pod informer
// that maintains label-keyed usage accounting.
func New(ctx context.Context, _ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)

	cl, ccache, err := schedutil.NewClientWithCachedReader(ctx, handle.KubeConfig(), scheme)
	if err != nil {
		return nil, err
	}

	e := &ElasticScheduling{
		logger:    logger,
		client:    cl,
		podLister: handle.SharedInformerFactory().Core().V1().Pods().Lister(),
		usage:     quota.New(),
	}

	// Warm the ElasticQuota cache so PreFilter Get calls are served locally.
	if _, err := ccache.GetInformer(ctx, &v1alpha1.ElasticQuota{}); err != nil {
		return nil, err
	}

	podInformer := handle.SharedInformerFactory().Core().V1().Pods().Informer()
	if _, err := podInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			switch t := obj.(type) {
			case *v1.Pod:
				return managedPod(t)
			case cache.DeletedFinalStateUnknown:
				if pod, ok := t.Obj.(*v1.Pod); ok {
					return managedPod(pod)
				}
				return false
			default:
				return false
			}
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    e.onPodAdd,
			UpdateFunc: e.onPodUpdate,
			DeleteFunc: e.onPodDelete,
		},
	}); err != nil {
		return nil, err
	}

	logger.Info("ElasticScheduling started")
	return e, nil
}

// PreFilter rejects the Pod if (quota usage + pod request) would exceed spec.max
// for any constrained resource. It fails closed: a Pod that reaches this
// scheduler without the quota label, or whose ElasticQuota does not yet exist,
// is left Unschedulable rather than admitted unaccounted — that upholds the
// system "no quota-bypass" invariant at the scheduler. Such Pods become
// schedulable once the label is set or the ElasticQuota is created (both are
// registered as requeue events in EventsToRegister).
func (e *ElasticScheduling) PreFilter(ctx context.Context, _ fwk.CycleState, pod *v1.Pod, _ []fwk.NodeInfo) (*framework.PreFilterResult, *fwk.Status) {
	eqName := pod.Labels[quota.LabelQuota]
	if eqName == "" {
		return nil, fwk.NewStatus(fwk.Unschedulable,
			fmt.Sprintf("missing required quota label %q; no quota-bypass scheduling path is allowed", quota.LabelQuota))
	}

	eq := &v1alpha1.ElasticQuota{}
	if err := e.client.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: eqName}, eq); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fwk.NewStatus(fwk.Unschedulable,
				fmt.Sprintf("ElasticQuota %s/%s not found; waiting for it before admitting the pod", pod.Namespace, eqName))
		}
		return nil, fwk.AsStatus(fmt.Errorf("get elasticquota %s/%s: %w", pod.Namespace, eqName, err))
	}
	if len(eq.Spec.Max) == 0 {
		return nil, fwk.NewStatus(fwk.Success)
	}

	req := quota.PodRequest(pod)
	// Used() already returns a private copy, so the per-resource value can be
	// mutated in place without a second DeepCopy.
	used := e.usage.Used(quota.Key(pod.Namespace, eqName))
	for name, max := range eq.Spec.Max {
		total := used[name]
		if r, ok := req[name]; ok {
			total.Add(r)
		}
		if total.Cmp(max) > 0 {
			return nil, fwk.NewStatus(fwk.Unschedulable,
				fmt.Sprintf("ElasticQuota %s/%s exceeded for %s: requested-with-usage %s > max %s",
					pod.Namespace, eqName, name, total.String(), max.String()))
		}
	}
	return nil, fwk.NewStatus(fwk.Success)
}

// PreFilterExtensions returns nil: the plugin does not adjust state when other
// pods are added to or removed from a node during scheduling.
func (e *ElasticScheduling) PreFilterExtensions() framework.PreFilterExtensions { return nil }

// EventsToRegister wakes Pods this plugin left Unschedulable: a freed Pod (more
// quota headroom) or any ElasticQuota change (a newly-created quota, or a raised
// max). Without these hints such Pods would only retry on the periodic backoff
// flush.
func (e *ElasticScheduling) EventsToRegister(_ context.Context) ([]fwk.ClusterEventWithHint, error) {
	eqGVK := fmt.Sprintf("elasticquotas.v1alpha1.%v", scheduling.GroupName)
	return []fwk.ClusterEventWithHint{
		{Event: fwk.ClusterEvent{Resource: fwk.Pod, ActionType: fwk.Delete}},
		{Event: fwk.ClusterEvent{Resource: fwk.EventResource(eqGVK), ActionType: fwk.All}},
	}, nil
}

// Reserve accounts the assumed pod against its quota before the Pod informer
// observes the binding, so concurrent cycles in the same quota see it.
func (e *ElasticScheduling) Reserve(_ context.Context, _ fwk.CycleState, pod *v1.Pod, _ string) *fwk.Status {
	if eqName := pod.Labels[quota.LabelQuota]; eqName != "" {
		e.usage.AddPod(quota.Key(pod.Namespace, eqName), string(pod.UID), quota.PodRequest(pod))
	}
	return nil
}

// Unreserve rolls back the Reserve accounting when binding fails.
func (e *ElasticScheduling) Unreserve(_ context.Context, _ fwk.CycleState, pod *v1.Pod, _ string) {
	if eqName := pod.Labels[quota.LabelQuota]; eqName != "" {
		e.usage.RemovePod(quota.Key(pod.Namespace, eqName), string(pod.UID))
	}
}

func (e *ElasticScheduling) onPodAdd(obj interface{}) {
	if pod, ok := obj.(*v1.Pod); ok {
		e.usage.AddPod(quota.Key(pod.Namespace, pod.Labels[quota.LabelQuota]), string(pod.UID), quota.PodRequest(pod))
	}
}

func (e *ElasticScheduling) onPodUpdate(oldObj, newObj interface{}) {
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		return
	}
	newKey := quota.Key(newPod.Namespace, newPod.Labels[quota.LabelQuota])
	// If the pod's quota binding changed, drop the stale accounting under the old
	// key — otherwise the request leaks against the previous quota forever.
	if oldPod, ok := oldObj.(*v1.Pod); ok {
		if oldKey := quota.Key(oldPod.Namespace, oldPod.Labels[quota.LabelQuota]); oldKey != newKey {
			e.usage.RemovePod(oldKey, string(oldPod.UID))
		}
	}
	if quota.Terminal(newPod) {
		e.usage.RemovePod(newKey, string(newPod.UID))
		return
	}
	e.usage.AddPod(newKey, string(newPod.UID), quota.PodRequest(newPod))
}

func (e *ElasticScheduling) onPodDelete(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
			pod, ok = tombstone.Obj.(*v1.Pod)
			if !ok {
				return
			}
		} else {
			return
		}
	}
	e.usage.RemovePod(quota.Key(pod.Namespace, pod.Labels[quota.LabelQuota]), string(pod.UID))
}

// managedPod selects assigned, non-terminal Pods carrying the quota label — the
// ones that actually consume quota. Excluding terminal pods keeps informer
// resync (which replays finished pods via AddFunc) from inflating usage, and
// makes a running→terminal transition surface as a synthetic delete (the
// filtering handler calls OnDelete when the object stops matching), which clears
// the pod's accounting.
func managedPod(pod *v1.Pod) bool {
	return pod.Spec.NodeName != "" && pod.Labels[quota.LabelQuota] != "" && !quota.Terminal(pod)
}
