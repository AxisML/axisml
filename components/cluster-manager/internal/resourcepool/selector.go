package resourcepool

import (
	"k8s.io/apimachinery/pkg/labels"
)

// labelSelectorFrom parses a K8s-style labelSelector string into a
// labels.Selector. Returns the empty (match-all) selector for an empty
// string.
func labelSelectorFrom(raw string) (labels.Selector, error) {
	if raw == "" {
		return labels.Everything(), nil
	}
	return labels.Parse(raw)
}
