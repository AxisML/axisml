package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
)

func TestValidateDNS1123Name(t *testing.T) {
	cases := []struct {
		name  string
		value string
		ok    bool
	}{
		{"valid", "gpu-a100", true},
		{"min length", "abc", true},
		{"max length", strings.Repeat("a", 40), true},
		{"too short", "ab", false},
		{"too long", strings.Repeat("a", 41), false},
		{"uppercase rejected", "GPU", false},
		{"leading hyphen", "-abc", false},
		{"trailing hyphen", "abc-", false},
		{"underscore rejected", "a_b_c", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := srv.ValidateDNS1123Name("field", tc.value)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "field")
			}
		})
	}
}

func TestValidateResourceList(t *testing.T) {
	tests := []struct {
		name    string
		list    corev1.ResourceList
		wantSub string
	}{
		{name: "native resources", list: corev1.ResourceList{
			corev1.ResourceCPU:              resource.MustParse("500m"),
			corev1.ResourceMemory:           resource.MustParse("1Gi"),
			corev1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
		}},
		{name: "qualified extended resource", list: corev1.ResourceList{"example.com/dongle": resource.MustParse("1")}},
		{name: "huge pages", list: corev1.ResourceList{"hugepages-2Mi": resource.MustParse("2Mi")}},
		{name: "unqualified gpu typo", list: corev1.ResourceList{"gpu": resource.MustParse("1")}, wantSub: `resource "gpu"`},
		{name: "malformed name", list: corev1.ResourceList{"bad resource": resource.MustParse("1")}, wantSub: "is invalid"},
		{name: "negative quantity", list: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("-1")}, wantSub: "must be non-negative"},
		{name: "fractional GPU", list: corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("0.5")}, wantSub: "must be a whole number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.ValidateResourceList("resources", tt.list)
			if tt.wantSub == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantSub)
		})
	}
}
