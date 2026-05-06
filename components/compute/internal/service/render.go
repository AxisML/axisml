package service

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// ToCR materialises an MLService CR from a PG row. After de-tenant
// rewrite the namespace comes straight from the row; tenant label is
// no longer set.
func ToCR(s *Service) (*mlservicev1alpha1.MLService, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if len(s.Spec) > 0 {
		if err := json.Unmarshal(s.Spec, &spec); err != nil {
			return nil, err
		}
	}
	return &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
			Labels: map[string]string{
				mlservicev1alpha1.LabelServiceID: s.ID.String(),
				mlservicev1alpha1.LabelQuota:     spec.Scheduling.Quota,
			},
		},
		Spec: spec,
	}, nil
}
