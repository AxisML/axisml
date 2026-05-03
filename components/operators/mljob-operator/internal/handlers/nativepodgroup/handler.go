// Package nativepodgroup implements the (native, podgroup) handler from
// design §8.2: a single-role MLJob is rendered as one PodGroup CR
// (sigs.k8s.io/scheduler-plugins) plus N bare Pods, all bound to the
// PodGroup via the standard label and dispatched by koord-scheduler.
//
// The "all-or-nothing" gang scheduling semantics make this the right
// pick for synchronous distributed training where partial Pod readiness
// would deadlock NCCL/Gloo handshakes.
package nativepodgroup

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
	schedulingv1alpha1 "sigs.k8s.io/scheduler-plugins/apis/scheduling/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axishandler "axisml.io/operators/mljob/internal/handler"
)

const (
	BackendName   = "native"
	BackendEngine = "podgroup"

	roleName = "worker"
)

// Handler implements axishandler.Handler.
type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Key() axishandler.Key {
	return axishandler.Key{Backend: BackendName, Engine: BackendEngine}
}

func (h *Handler) WatchTargets() []client.Object {
	// Watch PodGroup AND bare Pods: PodGroup events tell us when gang
	// resolution flipped; Pod events propagate Running/Succeeded/Failed
	// transitions for MapStatus aggregation.
	return []client.Object{
		&schedulingv1alpha1.PodGroup{},
		&corev1.Pod{},
	}
}

// underlying is what we hand to MapStatus: the PodGroup plus the live
// Pods. nil-safety is at MapStatus.
type underlying struct {
	PodGroup *schedulingv1alpha1.PodGroup
	Pods     []corev1.Pod
	// DesiredReplicas surfaces the spec-side count even when no Pods
	// exist yet (e.g. gang waiting for resources).
	DesiredReplicas int32
}

// pgName mirrors the MLJob name; PodGroups live in the same namespace.
func pgName(mljobName string) string { return mljobName }

// podName is deterministic so reconcile is a true upsert, not a "list +
// recreate missing" loop. Indexed by 0..replicas-1.
func podName(mljobName string, idx int32) string {
	return mljobName + "-" + strconv.FormatInt(int64(idx), 10)
}
