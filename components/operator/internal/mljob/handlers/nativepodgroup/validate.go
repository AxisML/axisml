package nativepodgroup

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	axisv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
	axishandler "github.com/axisml/axisml/components/operator/internal/mljob/handler"
)

func (h *Handler) Validate(mlJob *axisv1alpha1.MLJob) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, axishandler.EnsureRequiredCRLabels(mlJob)...)

	rolesPath := field.NewPath("spec", "roles")
	if len(mlJob.Spec.Roles) != 1 {
		errs = append(errs, field.Invalid(rolesPath, len(mlJob.Spec.Roles), "(native, podgroup) requires exactly one role"))
		return errs
	}
	role := mlJob.Spec.Roles[0]
	if role.Name != roleName {
		errs = append(errs, field.Invalid(rolesPath.Index(0).Child("name"), role.Name,
			fmt.Sprintf("(native, podgroup) requires role name %q", roleName)))
	}
	if role.Replicas < 1 {
		errs = append(errs, field.Invalid(rolesPath.Index(0).Child("replicas"), role.Replicas,
			"must be >= 1 (gang scheduling needs at least one Pod)"))
	}
	if role.Template.Image == "" {
		errs = append(errs, field.Required(rolesPath.Index(0).Child("template", "image"), "image is required"))
	}
	switch role.RestartPolicy {
	case "", corev1.RestartPolicyOnFailure, corev1.RestartPolicyNever:
	default:
		errs = append(errs, field.NotSupported(
			rolesPath.Index(0).Child("restartPolicy"),
			role.RestartPolicy,
			[]string{string(corev1.RestartPolicyOnFailure), string(corev1.RestartPolicyNever)},
		))
	}
	if mlJob.Spec.Scheduling.Quota == "" {
		errs = append(errs, field.Required(field.NewPath("spec", "scheduling", "quota"), "quota is required"))
	}
	return errs
}
