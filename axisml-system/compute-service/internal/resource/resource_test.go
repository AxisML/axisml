package resource_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	res "github.com/axisml/axisml/axisml-system/compute-service/internal/resource"
)

func TestExpand_NodeSelectorMerge(t *testing.T) {
	tolerations := []corev1.Toleration{{
		Key:      "gpu",
		Operator: corev1.TolerationOpExists,
		Effect:   corev1.TaintEffectNoSchedule,
	}}
	requests := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")}
	limits := corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("4Gi")}

	tests := []struct {
		name     string
		poolSel  map[string]string
		unitSel  map[string]string
		wantSel  map[string]string // nil means expect nil
		wantNilM bool
	}{
		{
			name:     "both empty yields nil",
			poolSel:  nil,
			unitSel:  nil,
			wantNilM: true,
		},
		{
			name:    "pool keys win, unit-only keys fill gaps",
			poolSel: map[string]string{"zone": "a", "shared": "pool"},
			unitSel: map[string]string{"shared": "unit", "gpu": "true"},
			wantSel: map[string]string{"zone": "a", "shared": "pool", "gpu": "true"},
		},
		{
			name:    "pool only",
			poolSel: map[string]string{"zone": "a"},
			unitSel: nil,
			wantSel: map[string]string{"zone": "a"},
		},
		{
			name:    "unit only",
			poolSel: nil,
			unitSel: map[string]string{"gpu": "true"},
			wantSel: map[string]string{"gpu": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &cmv1alpha1.ResourcePool{
				Spec: cmv1alpha1.ResourcePoolSpec{
					NodeSelector: tt.poolSel,
					Tolerations:  tolerations,
				},
			}
			unit := &cmv1alpha1.ResourceUnit{
				Name:         "small",
				NodeSelector: tt.unitSel,
				Requests:     requests,
				Limits:       limits,
			}

			got := res.Expand(pool, unit)

			if tt.wantNilM {
				assert.Nil(t, got.NodeSelector)
			} else {
				assert.Equal(t, tt.wantSel, got.NodeSelector)
			}
			// Tolerations pass through verbatim from the pool.
			assert.Equal(t, tolerations, got.Tolerations)
			// Requests/limits copied straight off the unit.
			assert.Equal(t, requests, got.Requests)
			assert.Equal(t, limits, got.Limits)
		})
	}
}

func TestExpand_DoesNotMutateInputs(t *testing.T) {
	pool := &cmv1alpha1.ResourcePool{
		Spec: cmv1alpha1.ResourcePoolSpec{NodeSelector: map[string]string{"zone": "a"}},
	}
	unit := &cmv1alpha1.ResourceUnit{NodeSelector: map[string]string{"gpu": "true"}}

	got := res.Expand(pool, unit)
	got.NodeSelector["mutated"] = "x"

	// The merge allocates a fresh map; the pool/unit selectors are untouched.
	assert.NotContains(t, pool.Spec.NodeSelector, "mutated")
	assert.NotContains(t, unit.NodeSelector, "mutated")
	assert.Len(t, pool.Spec.NodeSelector, 1)
	assert.Len(t, unit.NodeSelector, 1)
}

func TestBuildResources(t *testing.T) {
	req := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}
	lim := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")}

	tests := []struct {
		name       string
		req, lim   corev1.ResourceList
		wantReqNil bool
		wantLimNil bool
	}{
		{name: "both set", req: req, lim: lim},
		{name: "requests only", req: req, lim: nil, wantLimNil: true},
		{name: "limits only", req: nil, lim: lim, wantReqNil: true},
		{name: "both empty", req: nil, lim: nil, wantReqNil: true, wantLimNil: true},
		{name: "both empty non-nil maps", req: corev1.ResourceList{}, lim: corev1.ResourceList{}, wantReqNil: true, wantLimNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := res.BuildResources(tt.req, tt.lim)
			if tt.wantReqNil {
				assert.Nil(t, rr.Requests)
			} else {
				assert.Equal(t, tt.req, rr.Requests)
			}
			if tt.wantLimNil {
				assert.Nil(t, rr.Limits)
			} else {
				assert.Equal(t, tt.lim, rr.Limits)
			}
		})
	}
}
