package mlrun

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
)

// ToCR materialises an MLRun CR from a PG row. Pool/unit provenance is
// read out of mlruns.labels[resource.axisml.io/pool / -unit] (compute-
// service.md §5.4). The compute.axisml.io/quota label is sourced from
// spec.scheduling.quota; compute-operator's Validate rejects CRs missing it.
func ToCR(j *store.MLRun, tenantPrefix bool) (*mlrunv1alpha1.MLRun, error) {
	var spec mlrunv1alpha1.MLRunSpec
	if len(j.Spec) > 0 {
		if err := json.Unmarshal(j.Spec, &spec); err != nil {
			return nil, err
		}
	}
	rowLabels := decodeStringMap(j.Labels)
	rowAnnotations := decodeStringMap(j.Annotations)
	labels := map[string]string{
		mlrunv1alpha1.LabelRunID:  j.ID.String(),
		mlrunv1alpha1.LabelTenant: j.Namespace,
		mlrunv1alpha1.LabelQuota:  spec.Scheduling.Quota,
	}
	for _, k := range []string{
		mlrunv1alpha1.LabelResourcePool,
		mlrunv1alpha1.LabelResourceUnit,
	} {
		if v, ok := rowLabels[k]; ok && v != "" {
			labels[k] = v
		}
	}
	cr := &mlrunv1alpha1.MLRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        j.Name,
			Namespace:   j.Namespace,
			Labels:      labels,
			Annotations: reservedAnnotations(rowAnnotations),
		},
		Spec: spec,
	}
	workloadname.Annotate(cr, j.Namespace, j.Name, tenantPrefix)
	return cr, nil
}

func reservedAnnotations(row map[string]string) map[string]string {
	if value, ok := row[mlrunv1alpha1.AnnotationPriority]; ok {
		return map[string]string{mlrunv1alpha1.AnnotationPriority: value}
	}
	return nil
}

func decodeStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	m := map[string]string{}
	_ = json.Unmarshal(raw, &m)
	return m
}
