package docker

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// Snapshot models the Docker host as one schedulable virtual node. Only alive
// containers consume capacity; stopped records are retained by Docker for
// observability but do not hold CPU, memory, or GPUs.
func (r *Runtime) Snapshot(ctx context.Context) (extensions.ResourceSnapshot, error) {
	info, err := r.cli.Info(ctx)
	if err != nil {
		return extensions.ResourceSnapshot{}, err
	}
	nodeName := info.Name
	if nodeName == "" {
		nodeName = "standalone"
	}
	allocatable := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(info.NCPU)*1000, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(info.MemTotal, resource.BinarySI),
	}
	if count := len(r.gpu.schedulable); count > 0 {
		allocatable[corev1.ResourceName("nvidia.com/gpu")] = *resource.NewQuantity(int64(count), resource.DecimalSI)
	}
	subtractReserved(allocatable, r.cfg.Reserved)

	containers, err := r.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return extensions.ResourceSnapshot{}, err
	}
	snapshot := extensions.ResourceSnapshot{ObservedAt: time.Now().UTC(), SourceVersion: dockerSnapshotVersion(info.ServerVersion, info.ID, containers), Nodes: []extensions.ResourceNode{{
		Name: nodeName, Labels: map[string]string{corev1.LabelHostname: nodeName}, Allocatable: allocatable,
	}}}
	for _, summary := range containers {
		if !gpuAlive(summary.State) {
			continue
		}
		inspect, err := r.cli.ContainerInspect(ctx, summary.ID)
		if err != nil {
			return extensions.ResourceSnapshot{}, err
		}
		requests := corev1.ResourceList{}
		if inspect.HostConfig != nil {
			if inspect.HostConfig.NanoCPUs > 0 {
				requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(inspect.HostConfig.NanoCPUs/1_000_000, resource.DecimalSI)
			}
			if inspect.HostConfig.Memory > 0 {
				requests[corev1.ResourceMemory] = *resource.NewQuantity(inspect.HostConfig.Memory, resource.BinarySI)
			}
			if gpuCount := dockerGPUCount(inspect.HostConfig.DeviceRequests, summary.Labels[LabelGPUDevices], len(r.gpu.schedulable)); gpuCount > 0 {
				requests[corev1.ResourceName("nvidia.com/gpu")] = *resource.NewQuantity(int64(gpuCount), resource.DecimalSI)
			}
		}
		workloadID := summary.Labels[mlrunv1alpha1.LabelRunID]
		if workloadID == "" {
			workloadID = summary.Labels[mlservicev1alpha1.LabelServiceID]
		}
		snapshot.Allocations = append(snapshot.Allocations, extensions.ResourceAllocation{
			NodeName: nodeName, WorkloadID: workloadID, Role: summary.Labels[LabelRole],
			Replica: summary.Labels[LabelReplicaIndex], Requests: requests,
		})
	}
	return snapshot, nil
}

func dockerSnapshotVersion(serverVersion, daemonID string, containers []container.Summary) string {
	parts := make([]string, 0, len(containers)+1)
	parts = append(parts, serverVersion+"/"+daemonID)
	for _, summary := range containers {
		parts = append(parts, summary.ID+"/"+summary.State+"/"+summary.Status)
	}
	sort.Strings(parts)
	return "docker:" + shortHash(strings.Join(parts, "\x00"))
}

func dockerGPUCount(requests []container.DeviceRequest, assigned string, managedCapacity int) int {
	if assigned != "" {
		return len(parseDeviceList(assigned))
	}
	total := 0
	for _, request := range requests {
		if request.Driver != "nvidia" && !capsHaveGPU(request.Capabilities) {
			continue
		}
		if request.Count > 0 {
			total += request.Count
		} else if request.Count < 0 {
			total += managedCapacity
		} else {
			total += len(request.DeviceIDs)
		}
	}
	return total
}

func subtractReserved(allocatable, reserved corev1.ResourceList) {
	for name, quantity := range reserved {
		current := allocatable[name]
		current.Sub(quantity)
		if current.Sign() < 0 {
			current.Set(0)
		}
		allocatable[name] = current
	}
}
