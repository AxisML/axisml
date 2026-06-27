package docker

import (
	"context"
	"fmt"

	cmext "github.com/axisml/axisml/components/cluster-manager/pkg/extensions"
)

// Runtime satisfies the cluster-manager VolumeStore: a durable volume is a
// managed Docker named volume. It backs cluster-manager's Volume REST (which
// Platform calls to pre-provision workspace volumes); compute then references
// the volume by claim name in the workload's pod template (design §3.4, §6.4).
// size/storageClass are accepted for contract parity but a single-host Docker
// volume has no class and grows on demand.
var _ cmext.VolumeStore = (*Runtime)(nil)

// Ensure provisions (or confirms) the named volume's backing Docker volume.
func (r *Runtime) Ensure(ctx context.Context, v cmext.Volume) error {
	labels := r.baseLabels(KindService, v.Namespace, v.Name)
	labels[LabelResourceKind] = "volume"
	return r.ensureVolume(ctx, r.pvcVolumeName(v.Namespace, v.Name), labels)
}

// Delete removes the named volume's backing Docker volume. A missing volume is
// not an error.
func (r *Runtime) Delete(ctx context.Context, namespace, name string) error {
	return r.removeVolume(ctx, r.pvcVolumeName(namespace, name))
}

// pvcVolumeName derives the Docker volume name for a PVC-backed volume keyed by
// (namespace, claimName). serviceMounts uses the same derivation so the mounted
// volume and the provisioned volume are one and the same.
func (r *Runtime) pvcVolumeName(namespace, claimName string) string {
	raw := fmt.Sprintf("axisml-%s-%s", namespace, claimName)
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 100 {
		return clean
	}
	return fmt.Sprintf("axisml-vol-%s", shortHash(raw))
}
