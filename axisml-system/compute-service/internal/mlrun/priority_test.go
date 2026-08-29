package mlrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
)

func TestPriorityFromAnnotations(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]string
		want int32
		err  bool
	}{
		{name: "missing defaults to zero", want: 0},
		{name: "positive", raw: map[string]string{mlrunv1alpha1.AnnotationPriority: "42"}, want: 42},
		{name: "negative", raw: map[string]string{mlrunv1alpha1.AnnotationPriority: "-7"}, want: -7},
		{name: "int32 max", raw: map[string]string{mlrunv1alpha1.AnnotationPriority: "2147483647"}, want: 2147483647},
		{name: "overflow", raw: map[string]string{mlrunv1alpha1.AnnotationPriority: "2147483648"}, err: true},
		{name: "not decimal", raw: map[string]string{mlrunv1alpha1.AnnotationPriority: "high"}, err: true},
		{name: "empty", raw: map[string]string{mlrunv1alpha1.AnnotationPriority: ""}, err: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PriorityFromAnnotations(tt.raw)
			if tt.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
