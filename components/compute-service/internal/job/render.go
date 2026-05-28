package job

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mljobv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mljob/v1alpha1"
)

// ToCR materialises an MLJob CR from a PG row. Pool/unit provenance is
// read out of jobs.labels[axisml.io/resource-pool / -unit] (compute-
// service.md §5.4). The axisml.io/quota label is sourced from
// spec.scheduling.quota; compute-operator's Validate rejects CRs missing it.
func ToCR(j *Job) (*mljobv1alpha1.MLJob, error) {
	var spec mljobv1alpha1.MLJobSpec
	if len(j.Spec) > 0 {
		if err := json.Unmarshal(j.Spec, &spec); err != nil {
			return nil, err
		}
	}
	rowLabels := decodeStringMap(j.Labels)
	labels := map[string]string{
		mljobv1alpha1.LabelJobID:  j.ID.String(),
		mljobv1alpha1.LabelTenant: j.Namespace,
		mljobv1alpha1.LabelQuota:  spec.Scheduling.Quota,
	}
	for _, k := range []string{
		mljobv1alpha1.LabelResourcePool,
		mljobv1alpha1.LabelResourceUnit,
	} {
		if v, ok := rowLabels[k]; ok && v != "" {
			labels[k] = v
		}
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

func decodeStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	m := map[string]string{}
	_ = json.Unmarshal(raw, &m)
	return m
}
