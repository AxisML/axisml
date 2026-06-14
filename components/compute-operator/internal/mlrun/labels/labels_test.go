package labels

import (
	"testing"

	axisv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
)

func TestLabels_AlignWithPublicAPI(t *testing.T) {
	cases := []struct{ got, want string }{
		{RunIDLabel, axisv1alpha1.LabelRunID},
		{QuotaLabel, axisv1alpha1.LabelQuota},
		{RoleLabel, axisv1alpha1.LabelRole},
		{KoordQuotaLabel, axisv1alpha1.LabelKoordQuotaName},
		{AppliedSpecAnnotation, axisv1alpha1.AnnotationAppliedSpec},
		{KoordSchedulerName, axisv1alpha1.SchedulerName},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("label drift: %q vs %q", tc.got, tc.want)
		}
	}
}

func TestLabels_KeysAreNonEmpty(t *testing.T) {
	for _, k := range []string{
		RunIDLabel, QuotaLabel, RoleLabel, KoordQuotaLabel,
		AppliedSpecAnnotation, KoordSchedulerName,
	} {
		if k == "" {
			t.Errorf("label/annotation constant is empty")
		}
	}
}
