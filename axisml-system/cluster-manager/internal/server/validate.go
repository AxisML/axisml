package server

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/validate/content"
)

// dns1123 matches DNS-1123 labels (lower-case, digits, hyphens; starts and
// ends with alphanumeric). Length 3–40 is enforced separately.
var dns1123 = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ValidateDNS1123Name checks that name is a DNS-1123 label of length 3–40
// per cluster-manager.md §3.1 ("ResourcePool 形状") and the OpenAPI pattern.
func ValidateDNS1123Name(field, value string) error {
	if len(value) < 3 || len(value) > 40 {
		return fmt.Errorf("%s length must be 3–40 (got %d)", field, len(value))
	}
	if !dns1123.MatchString(value) {
		return fmt.Errorf("%s %q does not match DNS-1123 (regex %s)", field, value, dns1123.String())
	}
	return nil
}

// ValidateResourceList checks the Kubernetes resource-name contract used by
// ResourceUnits and direct tenant quotas. Bare names are limited to native
// container resources; custom devices must use a qualified extended-resource
// name. Nvidia GPU quantities must be whole device counts.
func ValidateResourceList(field string, resources corev1.ResourceList) error {
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, string(name))
	}
	sort.Strings(names)

	for _, raw := range names {
		name := corev1.ResourceName(raw)
		if problems := content.IsQualifiedName(raw); len(problems) > 0 {
			return fmt.Errorf("%s resource %q is invalid: %s", field, raw, strings.Join(problems, "; "))
		}
		if !strings.Contains(raw, "/") &&
			name != corev1.ResourceCPU &&
			name != corev1.ResourceMemory &&
			name != corev1.ResourceEphemeralStorage &&
			!strings.HasPrefix(raw, corev1.ResourceHugePagesPrefix) {
			return fmt.Errorf("%s resource %q must be cpu, memory, ephemeral-storage, hugepages-*, or a qualified extended resource", field, raw)
		}

		quantity := resources[name]
		if quantity.Sign() < 0 {
			return fmt.Errorf("%s resource %q must be non-negative", field, raw)
		}
		if name == corev1.ResourceName("nvidia.com/gpu") {
			if _, exact := quantity.AsInt64(); !exact {
				return fmt.Errorf("%s resource %q must be a whole number", field, raw)
			}
		}
	}
	return nil
}
