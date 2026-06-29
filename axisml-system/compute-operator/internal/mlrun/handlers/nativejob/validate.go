package nativejob

import (
	"bytes"
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"
	axishandler "github.com/axisml/axisml/axisml-system/compute-operator/internal/mlrun/handler"
)

func (h *Handler) Validate(mlJob *axisv1alpha1.MLRun) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, axishandler.EnsureRequiredCRLabels(mlJob)...)

	rolesPath := field.NewPath("spec", "roles")
	if len(mlJob.Spec.Roles) != 1 {
		errs = append(errs, field.Invalid(rolesPath, len(mlJob.Spec.Roles), "(native, job) requires exactly one role"))
		return errs
	}
	role := mlJob.Spec.Roles[0]
	if role.Name != roleName {
		errs = append(errs, field.Invalid(rolesPath.Index(0).Child("name"), role.Name,
			fmt.Sprintf("(native, job) requires role name %q", roleName)))
	}
	if role.Replicas < 1 {
		errs = append(errs, field.Invalid(rolesPath.Index(0).Child("replicas"), role.Replicas, "must be >= 1"))
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

	if mlJob.Spec.Backend.Config != nil && len(mlJob.Spec.Backend.Config.Raw) > 0 {
		var cfg backendConfig
		dec := json.NewDecoder(bytes.NewReader(mlJob.Spec.Backend.Config.Raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cfg); err != nil {
			errs = append(errs, field.Invalid(field.NewPath("spec", "backend", "config"), string(mlJob.Spec.Backend.Config.Raw),
				fmt.Sprintf("invalid backend.config: %v", err)))
		} else if cfg.CompletionMode != "" && cfg.CompletionMode != batchv1.NonIndexedCompletion && cfg.CompletionMode != batchv1.IndexedCompletion {
			errs = append(errs, field.NotSupported(
				field.NewPath("spec", "backend", "config", "completionMode"),
				string(cfg.CompletionMode),
				[]string{string(batchv1.NonIndexedCompletion), string(batchv1.IndexedCompletion)},
			))
		}
	}
	return errs
}
