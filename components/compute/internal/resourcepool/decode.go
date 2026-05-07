package resourcepool

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
)

// DecodeNodeSelector decodes the pool's NodeSelector JSON column into a
// map. An empty/missing column returns nil. Malformed JSON is surfaced
// as an error rather than silently dropped — callers that don't care
// can ignore the error explicitly.
func (p *ResourcePool) DecodeNodeSelector() (map[string]string, error) {
	if len(p.NodeSelector) == 0 {
		return nil, nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(p.NodeSelector, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// DecodeTolerations decodes the pool's Tolerations JSON column. Empty
// returns nil; malformed JSON returns the error.
func (p *ResourcePool) DecodeTolerations() ([]corev1.Toleration, error) {
	if len(p.Tolerations) == 0 {
		return nil, nil
	}
	var t []corev1.Toleration
	if err := json.Unmarshal(p.Tolerations, &t); err != nil {
		return nil, err
	}
	return t, nil
}
