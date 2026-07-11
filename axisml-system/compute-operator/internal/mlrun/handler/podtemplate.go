package handler

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	axislabels "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/labels"
)

// EnsureRequiredCRLabels validates that the MLRun CR carries the labels
// Compute is contracted to set (design §3 / §6 Pod injection contract).
// Pod-side enforcement happens in InjectAxisMLLabels; this guards the
// inputs so a contract violation surfaces at Validate time, not after
// a Pod escapes with missing labels.
func EnsureRequiredCRLabels(mlJob *axisv1alpha1.MLRun) field.ErrorList {
	var errs field.ErrorList
	required := []string{axislabels.RunIDLabel, axislabels.QuotaLabel}
	if mlJob.Labels == nil {
		errs = append(errs, field.Required(field.NewPath("metadata", "labels"), fmt.Sprintf("missing required labels %v (set by Compute)", required)))
		return errs
	}
	for _, key := range required {
		if mlJob.Labels[key] == "" {
			errs = append(errs, field.Required(field.NewPath("metadata", "labels").Key(key), "Compute must set this label before creating the CR"))
		}
	}
	return errs
}

// InjectAxisMLLabels mutates the supplied PodTemplateSpec so that every
// Pod the handler renders carries the five mandatory labels and uses
// axisml-scheduler. Returns an error if the MLRun CR is missing inputs
// (this should have been caught earlier by EnsureRequiredCRLabels).
//
// extraLabels lets the handler add backend-specific labels (e.g. the
// scheduler-plugins PodGroup membership label) without re-implementing
// the merge.
func InjectAxisMLLabels(tmpl *corev1.PodTemplateSpec, mlJob *axisv1alpha1.MLRun, role axisv1alpha1.RoleSpec, extraLabels map[string]string) error {
	if mlJob.Labels[axislabels.RunIDLabel] == "" {
		return fmt.Errorf("MLRun %s/%s is missing required label %s", mlJob.Namespace, mlJob.Name, axislabels.RunIDLabel)
	}
	if mlJob.Labels[axislabels.QuotaLabel] == "" {
		return fmt.Errorf("MLRun %s/%s is missing required label %s", mlJob.Namespace, mlJob.Name, axislabels.QuotaLabel)
	}
	if mlJob.Spec.Scheduling.Quota == "" {
		return fmt.Errorf("MLRun %s/%s spec.scheduling.quota is empty", mlJob.Namespace, mlJob.Name)
	}

	if tmpl.Labels == nil {
		tmpl.Labels = map[string]string{}
	}
	tmpl.Labels[axislabels.RunIDLabel] = mlJob.Labels[axislabels.RunIDLabel]
	tmpl.Labels[axislabels.QuotaLabel] = mlJob.Labels[axislabels.QuotaLabel]
	tmpl.Labels[axislabels.RoleLabel] = role.Name
	tmpl.Labels[axislabels.SchedulerQuotaLabel] = mlJob.Spec.Scheduling.Quota
	for k, v := range extraLabels {
		tmpl.Labels[k] = v
	}

	tmpl.Spec.SchedulerName = axislabels.SchedulerName
	tmpl.Spec.PriorityClassName = mlJob.Spec.Scheduling.PriorityClass
	if len(mlJob.Spec.Scheduling.NodeSelector) > 0 {
		if tmpl.Spec.NodeSelector == nil {
			tmpl.Spec.NodeSelector = map[string]string{}
		}
		for k, v := range mlJob.Spec.Scheduling.NodeSelector {
			tmpl.Spec.NodeSelector[k] = v
		}
	}
	if len(mlJob.Spec.Scheduling.Tolerations) > 0 {
		tmpl.Spec.Tolerations = append(tmpl.Spec.Tolerations, mlJob.Spec.Scheduling.Tolerations...)
	}
	return nil
}

// BuildContainer renders the role's PodTemplateSubset into a corev1
// Container. Container name is the role name so logs/exec are
// addressable by `kubectl exec -c <role>`.
func BuildContainer(role axisv1alpha1.RoleSpec) corev1.Container {
	return corev1.Container{
		Name:            role.Name,
		Image:           role.Template.Image,
		ImagePullPolicy: role.Template.ImagePullPolicy,
		Command:         role.Template.Command,
		Args:            role.Template.Args,
		Env:             role.Template.Env,
		EnvFrom:         role.Template.EnvFrom,
		WorkingDir:      role.Template.WorkingDir,
		Resources:       role.Template.Resources,
		VolumeMounts:    role.Template.VolumeMounts,
	}
}

// OwnerRef returns a controller=true ownerReference pointing to the
// MLRun, so derived resources are GC'd cascadingly.
func OwnerRef(mlJob *axisv1alpha1.MLRun) metav1.OwnerReference {
	t := true
	return metav1.OwnerReference{
		APIVersion:         axisv1alpha1.GroupVersion.String(),
		Kind:               "MLRun",
		Name:               mlJob.Name,
		UID:                mlJob.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}
