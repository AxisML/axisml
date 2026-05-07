package server

import "testing"

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"happy", "team-a", false},
		{"min length", "abc", false},
		{"max length", "a234567890123456789012345678901234567890", false},
		{"too short", "ab", true},
		{"too long", "a23456789012345678901234567890123456789012", true},
		{"uppercase", "Team-A", true},
		{"trailing dash", "team-", true},
		{"leading dash", "-team", true},
		{"consecutive dashes", "te--am", true},
		{"underscore", "team_a", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName("name", tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v want err=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateNamespaceDenylist(t *testing.T) {
	deny := []string{"kube-system", "default", "axisml-system"}
	if err := ValidateNamespace("kube-system", deny); err == nil {
		t.Fatal("expected denylist hit")
	}
	if err := ValidateNamespace("team-a", deny); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}
}

func TestValidateQuota(t *testing.T) {
	if err := ValidateQuota(QuotaSpec{Pool: "default", Name: "default", Max: map[string]string{"cpu": "10"}}); err != nil {
		t.Fatalf("happy path failed: %v", err)
	}
	if err := ValidateQuota(QuotaSpec{Pool: "default", Name: "default"}); err == nil {
		t.Fatal("expected missing max to fail")
	}
}
