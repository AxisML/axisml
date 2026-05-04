package job

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operators/mljob-operator/api/v1alpha1"
)

// LabelTenant is the AxisML-wide convention; the mljob operator package
// doesn't export this constant directly so we declare it locally.
const LabelTenant = "axisml.io/tenant"

// ToCR materialises an MLJob CR from a PG row, ready to be applied.
// quotaName overrides the label value when non-empty; otherwise it is
// derived from spec.scheduling.quota (the same string Compute already
// computed at submission time). The mljob-operator's validation rejects
// CRs without a non-empty axisml.io/quota label.
func ToCR(j *Job, tenantName, namespace, quotaName string) (*mljobv1alpha1.MLJob, error) {
	var spec mljobv1alpha1.MLJobSpec
	if len(j.Spec) > 0 {
		if err := json.Unmarshal(j.Spec, &spec); err != nil {
			return nil, err
		}
	}
	if quotaName == "" {
		quotaName = spec.Scheduling.Quota
	}
	return &mljobv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: namespace,
			Labels: map[string]string{
				mljobv1alpha1.LabelJobID: j.ID.String(),
				mljobv1alpha1.LabelQuota: quotaName,
				LabelTenant:              tenantName,
			},
		},
		Spec: spec,
	}, nil
}
