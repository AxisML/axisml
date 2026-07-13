package standalone

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// gpuAllocator tracks the set of physical GPU indices the host has handed to
// AxisML for scheduling. Occupancy is not kept in a ledger: it is recomputed
// from the io.axisml.gpu-devices labels of the currently-alive managed
// containers on every admission, so the allocator is restart- and crash-safe.
// The mutex serialises the "compute free → assign → create" critical section so
// two concurrent Applies (Run and Service share one Runtime) cannot double-book
// a card.
type gpuAllocator struct {
	schedulable []int
	mu          sync.Mutex
}

func newGPUAllocator(devices []int) *gpuAllocator {
	seen := map[int]bool{}
	var out []int
	for _, d := range devices {
		if d >= 0 && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Ints(out)
	return &gpuAllocator{schedulable: out}
}

// ResolveGPUDevices parses the AXISML_GPU_DEVICES value into the set of
// schedulable physical GPU indices. A comma list ("0,1,2") turns on managed
// scheduling over exactly those cards; empty turns managed scheduling OFF, in
// which case GPU workloads fall back to Docker's default count-based request.
func ResolveGPUDevices(spec string) ([]int, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	seen := map[int]bool{}
	var out []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid GPU device index %q in AXISML_GPU_DEVICES", part)
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out, nil
}

// gpuAlive reports whether a container in the given Docker state still holds its
// bound GPU. A container that has exited (or is being removed) has released its
// card even though the container record — and its device label — linger.
func gpuAlive(state string) bool {
	switch state {
	case "running", "restarting", "created", "paused":
		return true
	}
	return false
}

// gpuDockerClient is the subset of the Docker client the GPU occupancy scan
// needs. Extracted as an interface so the scan is testable without a daemon;
// *client.Client satisfies it.
type gpuDockerClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
}

// busyGPUDevices scans all alive managed containers and returns the set of
// occupied device indices plus a count of GPUs held by legacy containers that
// carry no device label (created before this feature); those are subtracted
// from capacity conservatively because their physical cards are unknown.
func busyGPUDevices(ctx context.Context, cli gpuDockerClient) (map[int]struct{}, int, error) {
	f := filters.NewArgs()
	f.Add("label", LabelManaged+"=true")
	conts, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, 0, err
	}
	busy := map[int]struct{}{}
	untracked := 0
	for _, c := range conts {
		if !gpuAlive(c.State) {
			continue
		}
		if devs := c.Labels[LabelGPUDevices]; devs != "" {
			for _, d := range parseDeviceList(devs) {
				busy[d] = struct{}{}
			}
			continue
		}
		n, err := legacyGPUCount(ctx, cli, c.ID)
		if err != nil {
			return nil, 0, err
		}
		untracked += n
	}
	return busy, untracked, nil
}

// legacyGPUCount inspects a labelless managed container and returns how many
// GPUs its DeviceRequests reserve, so an upgrade from the old count-based path
// does not double-book those cards.
func legacyGPUCount(ctx context.Context, cli gpuDockerClient, id string) (int, error) {
	ins, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return 0, err
	}
	if ins.ContainerJSONBase == nil || ins.HostConfig == nil {
		return 0, nil
	}
	n := 0
	for _, dr := range ins.HostConfig.DeviceRequests {
		if dr.Driver != "nvidia" && !capsHaveGPU(dr.Capabilities) {
			continue
		}
		if dr.Count > 0 {
			n += dr.Count
		} else {
			n += len(dr.DeviceIDs)
		}
	}
	return n, nil
}

func capsHaveGPU(caps [][]string) bool {
	for _, group := range caps {
		for _, c := range group {
			if c == "gpu" {
				return true
			}
		}
	}
	return false
}

