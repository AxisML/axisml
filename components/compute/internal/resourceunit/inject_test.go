package resourceunit

import (
	"reflect"
	"testing"
)

func TestMergeNodeSelectorPoolWins(t *testing.T) {
	pool := map[string]string{"axisml.io/pool": "gpu-a100"}
	unit := map[string]string{
		"axisml.io/pool":         "ignored",
		"nvidia.com/gpu.product": "A100-SXM4-80GB",
	}
	got := MergeNodeSelector(pool, unit)
	want := map[string]string{
		"axisml.io/pool":         "gpu-a100",
		"nvidia.com/gpu.product": "A100-SXM4-80GB",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MergeNodeSelector = %v, want %v", got, want)
	}
}

func TestMergeNodeSelectorEmpty(t *testing.T) {
	got := MergeNodeSelector(nil, nil)
	if got != nil {
		t.Errorf("MergeNodeSelector(nil, nil) = %v, want nil", got)
	}
}
