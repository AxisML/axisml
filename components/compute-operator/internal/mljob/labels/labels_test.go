package labels

import (
	"testing"

	axisv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
)

func TestLabels_AlignWithPublicAPI(t *testing.T) {
	cases := []struct{ got, want string }{
		{JobIDLabel, axisv1alpha1.LabelJobID},
		{QuotaLabel, axisv1alpha1.LabelQuota},
		{RoleLabel, axisv1alpha1.LabelRole},
		{KoordQuotaLabel, axisv1alpha1.LabelKoordQuotaName},
		{PodGroupLabel, axisv1alpha1.LabelPodGroup},
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
		JobIDLabel, QuotaLabel, RoleLabel, KoordQuotaLabel,
		PodGroupLabel, AppliedSpecAnnotation, KoordSchedulerName,
	} {
		if k == "" {
			t.Errorf("label/annotation constant is empty")
		}
	}
}
