package standalone

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// fakeGPUDocker is an in-memory gpuDockerClient: it returns canned container
// summaries for the occupancy scan and canned inspects for legacy containers,
// so the allocator's Docker seam is testable without a daemon.
type fakeGPUDocker struct {
	list    []container.Summary
	inspect map[string]container.InspectResponse
}

func (f *fakeGPUDocker) ContainerList(context.Context, container.ListOptions) ([]container.Summary, error) {
	return f.list, nil
}

func (f *fakeGPUDocker) ContainerInspect(_ context.Context, id string) (container.InspectResponse, error) {
	return f.inspect[id], nil
}

func summary(state string, labels map[string]string) container.Summary {
	return container.Summary{State: state, Labels: labels}
}

func TestBusyGPUDevices_FromLabels(t *testing.T) {
	fake := &fakeGPUDocker{list: []container.Summary{
		summary("running", map[string]string{LabelGPUDevices: "0"}),
		summary("running", map[string]string{LabelGPUDevices: "2,3"}),
		summary("exited", map[string]string{LabelGPUDevices: "1"}),  // released — not counted
		summary("running", map[string]string{}),                     // non-GPU container
		summary("created", map[string]string{LabelGPUDevices: "5"}), // created counts as busy
	}}
	busy, untracked, err := busyGPUDevices(context.Background(), fake)
	require.NoError(t, err)
	assert.Equal(t, 0, untracked)
	assert.Equal(t, map[int]struct{}{0: {}, 2: {}, 3: {}, 5: {}}, busy)
}

func TestBusyGPUDevices_LegacyUntracked(t *testing.T) {
	// A label-less, alive container that reserves 2 nvidia GPUs (pre-feature
	// count-based) must be counted as untracked so its cards are not double-booked.
	fake := &fakeGPUDocker{
		list: []container.Summary{
			summary("running", map[string]string{LabelGPUDevices: "0"}), // tracked
			{ID: "legacy", State: "running", Labels: map[string]string{}},
			{ID: "legacy-nogpu", State: "running", Labels: map[string]string{}},
		},
		inspect: map[string]container.InspectResponse{
			"legacy": {ContainerJSONBase: &container.ContainerJSONBase{HostConfig: &container.HostConfig{
				Resources: container.Resources{DeviceRequests: []container.DeviceRequest{
					{Driver: "nvidia", Count: 2, Capabilities: [][]string{{"gpu"}}},
				}},
			}}},
			"legacy-nogpu": {ContainerJSONBase: &container.ContainerJSONBase{HostConfig: &container.HostConfig{}}},
		},
	}
	busy, untracked, err := busyGPUDevices(context.Background(), fake)
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{0: {}}, busy)
	assert.Equal(t, 2, untracked)
}

func newTestRuntimeWithGPUs(devices ...int) *Runtime {
	return New(nil, Config{WorkloadsNetwork: "axisml-workloads", GPUDevices: devices}, logr.Discard())
}

func gpuPlan(n int) *ContainerPlan {
	return &ContainerPlan{
		Name:      "p",
		Image:     "busybox",
		Labels:    map[string]string{},
		Resources: ResourcePlan{GPUCount: n},
	}
}

func TestAssignGPUsLocked_EndToEnd(t *testing.T) {
	t.Run("assigns free cards and pins them onto the plan", func(t *testing.T) {
		r := newTestRuntimeWithGPUs(0, 1, 2, 3)
		fake := &fakeGPUDocker{list: []container.Summary{
			summary("running", map[string]string{LabelGPUDevices: "0"}),
			summary("running", map[string]string{LabelGPUDevices: "2"}),
		}}
		p := gpuPlan(1)
		require.NoError(t, r.assignGPUsLocked(context.Background(), fake, []*ContainerPlan{p}))
		// free cards were 1 and 3; the single-GPU plan takes the first free one.
		assert.Equal(t, []string{"1"}, p.Resources.GPUDeviceIDs)
		assert.Equal(t, "1", p.Labels[LabelGPUDevices])
	})

	t.Run("returns ResourceUnavailable when the host is full, pinning nothing", func(t *testing.T) {
		r := newTestRuntimeWithGPUs(0, 1)
		fake := &fakeGPUDocker{list: []container.Summary{
			summary("running", map[string]string{LabelGPUDevices: "0"}),
			summary("running", map[string]string{LabelGPUDevices: "1"}),
		}}
		p := gpuPlan(1)
		err := r.assignGPUsLocked(context.Background(), fake, []*ContainerPlan{p})
		require.Error(t, err)
		assert.True(t, extensions.IsResourceUnavailable(err))
		assert.Empty(t, p.Resources.GPUDeviceIDs, "no card should be pinned on failure")
		assert.Empty(t, p.Labels[LabelGPUDevices])
	})

	t.Run("multi-plan workload gets distinct cards atomically", func(t *testing.T) {
		r := newTestRuntimeWithGPUs(0, 1, 2, 3)
		fake := &fakeGPUDocker{list: []container.Summary{
			summary("running", map[string]string{LabelGPUDevices: "1"}),
		}}
		p0, p1 := gpuPlan(1), gpuPlan(1)
		require.NoError(t, r.assignGPUsLocked(context.Background(), fake, []*ContainerPlan{p0, p1}))
		// free = 0,2,3 -> plans get 0 and 2, never overlapping.
		assert.Equal(t, []string{"0"}, p0.Resources.GPUDeviceIDs)
		assert.Equal(t, []string{"2"}, p1.Resources.GPUDeviceIDs)
	})
}
