package quota

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// TestPodRequest confirms PodRequest sums the app containers' requests so the
// plugin (admission) and controller (reported usage) agree on one number.
func TestPodRequest(t *testing.T) {
	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Resources: v1.ResourceRequirements{Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("500m"),
					v1.ResourceMemory: resource.MustParse("256Mi"),
				}}},
				{Resources: v1.ResourceRequirements{Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("500m"),
				}}},
			},
		},
	}

	got := PodRequest(pod)

	cpu := got[v1.ResourceCPU]
	if want := resource.MustParse("1"); cpu.Cmp(want) != 0 {
		t.Errorf("cpu = %s; want %s", cpu.String(), want.String())
	}
	mem := got[v1.ResourceMemory]
	if want := resource.MustParse("256Mi"); mem.Cmp(want) != 0 {
		t.Errorf("memory = %s; want %s", mem.String(), want.String())
	}
}

func TestPodRequest_Empty(t *testing.T) {
	got := PodRequest(&v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{}}}})
	if cpu, ok := got[v1.ResourceCPU]; ok && !cpu.IsZero() {
		t.Errorf("expected no cpu request, got %s", cpu.String())
	}
}
