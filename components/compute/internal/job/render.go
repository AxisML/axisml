package job

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
)

// LabelTenant is the AxisML-wide convention; the mljob operator package
// doesn't export this constant directly so we declare it locally.
const LabelTenant = "axisml.io/tenant"

// ToCR materialises an MLJob CR from a PG row, ready to be applied. The
// axisml.io/quota label is sourced from spec.scheduling.quota (Compute
// already rendered the canonical name at submission); mljob-operator's
// Validate rejects CRs without it.
func ToCR(j *Job, tenantName, namespace string) (*mljobv1alpha1.MLJob, error) {
	var spec mljobv1alpha1.MLJobSpec
	if len(j.Spec) > 0 {
		if err := json.Unmarshal(j.Spec, &spec); err != nil {
			return nil, err
		}
	}
	return &mljobv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: namespace,
			Labels: map[string]string{
				mljobv1alpha1.LabelJobID: j.ID.String(),
				mljobv1alpha1.LabelQuota: spec.Scheduling.Quota,
				LabelTenant:              tenantName,
			},
		},
		Spec: spec,
	}, nil
}
