package mlservice

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/store"
)

// ToCR materialises an MLService CR from a PG row. Per design §6 / §5.2
// the CR carries the stable label set: service-id (UUID anchor),
// service-kind (`service`|`workspace`, for kubectl selectors), tenant
// (= partition namespace), and quota. Pool/unit provenance is read from
// the row's labels jsonb.
func ToCR(s *store.MLService) (*mlservicev1alpha1.MLService, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if len(s.Spec) > 0 {
		if err := json.Unmarshal(s.Spec, &spec); err != nil {
			return nil, err
		}
	}
	kind := s.Kind
	if kind == "" {
		kind = mlservicev1alpha1.ServiceKindService
	}
	rowLabels := decodeStringMap(s.Labels)
	labels := map[string]string{
		mlservicev1alpha1.LabelServiceID:   s.ID.String(),
		mlservicev1alpha1.LabelServiceKind: kind,
		mlservicev1alpha1.LabelTenant:      s.Namespace,
		mlservicev1alpha1.LabelQuota:       spec.Scheduling.Quota,
	}
	for _, k := range []string{
		mlservicev1alpha1.LabelResourcePool,
		mlservicev1alpha1.LabelResourceUnit,
	} {
		if v, ok := rowLabels[k]; ok && v != "" {
			labels[k] = v
		}
	}
	return &mlservicev1alpha1.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
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
