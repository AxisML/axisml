package nativejob

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisv1alpha1 "axisml.io/operators/mljob/api/v1alpha1"
	axishandler "axisml.io/operators/mljob/internal/handler"
)

func (h *Handler) Reconcile(ctx context.Context, c client.Client, mlJob *axisv1alpha1.MLJob) (any, axishandler.ReconcileResult, error) {
	key := types.NamespacedName{Namespace: mlJob.Namespace, Name: jobName(mlJob.Name)}

	var existing batchv1.Job
	getErr := c.Get(ctx, key, &existing)
	switch {
	case apierrors.IsNotFound(getErr):
		// Cancel-before-create: when a CR arrives with suspend=true and
		// no underlying Job yet, do not create one. Mark cancel complete.
		if mlJob.Spec.RunPolicy.Suspend {
			return nil, axishandler.ReconcileResult{
				SuspendCompleted: true,
				SuspendReason:    axisv1alpha1.ReasonCancelRequested,
			}, nil
		}
		want, err := h.buildJob(mlJob)
		if err != nil {
			return nil, axishandler.ReconcileResult{}, err
		}
		if err := c.Create(ctx, want); err != nil {
			return nil, axishandler.ReconcileResult{}, fmt.Errorf("create Job: %w", err)
		}
		return want, axishandler.ReconcileResult{}, nil
	case getErr != nil:
		return nil, axishandler.ReconcileResult{}, getErr
	}

	// Suspend path: Job exists and the user requested cancel. Patch the
	// native suspend field — Kubernetes evicts running Pods on its own.
	if mlJob.Spec.RunPolicy.Suspend && (existing.Spec.Suspend == nil || !*existing.Spec.Suspend) {
		patch := client.MergeFrom(existing.DeepCopy())
		t := true
		existing.Spec.Suspend = &t
		if err := c.Patch(ctx, &existing, patch); err != nil {
			return &existing, axishandler.ReconcileResult{}, fmt.Errorf("patch Job suspend: %w", err)
		}
		return &existing, axishandler.ReconcileResult{
			SuspendCompleted: true,
			SuspendReason:    axisv1alpha1.ReasonCancelRequested,
		}, nil
	}
	if mlJob.Spec.RunPolicy.Suspend && existing.Spec.Suspend != nil && *existing.Spec.Suspend {
		// Already suspended — keep declaring the cancel-complete signal so
		// dispatcher's status merge keeps the Suspended condition fresh.
		return &existing, axishandler.ReconcileResult{
			SuspendCompleted: true,
			SuspendReason:    axisv1alpha1.ReasonCancelRequested,
		}, nil
	}

	// Job exists and we're not in suspend territory: nothing to update;
	// design §3.3 makes most spec fields immutable. Return the live
	// object so MapStatus has fresh status.
	return &existing, axishandler.ReconcileResult{}, nil
}

func (h *Handler) buildJob(mlJob *axisv1alpha1.MLJob) (*batchv1.Job, error) {
	role := mlJob.Spec.Roles[0]
	cfg, err := decodeConfig(mlJob)
	if err != nil {
		return nil, err
	}

	completions := role.Replicas
	parallelism := role.Replicas

	tmpl := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: corev1.PodSpec{
			RestartPolicy: defaultRestartPolicy(role.RestartPolicy),
			Containers:    []corev1.Container{axishandler.BuildContainer(role)},
		},
	}
	if err := axishandler.InjectAxisMLLabels(&tmpl, mlJob, role, nil); err != nil {
		return nil, err
	}

	// Carry the Pod template's labels on the Job itself too, so
	// `kubectl get jobs -l axisml.io/job-id=<id>` works for ops. The
	// template was just rendered locally; sharing the map is safe.
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jobName(mlJob.Name),
			Namespace:       mlJob.Namespace,
			Labels:          tmpl.ObjectMeta.Labels,
			OwnerReferences: []metav1.OwnerReference{axishandler.OwnerRef(mlJob)},
		},
		Spec: batchv1.JobSpec{
			Parallelism: &parallelism,
			Completions: &completions,
			Template:    tmpl,
		},
	}

	if mlJob.Spec.RunPolicy.ActiveDeadlineSeconds != nil {
		job.Spec.ActiveDeadlineSeconds = mlJob.Spec.RunPolicy.ActiveDeadlineSeconds
	}
	if mlJob.Spec.RunPolicy.TTLSecondsAfterFinished != nil {
		job.Spec.TTLSecondsAfterFinished = mlJob.Spec.RunPolicy.TTLSecondsAfterFinished
	}
	if mlJob.Spec.RunPolicy.BackoffLimit != nil {
		job.Spec.BackoffLimit = mlJob.Spec.RunPolicy.BackoffLimit
	}
	if cfg != nil {
		if cfg.CompletionMode != "" {
			mode := cfg.CompletionMode
			job.Spec.CompletionMode = &mode
		}
		if cfg.PodFailurePolicy != nil {
			job.Spec.PodFailurePolicy = cfg.PodFailurePolicy
		}
	}

	if mlJob.Spec.RunPolicy.Suspend {
		t := true
		job.Spec.Suspend = &t
	}
	return job, nil
}

func decodeConfig(mlJob *axisv1alpha1.MLJob) (*backendConfig, error) {
	if mlJob.Spec.Backend.Config == nil || len(mlJob.Spec.Backend.Config.Raw) == 0 {
		return nil, nil
	}
	var cfg backendConfig
	dec := json.NewDecoder(bytes.NewReader(mlJob.Spec.Backend.Config.Raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode backend.config: %w", err)
	}
	return &cfg, nil
}

func defaultRestartPolicy(p corev1.RestartPolicy) corev1.RestartPolicy {
	if p == "" {
		return corev1.RestartPolicyOnFailure
	}
	return p
}
