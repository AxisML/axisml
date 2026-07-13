package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

func TestResolveGPUDevices(t *testing.T) {
	t.Run("empty disables GPU", func(t *testing.T) {
		devs, err := ResolveGPUDevices("")
		require.NoError(t, err)
		assert.Empty(t, devs)
	})
	t.Run("explicit list is parsed, deduped and sorted", func(t *testing.T) {
		devs, err := ResolveGPUDevices(" 2, 0 ,1, 0 ")
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2}, devs)
	})
	t.Run("invalid index errors", func(t *testing.T) {
		_, err := ResolveGPUDevices("0,x")
		require.Error(t, err)
		_, err = ResolveGPUDevices("-1")
		require.Error(t, err)
	})
}

func TestNewGPUAllocatorDedupSorts(t *testing.T) {
	a := newGPUAllocator([]int{3, 1, 1, -1, 2})
	assert.Equal(t, []int{1, 2, 3}, a.schedulable) // negatives dropped, deduped, sorted
}

func TestGPUAlive(t *testing.T) {
	for _, s := range []string{"running", "restarting", "created", "paused"} {
		assert.True(t, gpuAlive(s), "state %q should hold its card", s)
	}
	for _, s := range []string{"exited", "dead", "removing", ""} {
		assert.False(t, gpuAlive(s), "state %q should have released its card", s)
	}
}

func TestParseDeviceList(t *testing.T) {
	assert.Equal(t, []int{0, 2}, parseDeviceList("0,2"))
	assert.Equal(t, []int{1}, parseDeviceList(" 1 "))
	assert.Nil(t, parseDeviceList(""))
	assert.Equal(t, []int{3}, parseDeviceList("x,3")) // non-numeric skipped
}

func busySet(ids ...int) map[int]struct{} {
	m := map[int]struct{}{}
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func TestComputeAssignment(t *testing.T) {
	t.Run("assigns free cards, skipping busy ones", func(t *testing.T) {
		// schedulable 0,1,2,3; 0 and 2 busy -> free 1,3.
		got, err := computeAssignment([]int{0, 1, 2, 3}, busySet(0, 2), 0, []int{1, 1})
		require.NoError(t, err)
		assert.Equal(t, [][]int{{1}, {3}}, got)
	})

	t.Run("multi-card plan gets a contiguous slice of free cards", func(t *testing.T) {
		got, err := computeAssignment([]int{0, 1, 2, 3}, busySet(1), 0, []int{2})
		require.NoError(t, err)
		assert.Equal(t, [][]int{{0, 2}}, got) // free = 0,2,3 -> first two
	})

	t.Run("zero-GPU plans get no assignment", func(t *testing.T) {
		got, err := computeAssignment([]int{0, 1}, nil, 0, []int{0, 1, 0})
		require.NoError(t, err)
		assert.Equal(t, [][]int{nil, {0}, nil}, got)
	})

	t.Run("transient: not enough free cards", func(t *testing.T) {
		_, err := computeAssignment([]int{0, 1}, busySet(0), 0, []int{2})
		require.Error(t, err)
		assert.True(t, extensions.IsResourceUnavailable(err))
		assert.Contains(t, err.Error(), "等待可用 GPU")
	})

	t.Run("infeasible: request exceeds capacity", func(t *testing.T) {
		_, err := computeAssignment([]int{0, 1}, nil, 0, []int{3})
		require.Error(t, err)
		assert.True(t, extensions.IsResourceUnavailable(err))
		assert.Contains(t, err.Error(), "超过可调度容量")
	})

	t.Run("untracked legacy GPUs reduce availability", func(t *testing.T) {
		// schedulable 0,1; none label-busy but 1 untracked legacy card -> 1 free.
		got, err := computeAssignment([]int{0, 1}, nil, 1, []int{1})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Len(t, got[0], 1)

		_, err = computeAssignment([]int{0, 1}, nil, 1, []int{2})
		require.Error(t, err)
		assert.True(t, extensions.IsResourceUnavailable(err))
	})
}
