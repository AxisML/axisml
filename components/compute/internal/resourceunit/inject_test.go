package resourceunit

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestBuildResources(t *testing.T) {
	cases := []struct {
		name   string
		req    corev1.ResourceList
		lim    corev1.ResourceList
		hasReq bool
		hasLim bool
	}{
		{"both empty", nil, nil, false, false},
		{"requests only", corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")}, nil, true, false},
		{"limits only", nil, corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")}, false, true},
		{"both", corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")},
			corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m")}, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := BuildResources(tc.req, tc.lim)
			if tc.hasReq {
				if rr.Requests == nil {
					t.Error("expected Requests populated")
				}
			} else if rr.Requests != nil {
				t.Errorf("expected nil Requests, got %v", rr.Requests)
			}
			if tc.hasLim {
				if rr.Limits == nil {
					t.Error("expected Limits populated")
				}
			} else if rr.Limits != nil {
				t.Errorf("expected nil Limits, got %v", rr.Limits)
			}
		})
	}
}
