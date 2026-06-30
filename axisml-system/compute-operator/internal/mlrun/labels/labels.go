// Package labels re-exports the label and annotation keys defined on
// the public api/v1alpha1 surface under the names used by the mlrun
// operator's internal packages. The public package is the canonical
// source; this shim keeps the internal call sites compact.
package labels

import axisv1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlrun/v1alpha1"

const (
	RunIDLabel          = axisv1alpha1.LabelRunID
	QuotaLabel          = axisv1alpha1.LabelQuota
	RoleLabel           = axisv1alpha1.LabelRole
	SchedulerQuotaLabel = axisv1alpha1.LabelSchedulerQuota

	AppliedSpecAnnotation = axisv1alpha1.AnnotationAppliedSpec

	SchedulerName = axisv1alpha1.SchedulerName
)
