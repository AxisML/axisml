// Package provider declares the deployment-form-neutral service-provider
// interfaces the Compute module depends on. Each composition root (the
// Kubernetes binary, or Lite's axisml-system) injects concrete providers:
//
//   - ResourceCatalog expands a (pool, unit) name pair into a resource
//     snapshot. Kubernetes reads the ResourcePool CR informer cache; Lite reads
//     a static config catalog.
//   - WorkspaceVolumeProvisioner manages the durable volume backing a
//     kind=workspace MLService. Kubernetes creates a PVC; Lite creates a managed
//     Docker volume via the runtime.
//
// Keeping these as a leaf package (corev1 types only, no internal imports) lets
// both pkg/module and the internal business packages depend on them without an
// import cycle.
package provider

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// Expanded is the resource snapshot a ResourceCatalog returns for a
// (pool, unit) pair, per design §5.4 merge rules.
type Expanded struct {
	NodeSelector map[string]string
	Tolerations  []corev1.Toleration
	Requests     corev1.ResourceList
	Limits       corev1.ResourceList
}

// ResourceCatalog resolves (poolName, unitName) into the expanded snapshot that
// is frozen into the workload spec at create time.
type ResourceCatalog interface {
	Resolve(ctx context.Context, poolName, unitName string) (*Expanded, error)
}

// WorkspaceVolumeProvisioner manages the durable volume that backs a
// kind=workspace MLService. Implementations must be idempotent.
type WorkspaceVolumeProvisioner interface {
	// EnsureWorkspaceVolume provisions (or confirms) the volume backing the
	// named workspace. size is a Kubernetes Quantity string; storageClass may
	// be empty for the cluster default.
	EnsureWorkspaceVolume(ctx context.Context, namespace, name, size, storageClass string) error
	// DeleteWorkspaceVolume removes the workspace's backing volume. A missing
	// volume is not an error.
	DeleteWorkspaceVolume(ctx context.Context, namespace, name string) error
}

// BuildResources converts a unit's request/limit ResourceLists into the K8s
// ResourceRequirements shape expected by Pod templates.
func BuildResources(req, lim corev1.ResourceList) corev1.ResourceRequirements {
	rr := corev1.ResourceRequirements{}
	if len(req) > 0 {
		rr.Requests = req
	}
	if len(lim) > 0 {
		rr.Limits = lim
	}
	return rr
}
