package kubeinventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodRequestsUsesRegularSumAndInitMax(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("1Gi")}}},
			{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")}}},
		},
		InitContainers: []corev1.Container{{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("512Mi"),
		}}}},
		Overhead: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")},
	}}
	requests := podRequests(pod)
	assert.Equal(t, int64(2000), requests.Cpu().MilliValue())
	wantMemory := resource.MustParse("1152Mi")
	assert.Equal(t, wantMemory.Value(), requests.Memory().Value())
}

func TestNodeReady(t *testing.T) {
	assert.False(t, nodeReady(&corev1.Node{}))
	assert.True(t, nodeReady(&corev1.Node{Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{
		Type: corev1.NodeReady, Status: corev1.ConditionTrue,
	}}}}))
}

func TestSnapshotVersionTracksObjectVersions(t *testing.T) {
	base := snapshotVersion(
		corev1.NodeList{Items: []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "node", ResourceVersion: "1"}}}},
		corev1.PodList{Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "pod", ResourceVersion: "2"}}}},
	)
	changed := snapshotVersion(
		corev1.NodeList{Items: []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "node", ResourceVersion: "1"}}}},
		corev1.PodList{Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "pod", ResourceVersion: "3"}}}},
	)
	assert.NotEmpty(t, base)
	assert.NotEqual(t, base, changed)
}
