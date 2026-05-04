package nativepodgroup

import (
	"context"
	"fmt"
	"time"

	schedulingv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
	axishandler "github.com/axisml/axisml/components/operators/mljob-operator/internal/handler"
	axislabels "github.com/axisml/axisml/components/operators/mljob-operator/internal/labels"
)

func (h *Handler) Reconcile(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob) (any, axishandler.ReconcileResult, error) {
	role := mlJob.Spec.Roles[0]
	pgKey := types.NamespacedName{Namespace: mlJob.Namespace, Name: pgName(mlJob.Name)}

	// Suspend path FIRST: design §8.2 mandates ordered shutdown — patch
	// minMember=0 BEFORE deleting Pods, otherwise koord-scheduler's gang
	// plugin will reschedule the deleted Pods immediately.
	if mlJob.Spec.RunPolicy.Suspend {
		return h.reconcileSuspend(ctx, c, mlJob, pgKey)
	}

	// Normal path: ensure PodGroup exists with the right minMember.
	pg, err := h.ensurePodGroup(ctx, c, mlJob, pgKey, role.Replicas)
	if err != nil {
		return nil, axishandler.ReconcileResult{}, err
	}

	// List existing Pods first; create only the missing replicas.
	pods, err := h.listPods(ctx, c, mlJob)
	if err != nil {
		return nil, axishandler.ReconcileResult{}, err
	}
	if err := h.ensurePods(ctx, c, mlJob, role, pods); err != nil {
		return nil, axishandler.ReconcileResult{}, err
	}

	// TTL fallback: bare Pods have no native TTL; do it ourselves once
	// every Pod is in a terminal phase. We deliberately keep the local
	// `pods` snapshot intact even after deletion so the same reconcile
	// can let MapStatus record the terminal phase before dispatcher
	// short-circuits on subsequent watch events. Post-terminal sweeps
	// (TTL > 0 deadlines that haven't expired yet) re-enter via Sweep().
	res := axishandler.ReconcileResult{}
	if mlJob.Spec.RunPolicy.TTLSecondsAfterFinished != nil {
		requeueAfter, _, err := h.ttlSweep(ctx, c, mlJob, pods)
		if err != nil {
			return nil, axishandler.ReconcileResult{}, err
		}
		if requeueAfter > 0 {
			res.RequeueAfterSeconds = requeueAfter
		}
	}

	return underlying{PodGroup: pg, Pods: pods, DesiredReplicas: role.Replicas}, res, nil
}

// Sweep runs post-terminal TTL GC. Dispatcher invokes this after the
// MLJob enters a terminal phase, replacing the recreate-prone
// Reconcile path. Bare Pods have no native TTL controller, so we own
// the deadline.
func (h *Handler) Sweep(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob) (int32, error) {
	if mlJob.Spec.RunPolicy.TTLSecondsAfterFinished == nil {
		return 0, nil
	}
	pods, err := h.listPods(ctx, c, mlJob)
	if err != nil {
		return 0, err
	}
	if len(pods) == 0 {
		return 0, nil
	}
	requeueAfter, _, err := h.ttlSweep(ctx, c, mlJob, pods)
	return requeueAfter, err
}

func (h *Handler) reconcileSuspend(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob, pgKey types.NamespacedName) (any, axishandler.ReconcileResult, error) {
	// Step 1: patch PodGroup minMember=0 (if PodGroup exists).
	var pg schedulingv1alpha1.PodGroup
	getErr := c.Get(ctx, pgKey, &pg)
	pgFound := getErr == nil
	switch {
	case apierrors.IsNotFound(getErr):
		// Nothing was created yet. Treat as cancel-before-create — also
		// covered by the suspend-completed signal so dispatcher writes
		// the cancel condition immediately.
	case getErr != nil:
		return nil, axishandler.ReconcileResult{}, getErr
	default:
		if pg.Spec.MinMember != 0 {
			patch := client.MergeFrom(pg.DeepCopy())
			pg.Spec.MinMember = 0
			if err := c.Patch(ctx, &pg, patch); err != nil {
				return nil, axishandler.ReconcileResult{}, fmt.Errorf("patch PodGroup minMember=0: %w", err)
			}
		}
	}

	// Step 2: delete all Pods belonging to this MLJob.
	pods, err := h.listPods(ctx, c, mlJob)
	if err != nil {
		return nil, axishandler.ReconcileResult{}, err
	}
	for i := range pods {
		if pods[i].DeletionTimestamp != nil {
			continue
		}
		if err := c.Delete(ctx, &pods[i]); err != nil && !apierrors.IsNotFound(err) {
			return nil, axishandler.ReconcileResult{}, fmt.Errorf("delete Pod %s/%s: %w", pods[i].Namespace, pods[i].Name, err)
		}
	}

	u := underlying{}
	if pgFound {
		u.PodGroup = &pg
	}
	return u, axishandler.ReconcileResult{
		SuspendCompleted: true,
		SuspendReason:    axisv1alpha1.ReasonCancelRequested,
	}, nil
}

