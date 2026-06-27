package docker

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	cmext "github.com/axisml/axisml/components/cluster-manager/pkg/extensions"
)

// Runtime satisfies the cluster-manager VolumeManager: a durable volume is a
// managed Docker named volume. It backs cluster-manager's Volume REST (which
// Platform calls to pre-provision workspace volumes); compute then references
// the volume by claim name in the workload's pod template (design §3.4, §6.4).
// Only the PVC's ObjectMeta (Namespace/Name) is read; the spec's size/storageClass
// are ignored — a single-host Docker volume has no class and grows on demand.
var _ cmext.VolumeManager = (*Runtime)(nil)

// Ensure provisions (or confirms) the claim's backing Docker volume.
func (r *Runtime) Ensure(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	labels := r.baseLabels(KindService, pvc.Namespace, pvc.Name)
	labels[LabelResourceKind] = "volume"
	return r.ensureVolume(ctx, r.pvcVolumeName(pvc.Namespace, pvc.Name), labels)
}

// Delete removes the claim's backing Docker volume. A missing volume is not an
// error.
func (r *Runtime) Delete(ctx context.Context, key types.NamespacedName) error {
	return r.removeVolume(ctx, r.pvcVolumeName(key.Namespace, key.Name))
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
