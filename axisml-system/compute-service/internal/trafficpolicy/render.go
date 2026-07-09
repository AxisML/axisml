package trafficpolicy

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
)

// ToCR materialises an MLTrafficPolicy CR from a PG row. The CR carries the
// stable anchor label (traffic-policy-id = row UUID) plus the tenant label so
// the operator can stamp the derived HTTPRoute and the dispatcher can filter
// owned children.
func ToCR(p *store.TrafficPolicy) (*mltp.MLTrafficPolicy, error) {
	var spec mltp.MLTrafficPolicySpec
	if len(p.Spec) > 0 {
		if err := json.Unmarshal(p.Spec, &spec); err != nil {
			return nil, err
		}
	}
	labels := map[string]string{
		mltp.LabelTrafficPolicyID: p.ID.String(),
		mltp.LabelTenant:          p.Namespace,
	}
	return &mltp.MLTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
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
