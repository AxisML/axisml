package labels

import (
	"testing"

	axisv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
)

func TestLabels_AlignWithPublicAPI(t *testing.T) {
	cases := []struct{ got, want string }{
		{RunIDLabel, axisv1alpha1.LabelRunID},
		{QuotaLabel, axisv1alpha1.LabelQuota},
		{RoleLabel, axisv1alpha1.LabelRole},
		{SchedulerQuotaLabel, axisv1alpha1.LabelSchedulerQuota},
		{AppliedSpecAnnotation, axisv1alpha1.AnnotationAppliedSpec},
		{SchedulerName, axisv1alpha1.SchedulerName},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("label drift: %q vs %q", tc.got, tc.want)
		}
	}
}

func TestLabels_KeysAreNonEmpty(t *testing.T) {
	for _, k := range []string{
		RunIDLabel, QuotaLabel, RoleLabel, SchedulerQuotaLabel,
		AppliedSpecAnnotation, SchedulerName,
	} {
		if k == "" {
			t.Errorf("label/annotation constant is empty")
		}
	}
}