func parseDeviceList(s string) []int {
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if n, err := strconv.Atoi(part); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// assignGPUsLocked computes the free device set from live Docker state and pins
// concrete indices onto the plans that request GPUs. It must be called with the
// allocator mutex held and only creates containers after a successful return.
// Insufficient capacity surfaces as extensions.ResourceUnavailableError, which
// the reconciler treats as "stay Pending and retry".
func (r *Runtime) assignGPUsLocked(ctx context.Context, cli gpuDockerClient, toCreate []*ContainerPlan) error {
	busy, untracked, err := busyGPUDevices(ctx, cli)
	if err != nil {
		return err
	}
	needs := make([]int, len(toCreate))
	for i, p := range toCreate {
		needs[i] = p.Resources.GPUCount
	}
	assign, err := computeAssignment(r.gpu.schedulable, busy, untracked, needs)
	if err != nil {
		return err
	}
	for i, devs := range assign {
		if len(devs) == 0 {
			continue
		}
		ids := make([]string, len(devs))
		for j, d := range devs {
			ids[j] = strconv.Itoa(d)
		}
		toCreate[i].Resources.GPUDeviceIDs = ids
		toCreate[i].Labels[LabelGPUDevices] = strings.Join(ids, ",")
	}
	return nil
}

// computeAssignment is the pure allocation core (managed mode only): given the
// schedulable card set, the currently-busy indices, the count of GPUs held by
// label-less containers, and the per-plan GPU needs, it returns the concrete
// indices to pin per plan. It returns a ResourceUnavailableError — distinguishing
// "request exceeds capacity" (infeasible) from "no free card right now"
// (transient) — without allocating any card when capacity is insufficient.
func computeAssignment(schedulable []int, busy map[int]struct{}, untracked int, needs []int) ([][]int, error) {
	total := 0
	for _, n := range needs {
		total += n
	}
	if total > len(schedulable) {
		return nil, extensions.NewResourceUnavailable("请求 %d 张 GPU 超过可调度容量 %d", total, len(schedulable))
	}
	var free []int
	for _, d := range schedulable {
		if _, ok := busy[d]; !ok {
			free = append(free, d)
		}
	}
	if untracked > 0 {
		if untracked >= len(free) {
			free = nil
		} else {
			free = free[untracked:]
		}
	}
	if total > len(free) {
		return nil, extensions.NewResourceUnavailable("等待可用 GPU（需 %d，空闲 %d）", total, len(free))
	}
	out := make([][]int, len(needs))
	idx := 0
	for i, n := range needs {
		if n == 0 {
			continue
		}
		devs := make([]int, n)
		copy(devs, free[idx:idx+n])
		idx += n
		out[i] = devs
	}
	return out, nil
}

// createPlans pulls images (outside the allocation lock), atomically reserves
// GPUs for any plans that need them, then creates and starts the containers.
// When a card is unavailable it returns without creating anything, so a
// multi-replica workload never half-occupies the host.
func (r *Runtime) createPlans(ctx context.Context, kind, namespace, name string, toCreate []*ContainerPlan) error {
	for _, p := range toCreate {
		if err := r.pullImage(ctx, p.Image); err != nil {
			r.log.Info("image pull failed, trying local", "image", p.Image, "err", err.Error())
		}
	}
	// Managed GPU scheduling only kicks in when AXISML_GPU_DEVICES names a card
	// set. Otherwise GPU plans keep their count-based request and Docker's NVIDIA
	// runtime picks the cards (toDocker falls back to DeviceRequest.Count).
	need := 0
	for _, p := range toCreate {
		need += p.Resources.GPUCount
	}
	if need > 0 && len(r.gpu.schedulable) > 0 {
		r.gpu.mu.Lock()
		defer r.gpu.mu.Unlock()
		if err := r.assignGPUsLocked(ctx, r.cli, toCreate); err != nil {
			return err
		}
	}
	for _, p := range toCreate {
		if _, err := r.createAndStart(ctx, p); err != nil {
			r.events.record(kind, namespace, name, "", "ApplyFailed", err.Error())
			return err
		}
	}
	return nil
}
