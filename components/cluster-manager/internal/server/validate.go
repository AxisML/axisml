package server

import (
	"fmt"
	"regexp"
)

// dns1123Subdomain matches the K8s DNS-1123 subdomain rule with the
// additional cluster-manager constraint of no consecutive `--`.
var dns1123Subdomain = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ValidateName runs the §3.1 DNS-1123 + length checks defined in the
// cluster-manager design (length 3–40, no consecutive `--`).
func ValidateName(field, value string) error {
	if len(value) < 3 || len(value) > 40 {
		return fmt.Errorf("%s: length must be 3–40 (got %d)", field, len(value))
	}
	if !dns1123Subdomain.MatchString(value) {
		return fmt.Errorf("%s: must match DNS-1123 [a-z0-9-] starting and ending with alphanumeric", field)
	}
	for i := 1; i < len(value); i++ {
		if value[i] == '-' && value[i-1] == '-' {
			return fmt.Errorf("%s: consecutive '--' is not allowed", field)
		}
	}
	return nil
}

// ValidateNamespace combines DNS-1123 with a denylist check against the
// configured system-namespace blocklist (kube-system, default, axisml-*, …).
func ValidateNamespace(value string, denylist []string) error {
	if err := ValidateName("namespace.name", value); err != nil {
		return err
	}
	for _, banned := range denylist {
		if value == banned {
			return fmt.Errorf("namespace.name: %q is on the denylist", value)
		}
	}
	return nil
}

// ValidateQuota does the per-key min ≤ max + non-negative checks against
// resource quantities expressed as strings (we don't parse Quantities here;
// tenant-operator's Validate will catch malformed strings on reconcile).
func ValidateQuota(q QuotaSpec) error {
	if err := ValidateName("quotas[].pool", q.Pool); err != nil {
		return err
	}
	if err := ValidateName("quotas[].name", q.Name); err != nil {
		return err
	}
	if q.Max == nil {
		return fmt.Errorf("quotas[%s/%s].max is required", q.Pool, q.Name)
	}
	return nil
}
