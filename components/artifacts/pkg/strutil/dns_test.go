package strutil

import "testing"

func TestIsValidName(t *testing.T) {
	cases := map[string]bool{
		"default":       true,
		"my-repo":       true,
		"a1-b2-c3":      true,
		"ab":            false,
		"-leading":      false,
		"trailing-":     false,
		"UPPER":         false,
		"con--secutive": false,
		"a b":           false,
		"":              false,
	}
	for name, want := range cases {
		if got := IsValidName(name); got != want {
			t.Errorf("IsValidName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsValidVersion(t *testing.T) {
	cases := map[string]bool{
		"v1":         true,
		"v1.0.0":     true,
		"2024-09":    true,
		"sha-abcdef": true,
		"V1_BETA":    true,
		"":           false,
		"v1/v2":      false,
		"with space": false,
	}
	for v, want := range cases {
		if got := IsValidVersion(v); got != want {
			t.Errorf("IsValidVersion(%q) = %v, want %v", v, got, want)
		}
	}
}
