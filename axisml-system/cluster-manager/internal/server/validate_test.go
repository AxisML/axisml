package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
