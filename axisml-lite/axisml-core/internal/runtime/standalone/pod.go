package standalone

import (
	"context"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// projectPods projects managed containers as corev1.Pods so the Compute layer
// can serve the existing Pod list / logs / events API against Docker
// containers (design §2, §6.1). The instance (Pod) name is the stable
// container name.
func (r *Runtime) projectPods(ctx context.Context, namespace string, conts []container.Summary) (*corev1.PodList, error) {
	states, err := r.inspectAll(ctx, conts)
	if err != nil {
		return nil, err
	}
	list := &corev1.PodList{}
	for i := range states {
		s := &states[i]
		list.Items = append(list.Items, r.projectPod(namespace, s))
	}
	return list, nil
}

func (r *Runtime) projectPod(namespace string, s *instState) corev1.Pod {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: namespace,
			Labels: map[string]string{
				LabelRole:         s.role,
				LabelReplicaIndex: s.replica,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: roleOr(s.role), Image: s.image}},
		},
	}
	phase, cs := containerStatus(s)
	pod.Status.Phase = phase
	if !s.startedAt.IsZero() {
		t := metav1.NewTime(s.startedAt)
		pod.Status.StartTime = &t
	}
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{cs}
	if s.ready() {
		pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	}
	return pod
}

func containerStatus(s *instState) (corev1.PodPhase, corev1.ContainerStatus) {
	cs := corev1.ContainerStatus{Name: roleOr(s.role), Image: s.image, Ready: s.ready()}
	switch {
	case s.status == "running":
		cs.State.Running = &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(s.startedAt)}
		if s.ready() {
			return corev1.PodRunning, cs
		}
		return corev1.PodRunning, cs
	case s.status == "exited" && s.exitCode == 0:
		cs.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: 0, Reason: "Completed", FinishedAt: metav1.NewTime(s.finishedAt)}
		return corev1.PodSucceeded, cs
	case s.status == "exited":
		reason := "Error"
		if s.oom {
			reason = "OOMKilled"
		}
		cs.State.Terminated = &corev1.ContainerStateTerminated{ExitCode: int32(s.exitCode), Reason: reason, FinishedAt: metav1.NewTime(s.finishedAt)}
		return corev1.PodFailed, cs
	default:
		cs.State.Waiting = &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}
		return corev1.PodPending, cs
	}
}

func roleOr(role string) string {
	if role == "" {
		return "worker"
	}
	return role
}

// verifyInstance confirms the named instance belongs to the addressed workload,
// returning a NotFound for an unknown instance and ErrInstanceNotOwned for a
// managed container that belongs to a different workload (design contract).
func (r *Runtime) verifyInstance(ctx context.Context, kind string, key types.NamespacedName, instance string) error {
	conts, err := r.listContainers(ctx, kind, key.Namespace, key.Name)
	if err != nil {
		return err
	}
	for _, c := range conts {
		if summaryName(c) == instance {
			return nil
		}
	}
	if _, err := r.cli.ContainerInspect(ctx, instance); err != nil {
		if cerrdefs.IsNotFound(err) {
			return notFound("pods", types.NamespacedName{Namespace: key.Namespace, Name: instance})
		}
		return err
	}
	return extensions.ErrInstanceNotOwned
}