func (h *Handler) ensurePodGroup(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob, pgKey types.NamespacedName, minMember int32) (*schedulingv1alpha1.PodGroup, error) {
	want := &schedulingv1alpha1.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pgKey.Name,
			Namespace: pgKey.Namespace,
			Labels: map[string]string{
				axislabels.JobIDLabel: mlJob.Labels[axislabels.JobIDLabel],
				axislabels.QuotaLabel: mlJob.Labels[axislabels.QuotaLabel],
			},
			OwnerReferences: []metav1.OwnerReference{axishandler.OwnerRef(mlJob)},
		},
		Spec: schedulingv1alpha1.PodGroupSpec{
			MinMember: minMember,
		},
	}
	var existing schedulingv1alpha1.PodGroup
	getErr := c.Get(ctx, pgKey, &existing)
	switch {
	case apierrors.IsNotFound(getErr):
		if err := c.Create(ctx, want); err != nil {
			return nil, fmt.Errorf("create PodGroup: %w", err)
		}
		return want, nil
	case getErr != nil:
		return nil, getErr
	}
	// Reconcile both minMember and the identity labels we own. Labels can
	// drift via manual edits or external controllers; without this, ops
	// queries like `kubectl get podgroups -l axisml.io/job-id=<id>` can
	// silently miss the PodGroup.
	needsPatch := existing.Spec.MinMember != minMember
	for k, v := range want.Labels {
		if existing.Labels[k] != v {
			needsPatch = true
			break
		}
	}
	if needsPatch {
		patch := client.MergeFrom(existing.DeepCopy())
		existing.Spec.MinMember = minMember
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		for k, v := range want.Labels {
			existing.Labels[k] = v
		}
		if err := c.Patch(ctx, &existing, patch); err != nil {
			return nil, fmt.Errorf("patch PodGroup: %w", err)
		}
	}
	return &existing, nil
}

// ensurePods creates only the Pods (by deterministic index name) that
// are missing from the supplied snapshot. The snapshot is the listPods
// result the caller already paid for, so we avoid the per-Pod Get round
// trip that would otherwise dominate API load on wide gangs.
func (h *Handler) ensurePods(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob, role axisv1alpha1.RoleSpec, existing []corev1.Pod) error {
	have := make(map[string]struct{}, len(existing))
	for i := range existing {
		have[existing[i].Name] = struct{}{}
	}
	for i := int32(0); i < role.Replicas; i++ {
		name := podName(mlJob.Name, i)
		if _, ok := have[name]; ok {
			continue
		}
		pod, err := h.buildPod(mlJob, role, i)
		if err != nil {
			return err
		}
		if err := c.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create Pod %s/%s: %w", pod.Namespace, name, err)
		}
	}
	return nil
}

func (h *Handler) buildPod(mlJob *axisv1alpha1.MLJob, role axisv1alpha1.RoleSpec, idx int32) (*corev1.Pod, error) {
	tmpl := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: corev1.PodSpec{
			RestartPolicy: defaultRestartPolicy(role.RestartPolicy),
			Containers:    []corev1.Container{axishandler.BuildContainer(role)},
		},
	}
	extra := map[string]string{
		axislabels.PodGroupLabel: pgName(mlJob.Name),
	}
	if err := axishandler.InjectAxisMLLabels(&tmpl, mlJob, role, extra); err != nil {
		return nil, err
	}
	if mlJob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
		tmpl.Spec.ActiveDeadlineSeconds = mlJob.Spec.RunPolicy.ActiveDeadlineSeconds
	}

	// The template was just rendered locally; sharing its label map with
	// the Pod's ObjectMeta is safe.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            podName(mlJob.Name, idx),
			Namespace:       mlJob.Namespace,
			Labels:          tmpl.Labels,
			OwnerReferences: []metav1.OwnerReference{axishandler.OwnerRef(mlJob)},
		},
		Spec: tmpl.Spec,
	}
	return pod, nil
}

func defaultRestartPolicy(p corev1.RestartPolicy) corev1.RestartPolicy {
	if p == "" {
		return corev1.RestartPolicyNever
	}
	return p
}

func (h *Handler) listPods(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob) ([]corev1.Pod, error) {
	var list corev1.PodList
	err := c.List(ctx, &list,
		client.InNamespace(mlJob.Namespace),
		client.MatchingLabels{axislabels.JobIDLabel: mlJob.Labels[axislabels.JobIDLabel]},
	)
	if err != nil {
		return nil, fmt.Errorf("list Pods: %w", err)
	}
	return list.Items, nil
}

// ttlSweep returns (requeueAfterSeconds, gced, err). When all Pods are
// in a terminal phase and TTL has expired, every Pod is deleted and
// gced=true. When TTL has not yet expired, requeueAfterSeconds is the
// time until the deadline.
func (h *Handler) ttlSweep(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob, pods []corev1.Pod) (int32, bool, error) {
	if len(pods) == 0 {
		return 0, false, nil
	}
	var latest *time.Time
	for i := range pods {
		switch pods[i].Status.Phase {
		case corev1.PodSucceeded, corev1.PodFailed:
		default:
			return 0, false, nil
		}
		fin := lastTerminationTime(&pods[i])
		if fin == nil {
			return 0, false, nil
		}
		if latest == nil || fin.After(*latest) {
			latest = fin
		}
	}
	deadline := latest.Add(time.Duration(*mlJob.Spec.RunPolicy.TTLSecondsAfterFinished) * time.Second)
	if !time.Now().After(deadline) {
		remaining := int32(time.Until(deadline).Seconds())
		if remaining < 1 {
			remaining = 1
		}
		return remaining, false, nil
	}
	for i := range pods {
		if err := c.Delete(ctx, &pods[i]); err != nil && !apierrors.IsNotFound(err) {
			return 0, false, fmt.Errorf("ttl GC delete Pod %s: %w", pods[i].Name, err)
		}
	}
	return 0, true, nil
}
