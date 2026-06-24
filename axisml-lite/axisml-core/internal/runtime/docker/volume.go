package docker

import (
	"context"
	"fmt"

	csprovider "github.com/axisml/axisml/components/compute-service/pkg/provider"
)

// Runtime also satisfies the Compute workspace-volume provisioner: the durable
// volume backing a kind=workspace MLService is a managed Docker named volume
// (design §4.2, §6.4).
var _ csprovider.WorkspaceVolumeProvisioner = (*Runtime)(nil)

// EnsureWorkspaceVolume provisions (or confirms) the named workspace's backing
// volume. size/storageClass are accepted for contract parity but a single-host
// Docker volume has no class and grows on demand.
func (r *Runtime) EnsureWorkspaceVolume(ctx context.Context, namespace, name, _ string, _ string) error {
	labels := r.baseLabels(KindService, namespace, name)
	labels[LabelResourceKind] = "workspace"
	return r.ensureVolume(ctx, r.workspaceVolumeName(namespace, name), labels)
}

// DeleteWorkspaceVolume removes the workspace's backing volume. A missing
// volume is not an error.
func (r *Runtime) DeleteWorkspaceVolume(ctx context.Context, namespace, name string) error {
	return r.removeVolume(ctx, r.workspaceVolumeName(namespace, name))
}

func (r *Runtime) workspaceVolumeName(namespace, name string) string {
	raw := fmt.Sprintf("axisml-ws-%s-%s", namespace, name)
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 100 {
		return clean
	}
	return fmt.Sprintf("axisml-ws-%s", shortHash(raw))
}
