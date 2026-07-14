package standalone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectedPodUsesMainContainerAndRoleLabel(t *testing.T) {
	r := testRuntime()
	pod := r.projectPod("team-a", &instState{
		name:    "hello-world-worker-k7m2q",
		image:   "busybox",
		tenant:  "team-a",
		role:    "worker",
		replica: "0",
	})
	assert.Equal(t, "hello-world-worker-k7m2q", pod.Name)
	assert.Equal(t, "team-a", pod.Labels[LabelTenant])
	assert.Equal(t, "worker", pod.Labels[LabelRole])
	assert.Equal(t, "main", pod.Spec.Containers[0].Name)
	assert.Equal(t, "main", pod.Status.ContainerStatuses[0].Name)
}
