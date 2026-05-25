package resourcepool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
)

func TestDecodeNodeSelector(t *testing.T) {
	cases := []struct {
		name string
		in   datatypes.JSON
		want map[string]string
		err  bool
	}{
		{"empty bytes", datatypes.JSON{}, nil, false},
		{"empty object", datatypes.JSON([]byte(`{}`)), map[string]string{}, false},
		{"populated", datatypes.JSON([]byte(`{"axisml.io/pool":"gpu-a100"}`)),
			map[string]string{"axisml.io/pool": "gpu-a100"}, false},
		{"malformed", datatypes.JSON([]byte(`{not-json`)), nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &ResourcePool{NodeSelector: tc.in}
			got, err := p.DecodeNodeSelector()
			if tc.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecodeTolerations(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		p := &ResourcePool{}
		got, err := p.DecodeTolerations()
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("populated", func(t *testing.T) {
		raw := `[{"key":"nvidia.com/gpu","operator":"Exists","effect":"NoSchedule"}]`
		p := &ResourcePool{Tolerations: datatypes.JSON([]byte(raw))}
		got, err := p.DecodeTolerations()
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "nvidia.com/gpu", got[0].Key)
		assert.Equal(t, corev1.TolerationOpExists, got[0].Operator)
		assert.Equal(t, corev1.TaintEffectNoSchedule, got[0].Effect)
	})

	t.Run("malformed", func(t *testing.T) {
		p := &ResourcePool{Tolerations: datatypes.JSON([]byte(`[not-json`))}
		_, err := p.DecodeTolerations()
		assert.Error(t, err)
	})
}
