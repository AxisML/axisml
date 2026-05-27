package job

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
)

// ToCR materialises an MLJob CR from a PG row, ready to be applied. The
// axisml.io/quota label is sourced from spec.scheduling.quota; compute-
// operator's Validate rejects CRs without it.
func ToCR(j *Job) (*mljobv1alpha1.MLJob, error) {
	var spec mljobv1alpha1.MLJobSpec
	if len(j.Spec) > 0 {
		if err := json.Unmarshal(j.Spec, &spec); err != nil {
			return nil, err
		}
	}
	labels := map[string]string{
		mljobv1alpha1.LabelJobID:  j.ID.String(),
		mljobv1alpha1.LabelTenant: j.Namespace,
		mljobv1alpha1.LabelQuota:  spec.Scheduling.Quota,
	}
	if j.PoolName != "" {
		labels[mljobv1alpha1.LabelResourcePool] = j.PoolName
	}
	if j.UnitName != "" {
		labels[mljobv1alpha1.LabelResourceUnit] = j.UnitName
	}
	return &mljobv1alpha1.MLJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: j.Namespace,
			Labels:    labels,
		},
		Spec: spec,
	}, nil
}
