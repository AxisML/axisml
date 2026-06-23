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

func TestIsValidResourceUnitName_DelegatesToIsValidName(t *testing.T) {
	if !IsValidResourceUnitName("ru-1") {
		t.Error("expected true for valid resource unit name")
	}
	if IsValidResourceUnitName("Bad") {
		t.Error("expected false for invalid name (uppercase)")
	}
}

func TestIsValidName_BoundaryLengths(t *testing.T) {
	if !IsValidName("abc") {
		t.Error("3-char name should be valid")
	}
	long40 := "a234567890123456789012345678901234567890" // 40 chars
	if !IsValidName(long40) {
		t.Errorf("40-char name should be valid; got false for %s (len=%d)", long40, len(long40))
	}
	long41 := long40 + "1"
	if IsValidName(long41) {
		t.Error("41-char name should be invalid")
	}
}
