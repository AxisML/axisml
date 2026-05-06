// Package labels re-exports the label and annotation keys defined on
// the public api/v1alpha1 surface under the names used by the mljob
// operator's internal packages. The public package is the canonical
// source; this shim keeps the internal call sites compact.
package labels

import axisv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"

const (
	JobIDLabel      = axisv1alpha1.LabelJobID
	QuotaLabel      = axisv1alpha1.LabelQuota
	RoleLabel       = axisv1alpha1.LabelRole
	KoordQuotaLabel = axisv1alpha1.LabelKoordQuotaName
	PodGroupLabel   = axisv1alpha1.LabelPodGroup

	AppliedSpecAnnotation = axisv1alpha1.AnnotationAppliedSpec

	KoordSchedulerName = axisv1alpha1.SchedulerName
)
