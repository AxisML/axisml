package strutil

import "regexp"

// dns1123Like enforces the AxisML §6.1 name policy:
//   - charset [a-z0-9-]
//   - first/last char alphanumeric
//   - length 3..40
//   - no consecutive --
var dns1123Like = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,38}[a-z0-9])?$`)
var consecutiveDash = regexp.MustCompile(`--`)

// IsValidName reports whether s satisfies the AxisML name convention.
func IsValidName(s string) bool {
	if len(s) < 3 || len(s) > 40 {
		return false
	}
	if !dns1123Like.MatchString(s) {
		return false
	}
	if consecutiveDash.MatchString(s) {
		return false
	}
	return true
}

// resourceUnitName layers an additional shape on top of IsValidName.
// We accept the same charset/length constraints; downstream consumers can
// further inspect the canonical components if they need to.
func IsValidResourceUnitName(s string) bool {
	return IsValidName(s)
}
