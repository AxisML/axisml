package standalone

import (
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
)

const nvidiaGPUResource = corev1.ResourceName("nvidia.com/gpu")

var standaloneResourceNames = map[corev1.ResourceName]struct{}{
	corev1.ResourceCPU:    {},
	corev1.ResourceMemory: {},
	nvidiaGPUResource:     {},
}

// validateStandaloneResourceList limits scheduler-facing resource maps to the
// dimensions the Docker runtime inventories and enforces.
func validateStandaloneResourceList(field string, resources corev1.ResourceList) error {
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, string(name))
	}
	sort.Strings(names)

	for _, raw := range names {
		name := corev1.ResourceName(raw)
		if _, ok := standaloneResourceNames[name]; !ok {
			return fmt.Errorf("%s resource %q is not supported in standalone; supported resources are cpu, memory, nvidia.com/gpu", field, raw)
		}
		quantity := resources[name]
		if quantity.Sign() < 0 {
			return fmt.Errorf("%s resource %q must be non-negative", field, raw)
		}
		if name == nvidiaGPUResource {
			if _, exact := quantity.AsInt64(); !exact {
				return fmt.Errorf("%s resource %q must be a whole number", field, raw)
			}
		}
	}
	return nil
}
