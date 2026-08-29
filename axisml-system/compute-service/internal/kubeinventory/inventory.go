// Package kubeinventory exposes a read-only Kubernetes capacity snapshot for
// Compute admission. It never creates or binds Pods.
package kubeinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

type Inventory struct{ reader client.Reader }

func New(reader client.Reader) *Inventory { return &Inventory{reader: reader} }

var _ extensions.ResourceInventory = (*Inventory)(nil)

func (i *Inventory) Snapshot(ctx context.Context) (extensions.ResourceSnapshot, error) {
	var nodeList corev1.NodeList
	if err := i.reader.List(ctx, &nodeList); err != nil {
		return extensions.ResourceSnapshot{}, err
	}
	var podList corev1.PodList
	if err := i.reader.List(ctx, &podList); err != nil {
		return extensions.ResourceSnapshot{}, err
	}

	snapshot := extensions.ResourceSnapshot{
		ObservedAt:    time.Now().UTC(),
		SourceVersion: snapshotVersion(nodeList, podList),
	}
	for idx := range nodeList.Items {
		node := &nodeList.Items[idx]
		if node.Spec.Unschedulable || !nodeReady(node) {
			continue
		}
		snapshot.Nodes = append(snapshot.Nodes, extensions.ResourceNode{
			Name: node.Name, Labels: node.Labels, Taints: node.Spec.Taints,
			Allocatable: copyResources(node.Status.Allocatable),
		})
	}
	for idx := range podList.Items {
		pod := &podList.Items[idx]
		if pod.Spec.NodeName == "" || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		workloadID := pod.Labels[mlrunv1alpha1.LabelRunID]
		role := pod.Labels[mlrunv1alpha1.LabelRole]
		if workloadID == "" {
			workloadID = pod.Labels[mlservicev1alpha1.LabelServiceID]
			role = pod.Labels[mlservicev1alpha1.LabelRole]
		}
		snapshot.Allocations = append(snapshot.Allocations, extensions.ResourceAllocation{
			NodeName: pod.Spec.NodeName, WorkloadID: workloadID, Role: role, Replica: pod.Name,
			Requests: podRequests(pod),
		})
	}
	return snapshot, nil
}

func snapshotVersion(nodes corev1.NodeList, pods corev1.PodList) string {
	versions := make([]string, 0, len(nodes.Items)+len(pods.Items)+2)
	versions = append(versions, "nodes="+nodes.ResourceVersion, "pods="+pods.ResourceVersion)
	for i := range nodes.Items {
		versions = append(versions, "node/"+nodes.Items[i].Name+"="+nodes.Items[i].ResourceVersion)
	}
	for i := range pods.Items {
		versions = append(versions, "pod/"+pods.Items[i].Namespace+"/"+pods.Items[i].Name+"="+pods.Items[i].ResourceVersion)
	}
	sort.Strings(versions)
	hash := sha256.New()
	for _, version := range versions {
		_, _ = hash.Write([]byte(version))
		_, _ = hash.Write([]byte{0})
	}
	return "kubernetes:" + hex.EncodeToString(hash.Sum(nil)[:8])
}

func nodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func podRequests(pod *corev1.Pod) corev1.ResourceList {
	requests := corev1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		add(requests, container.Resources.Requests)
	}
	// Init containers run sequentially: scheduler reservation is max(sum of
	// regular containers, each init container) per resource.
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			current := requests[name]
			if quantity.Cmp(current) > 0 {
				requests[name] = quantity.DeepCopy()
			}
		}
	}
	add(requests, pod.Spec.Overhead)
	return requests
}

func add(dst, src corev1.ResourceList) {
	for name, quantity := range src {
		current := dst[name]
		current.Add(quantity)
		dst[name] = current
	}
}

func copyResources(in corev1.ResourceList) corev1.ResourceList {
	out := make(corev1.ResourceList, len(in))
	for name, quantity := range in {
		out[name] = quantity.DeepCopy()
	}
	return out
}
