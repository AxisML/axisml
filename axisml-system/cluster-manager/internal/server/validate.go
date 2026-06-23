package server

import (
	"fmt"
	"regexp"
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
