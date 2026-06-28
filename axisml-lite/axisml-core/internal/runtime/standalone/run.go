package standalone

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
)

// renderRunPlans renders a (native, job) MLRun into one ContainerPlan per role
// replica. Unsupported backends and fields surface as CapabilityError.
func (r *Runtime) renderRunPlans(run *mlrunv1alpha1.MLRun) ([]ContainerPlan, error) {
	if run.Spec.Backend.Name != "native" || run.Spec.Backend.Engine != "job" {
		return nil, capabilityError("MLRun backend %s/%s is unsupported in Lite (only native/job)",
			run.Spec.Backend.Name, run.Spec.Backend.Engine)
	}
	ns, name := run.Namespace, run.Name
	var plans []ContainerPlan
	for _, role := range run.Spec.Roles {
		if role.Replicas <= 0 {
			continue
		}
		tmpl := role.Template
		if len(tmpl.EnvFrom) > 0 {
			return nil, capabilityError("MLRun role %q envFrom is unsupported in Lite", role.Name)
		}
		env, err := envToStrings(tmpl.Env)
		if err != nil {
			return nil, err
		}
		for i := 0; i < int(role.Replicas); i++ {
			labels := r.baseLabels(KindRun, ns, name)
			labels[LabelRole] = role.Name
			labels[LabelReplicaIndex] = formatLabelInt(i)
			p := ContainerPlan{
				Name:          r.containerName(KindRun, ns, name, role.Name, i),
				Image:         tmpl.Image,
				Command:       tmpl.Command,
				Args:          tmpl.Args,
				Env:           env,
				WorkingDir:    tmpl.WorkingDir,
				Labels:        labels,
				Resources:     resourcePlan(tmpl.Resources),
				RestartPolicy: string(container.RestartPolicyDisabled),
			}
			p.Labels[LabelSpecHash] = specHash(planIdentity(&p))
			plans = append(plans, p)
		}
	}
	if len(plans) == 0 {
		return nil, capabilityError("MLRun %s/%s has no enabled role replicas", ns, name)
	}
	return plans, nil
}

// planIdentity is the change-detection projection of a plan (everything but the
// stable name/labels), so an unchanged spec does not rebuild the container.
func planIdentity(p *ContainerPlan) any {
	return struct {
		Image         string
		Command       []string
		Args          []string
		Env           []string
		WorkingDir    string
		Ports         []PortPlan
		Mounts        []MountPlan
		Resources     ResourcePlan
		RestartPolicy string
	}{p.Image, p.Command, p.Args, p.Env, p.WorkingDir, p.Ports, p.Mounts, p.Resources, p.RestartPolicy}
}

// ApplyMLRun renders and idempotently converges the Run's containers. A Run's
// spec is immutable, so a re-apply with matching spec hashes is a no-op. On a
// mid-loop create failure the already-created containers are left running and
// the error is returned; the reconcile loop re-applies and, being idempotent
// (unchanged containers skip by spec hash, missing ones are created), converges
// the rest. Tearing the created containers back down would only force the next
// apply to rebuild them from scratch.
func (r *Runtime) ApplyMLRun(ctx context.Context, desired *mlrunv1alpha1.MLRun) error {
	ns, name := desired.Namespace, desired.Name
	plans, err := r.renderRunPlans(desired)
	if err != nil {
		return err
	}
	existing, err := r.listContainers(ctx, KindRun, ns, name)
	if err != nil {
		return err
	}
	byName := indexByName(existing)

	var created []string
	for i := range plans {
		p := &plans[i]
		if cur, ok := byName[p.Name]; ok {
			if cur.Labels[LabelSpecHash] == p.Labels[LabelSpecHash] {
				continue // unchanged
			}
			if err := r.removeContainer(ctx, cur.ID); err != nil {
				return err
			}
		}
		// Always attempt a pull; on failure fall back to a locally-present image.
		if err := r.pullImage(ctx, p.Image); err != nil {
			r.log.Info("image pull failed, trying local", "image", p.Image, "err", err.Error())
		}
		id, err := r.createAndStart(ctx, p)
		if err != nil {
			r.events.record(KindRun, ns, name, "", "ApplyFailed", err.Error())
			return err
		}
		created = append(created, id)
	}
	if len(created) > 0 {
		r.events.record(KindRun, ns, name, "", "Created", "run containers created")
	}
	return nil
}

