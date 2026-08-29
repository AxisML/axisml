package admission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

func resources(values map[corev1.ResourceName]string) corev1.ResourceList {
	out := corev1.ResourceList{}
	for name, value := range values {
		out[name] = resource.MustParse(value)
	}
	return out
}

func TestNodeStatePlaceChecksSelectorAndCapacity(t *testing.T) {
	state := newNodeState(extensions.ResourceSnapshot{Nodes: []extensions.ResourceNode{{
		Name: "cpu-node", Labels: map[string]string{"pool": "cpu"},
		Allocatable: resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "4", corev1.ResourceMemory: "8Gi"}),
	}}})

	ok, reason := state.clone().place(resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}), 1, map[string]string{"pool": "gpu"}, nil)
	assert.False(t, ok)
	assert.Equal(t, ReasonNoMatchingNode, reason)

	ok, reason = state.clone().place(resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "3"}), 2, map[string]string{"pool": "cpu"}, nil)
	assert.False(t, ok)
	assert.Equal(t, ReasonInsufficientResources, reason)
}

func TestFailedTrialDoesNotBlockBackfill(t *testing.T) {
	state := newNodeState(extensions.ResourceSnapshot{Nodes: []extensions.ResourceNode{{
		Name: "node", Allocatable: resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}),
	}}})

	highPriorityTrial := state.clone()
	ok, _ := highPriorityTrial.place(resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "1500m"}), 2, nil, nil)
	require.False(t, ok)

	lowPriorityTrial := state.clone()
	ok, reason := lowPriorityTrial.place(resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}), 1, nil, nil)
	require.True(t, ok)
	assert.Empty(t, reason)
}

func TestNodeStatePlaceUsesDeterministicBestFit(t *testing.T) {
	state := newNodeState(extensions.ResourceSnapshot{Nodes: []extensions.ResourceNode{
		{Name: "large", Allocatable: resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "8"})},
		{Name: "small", Allocatable: resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"})},
	}})

	ok, reason := state.place(resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}), 1, nil, nil)
	require.True(t, ok)
	assert.Empty(t, reason)
	assert.Equal(t, int64(8_000), state.nodes[0].available.Cpu().MilliValue())
	assert.Equal(t, int64(1_000), state.nodes[1].available.Cpu().MilliValue())
}

func TestStableRoleOrderPrefersGPUThenMemoryThenCPU(t *testing.T) {
	roles := []roleRequest{
		{name: "cpu", requests: resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "8"})},
		{name: "memory", requests: resources(map[corev1.ResourceName]string{corev1.ResourceMemory: "16Gi"})},
		{name: "gpu", requests: resources(map[corev1.ResourceName]string{"nvidia.com/gpu": "1"})},
	}

	stableRoleOrder(roles)
	assert.Equal(t, []string{"gpu", "memory", "cpu"}, []string{roles[0].name, roles[1].name, roles[2].name})
}

func TestQuotaComparisonTreatsMissingLimitAsZero(t *testing.T) {
	used := resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "2", "nvidia.com/gpu": "1"})
	assert.True(t, exceeds(used, resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "4"})))
	assert.False(t, exceeds(used, resources(map[corev1.ResourceName]string{corev1.ResourceCPU: "4", "nvidia.com/gpu": "1"})))
}
