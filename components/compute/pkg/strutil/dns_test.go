package strutil

import "testing"

func TestIsValidName(t *testing.T) {
	cases := map[string]bool{
		"default":       true,
		"my-tenant":     true,
		"a1-b2-c3":      true,
		"ab":            false, // too short
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
