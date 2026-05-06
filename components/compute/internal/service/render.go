package service

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// ToCR materialises an MLService CR from a PG row. The axisml.io/quota
// label is sourced from spec.scheduling.quota; mlservice-operator's
// Validate rejects CRs without it.
func ToCR(s *Service, tenantName, namespace string) (*mlservicev1alpha1.MLService, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if len(s.Spec) > 0 {
		if err := json.Unmarshal(s.Spec, &spec); err != nil {
			return nil, err
		}
	}
	return &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: namespace,
			Labels: map[string]string{
				mlservicev1alpha1.LabelServiceID: s.ID.String(),
				mlservicev1alpha1.LabelTenant:    tenantName,
				mlservicev1alpha1.LabelQuota:     spec.Scheduling.Quota,
			},
		},
		Spec: spec,
	}, nil
}
