package standalone

import (
	"context"
	"fmt"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dmount "github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	cmext "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

// Runtime satisfies the cluster-manager VolumeManager: a durable data volume is
// a managed Docker named volume. It backs cluster-manager's Volume REST (which
// Platform calls to manage data volumes); compute then references the volume by
// claim name in the workload's pod template (design §3.4). A single-host Docker
// volume has no class and grows on demand, so size / storageClass / accessModes
// are accepted but not enforced, and expand (Patch) is unavailable.
var _ cmext.VolumeManager = (*Runtime)(nil)

// descriptionAnnotation is the PVC annotation carrying the volume's description;
// descriptionLabel mirrors it onto the Docker volume for read-back.
const (
	descriptionAnnotation = "axisml.io/description"
	descriptionLabel      = "io.axisml.description"
)

// pvcResource scopes the NotFound error Get returns so the handler maps it to 404.
var pvcResource = schema.GroupResource{Resource: "persistentvolumeclaims"}

// Ensure provisions (or confirms) the claim's backing Docker volume, mirroring
// the PVC's description onto the volume labels for read-back.
func (r *Runtime) Ensure(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	labels := r.baseLabels(KindService, pvc.Namespace, pvc.Name)
	labels[LabelResourceKind] = "volume"
	if d := pvc.Annotations[descriptionAnnotation]; d != "" {
		labels[descriptionLabel] = d
	}
	return r.ensureVolume(ctx, r.pvcVolumeName(pvc.Namespace, pvc.Name), labels)
}

// Get inspects the claim's backing Docker volume and projects it back into a
// PVC. A missing volume yields a Kubernetes NotFound the handler maps to 404.
func (r *Runtime) Get(ctx context.Context, key types.NamespacedName) (*corev1.PersistentVolumeClaim, error) {
	v, err := r.cli.VolumeInspect(ctx, r.pvcVolumeName(key.Namespace, key.Name))
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil, apierrors.NewNotFound(pvcResource, key.Name)
		}
		return nil, err
	}
	return r.volumeToPVC(key.Namespace, key.Name, &v), nil
}

// List returns the managed data volumes (kind=volume) for a namespace. The
// labelSelector is a Kubernetes concept and does not apply to Docker volumes.
func (r *Runtime) List(ctx context.Context, namespace, _ string) ([]corev1.PersistentVolumeClaim, error) {
	f := filters.NewArgs()
	f.Add("label", LabelManaged+"=true")
	f.Add("label", LabelResourceKind+"=volume")
	if namespace != "" {
		f.Add("label", LabelNamespace+"="+namespace)
	}
	resp, err := r.cli.VolumeList(ctx, volume.ListOptions{Filters: f})
	if err != nil {
		return nil, err
	}
	out := make([]corev1.PersistentVolumeClaim, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		ns := v.Labels[LabelNamespace]
		name := v.Labels[LabelName]
		if name == "" {
			continue
		}
		out = append(out, *r.volumeToPVC(ns, name, v))
	}
	return out, nil
}

// Patch is unavailable on a single-host Docker volume: it has no class to expand
// and its labels are immutable after creation.
func (r *Runtime) Patch(_ context.Context, _ types.NamespacedName, _ cmext.VolumePatch) (*corev1.PersistentVolumeClaim, error) {
	return nil, cmext.ErrCapabilityUnavailable
}

// Mounts reports the managed containers currently mounting the claim's volume.
func (r *Runtime) Mounts(ctx context.Context, key types.NamespacedName) ([]cmext.VolumeMount, error) {
	f := filters.NewArgs()
	f.Add("label", LabelManaged+"=true")
	containers, err := r.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, err
	}
	target := r.pvcVolumeName(key.Namespace, key.Name)
	var out []cmext.VolumeMount
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type != dmount.TypeVolume || m.Name != target {
				continue
			}
			out = append(out, cmext.VolumeMount{
				Workload:  c.Labels[LabelName],
				Kind:      c.Labels[LabelResourceKind],
				MountPath: m.Destination,
				Running:   c.State == "running",
			})
			break
		}
	}
	return out, nil
}

// Delete removes the claim's backing Docker volume. A missing volume is not an
// error.
func (r *Runtime) Delete(ctx context.Context, key types.NamespacedName) error {
	return r.removeVolume(ctx, r.pvcVolumeName(key.Namespace, key.Name))
}

// ListStorageClasses returns nothing: a single-host Docker volume has no
// StorageClass concept (volumes grow on demand on the local filesystem).
func (r *Runtime) ListStorageClasses(_ context.Context) ([]cmext.StorageClass, error) {
	return nil, nil
}

// volumeToPVC projects a Docker volume into the PVC shape cluster-manager's
// server layer reads back. Single-host volumes are always considered Bound.
func (r *Runtime) volumeToPVC(namespace, name string, v *volume.Volume) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	if d := v.Labels[descriptionLabel]; d != "" {
		pvc.Annotations = map[string]string{descriptionAnnotation: d}
	}
	if v.CreatedAt != "" {
		if ts, err := time.Parse(time.RFC3339, v.CreatedAt); err == nil {
			pvc.CreationTimestamp = metav1.NewTime(ts)
		}
	}
	return pvc
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
