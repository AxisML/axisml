package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDockerSnapshotVersionChangesWithContainerState(t *testing.T) {
	running := []container.Summary{{ID: "one", State: "running"}}
	exited := []container.Summary{{ID: "one", State: "exited"}}
	assert.Equal(t, dockerSnapshotVersion("v1", "daemon", running), dockerSnapshotVersion("v1", "daemon", running))
	assert.NotEqual(t, dockerSnapshotVersion("v1", "daemon", running), dockerSnapshotVersion("v1", "daemon", exited))
}

func TestDockerGPUCount(t *testing.T) {
	requests := []container.DeviceRequest{{Driver: "nvidia", Count: 2, Capabilities: [][]string{{"gpu"}}}}
	assert.Equal(t, 2, dockerGPUCount(requests, "", 4))
	assert.Equal(t, 3, dockerGPUCount(requests, "0,2,4", 4), "explicit assignment is authoritative")
	assert.Equal(t, 4, dockerGPUCount([]container.DeviceRequest{{Driver: "nvidia", Count: -1}}, "", 4), "Docker Count=-1 reserves every managed GPU")
}

func TestSubtractReservedClampsAtZero(t *testing.T) {
	capacity := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("2"),
		corev1.ResourceMemory: resource.MustParse("2Gi"),
	}
	subtractReserved(capacity, corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("3Gi"),
	})
	assert.Equal(t, int64(1500), capacity.Cpu().MilliValue())
	assert.Zero(t, capacity.Memory().Value())
}