// ObserveMLRun aggregates the Run's container states into an MLRunStatus. A Run
// with no containers surfaces as NotFound.
func (r *Runtime) ObserveMLRun(ctx context.Context, key types.NamespacedName) (mlrunv1alpha1.MLRunStatus, error) {
	conts, err := r.listContainers(ctx, KindRun, key.Namespace, key.Name)
	if err != nil {
		return mlrunv1alpha1.MLRunStatus{}, err
	}
	if len(conts) == 0 {
		return mlrunv1alpha1.MLRunStatus{}, notFound("mlruns", key)
	}
	states, err := r.inspectAll(ctx, conts)
	if err != nil {
		return mlrunv1alpha1.MLRunStatus{}, err
	}

	var running, succeeded, failed, pending int
	var startedAt, finishedAt *time.Time
	var failMsg string
	for i := range states {
		s := &states[i]
		if startedAt == nil && !s.startedAt.IsZero() {
			t := s.startedAt
			startedAt = &t
		}
		switch {
		case s.status == "running":
			running++
		case s.status == "exited" && s.exitCode == 0:
			succeeded++
			if !s.finishedAt.IsZero() {
				t := s.finishedAt
				finishedAt = &t
			}
		case s.status == "exited":
			failed++
			if s.oom {
				failMsg = "container OOMKilled"
			} else {
				failMsg = "container exited with non-zero status"
			}
			if !s.finishedAt.IsZero() {
				t := s.finishedAt
				finishedAt = &t
			}
		default:
			pending++
		}
	}

	status := mlrunv1alpha1.MLRunStatus{}
	if startedAt != nil {
		status.StartedAt = &metav1.Time{Time: *startedAt}
	}

	switch {
	case failed > 0:
		status.Phase = mlrunv1alpha1.PhaseFailed
		status.Message = failMsg
		status.FinishedAt = metaTimePtr(finishedAt)
	case succeeded == len(states):
		status.Phase = mlrunv1alpha1.PhaseSucceeded
		status.FinishedAt = metaTimePtr(finishedAt)
	case running > 0:
		status.Phase = mlrunv1alpha1.PhaseRunning
	default:
		status.Phase = mlrunv1alpha1.PhasePending
	}

	// A cancel stops the containers; surface the agreed Suspended condition so
	// the Compute mapping converges Canceling → Cancelled regardless of the
	// observed exit code (design §6.6).
	if r.isCancelled(key.Namespace, key.Name) {
		status.Conditions = append(status.Conditions, metav1.Condition{
			Type:               mlrunv1alpha1.ConditionSuspended,
			Status:             metav1.ConditionTrue,
			Reason:             mlrunv1alpha1.ReasonCancelRequested,
			Message:            "run cancelled",
			LastTransitionTime: metav1.NewTime(nowMetaTime()),
		})
	}
	return status, nil
}

// CancelMLRun stops the Run's containers and records the cancel so Observe can
// report Suspended=CancelRequested. Idempotent.
func (r *Runtime) CancelMLRun(ctx context.Context, key types.NamespacedName) error {
	r.markCancelled(key.Namespace, key.Name)
	conts, err := r.listContainers(ctx, KindRun, key.Namespace, key.Name)
	if err != nil {
		return err
	}
	for _, c := range conts {
		if err := r.stopContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	r.events.record(KindRun, key.Namespace, key.Name, "", "Cancelled", "run cancel requested")
	return nil
}

// DeleteMLRun removes the Run's containers. Idempotent.
func (r *Runtime) DeleteMLRun(ctx context.Context, key types.NamespacedName) error {
	conts, err := r.listContainers(ctx, KindRun, key.Namespace, key.Name)
	if err != nil {
		return err
	}
	for _, c := range conts {
		if err := r.removeContainer(ctx, c.ID); err != nil {
			return err
		}
	}
	r.clearCancelled(key.Namespace, key.Name)
	if len(conts) > 0 {
		r.events.record(KindRun, key.Namespace, key.Name, "", "Deleted", "run containers removed")
	}
	return nil
}

// ListMLRunInstances projects the Run's containers as Pods.
func (r *Runtime) ListMLRunInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error) {
	conts, err := r.listContainers(ctx, KindRun, key.Namespace, key.Name)
	if err != nil {
		return nil, err
	}
	return r.projectPods(ctx, key.Namespace, conts)
}

// GetMLRunInstanceLogs streams an instance's logs after verifying ownership.
func (r *Runtime) GetMLRunInstanceLogs(ctx context.Context, key types.NamespacedName, instance string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	if err := r.verifyInstance(ctx, KindRun, key, instance); err != nil {
		return nil, err
	}
	return r.streamLogs(ctx, instance, opts)
}

// GetMLRunInstanceEvents returns runtime events for an instance after verifying
// ownership.
func (r *Runtime) GetMLRunInstanceEvents(ctx context.Context, key types.NamespacedName, instance string) (*eventsv1.EventList, error) {
	if err := r.verifyInstance(ctx, KindRun, key, instance); err != nil {
		return nil, err
	}
	return r.events.list(KindRun, key.Namespace, key.Name, instance), nil
}

// GetMLRunEvents returns resource-level runtime events for the Run.
func (r *Runtime) GetMLRunEvents(_ context.Context, key types.NamespacedName) (*eventsv1.EventList, error) {
	return r.events.list(KindRun, key.Namespace, key.Name, ""), nil
}

func metaTimePtr(t *time.Time) *metav1.Time {
	if t == nil {
		return nil
	}
	return &metav1.Time{Time: *t}
}
