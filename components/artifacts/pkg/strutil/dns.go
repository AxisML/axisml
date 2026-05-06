package strutil

import "regexp"

// dns1123Like enforces the AxisML name policy:
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

// ociTagSafe enforces design §4.1: OCI tag-safe subset, length 1..128.
var ociTagSafe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

// IsValidVersion reports whether s satisfies the OCI tag-safe subset used for
// artifact versions (design §4.1).
func IsValidVersion(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	return ociTagSafe.MatchString(s)
}
