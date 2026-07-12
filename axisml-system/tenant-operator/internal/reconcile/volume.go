package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

// Catalog-alignment metadata so a tenant's predefined volumes appear in — and
// are managed by — the same data-volume catalog cluster-manager serves: the
// catalog lists PVCs carrying this managed-by label, and mirrors the
// description from this annotation. Kept as literals (rather than importing
// cluster-manager) to keep tenant-operator dependency-light.
const (
	volumeManagedByLabel        = "app.kubernetes.io/managed-by"
	volumeManagedByValue        = "axisml-cluster-manager"
	volumeDescriptionAnnotation = "resource.axisml.io/description"
)

// Volumes reconciles spec.initResources.volumes[]: each declared data volume is
// ensured as a managed PersistentVolumeClaim in the tenant namespace, named
// exactly by VolumeSpec.Name so a workload references it by claim name. Unlike
// the credential subreconcilers this one is deliberately non-destructive — it
// sets no ownerReference (so tenant/namespace teardown never cascade-deletes a
// data volume) and runs no orphan GC (removing a spec entry stops the existence
// guarantee, it never deletes data). Readiness means the PVC exists; binding may
// still be deferred (WaitForFirstConsumer), which must not block the tenant.
//
// srcReader is the uncached APIReader: PVCs are read through it rather than the
// cached client because PVCs are not label-restricted in CacheByObject, so a
// cached Get would spin up a cluster-wide PVC informer and pull every PVC into
// memory. Writes (Create/Patch) go through the cached client's writer directly.
func Volumes(ctx context.Context, c client.Client, srcReader client.Reader, t *axisml.Tenant) ([]axisml.InitResourceItemStatus, error) {
	statuses := make([]axisml.InitResourceItemStatus, 0, len(t.Spec.InitResources.Volumes))
	for _, v := range t.Spec.InitResources.Volumes {
		ready, msg, err := ensureVolumePVC(ctx, c, srcReader, t, v)
		statuses = append(statuses, axisml.InitResourceItemStatus{Name: v.Name, Ready: ready, Message: msg})
		if err != nil {
			return statuses, err
		}
	}
	return statuses, nil
}

func ensureVolumePVC(ctx context.Context, c client.Client, srcReader client.Reader, t *axisml.Tenant, spec axisml.VolumeSpec) (bool, string, error) {
	// Fail closed on hostPath. Validate already rejects hostPath in Standard, but
	// guard here too so a spec that ever reaches this path is refused outright
	// rather than silently materialised as a plain (host-less) PVC.
	if spec.HostPath != "" {
		return false, "hostPath volumes are not supported in Standard (multi-tenant)", nil
	}

	ns := t.Spec.Namespace.Name
	labels := ApplyTenantLabels(t, map[string]string{volumeManagedByLabel: volumeManagedByValue})

	existing := &corev1.PersistentVolumeClaim{}
	getErr := srcReader.Get(ctx, types.NamespacedName{Namespace: ns, Name: spec.Name}, existing)
	switch {
	case apierrors.IsNotFound(getErr):
		pvc, err := buildVolumePVC(ns, spec, labels)
		if err != nil {
			// A malformed size is a config error, not a transient failure: report
			// it as not-ready without erroring so the controller doesn't requeue-loop.
			return false, err.Error(), nil
		}
		if err := c.Create(ctx, pvc); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return true, "", nil
			}
			return false, fmt.Sprintf("create pvc failed: %v", err), err
		}
		return true, phaseMessage(corev1.ClaimPending), nil
	case getErr != nil:
		return false, fmt.Sprintf("get pvc failed: %v", getErr), getErr
	default:
		// A PVC of this name already exists. Only adopt it if it is an AxisML-
		// managed volume (created by this operator or the DataVolumes catalog,
		// both of which stamp the managed-by label). Refuse to relabel a PVC we
		// don't own — silently hijacking a foreign/collision PVC into the catalog
		// would be surprising and is never safe. The conflict surfaces as a
		// persistent not-ready status rather than a silent mutation.
		if !isManagedVolume(existing) {
			return false, fmt.Sprintf("a PersistentVolumeClaim %q already exists and is not managed by AxisML; refusing to adopt it", spec.Name), nil
		}
		// Present and ours: keep catalog labels / description in sync, but never
		// touch the spec — size is expand-only (owned by the catalog) and
		// storageClass / accessModes are immutable. Data is never at risk here.
		if err := syncVolumeMeta(ctx, c, existing, labels, spec.Description); err != nil {
			return false, fmt.Sprintf("sync pvc metadata failed: %v", err), err
		}
		return true, phaseMessage(existing.Status.Phase), nil
	}
}

// isManagedVolume reports whether a PVC is an AxisML-managed data volume — i.e.
// carries the catalog managed-by label this operator and cluster-manager both
// stamp. A PVC lacking it was created by some other actor and must not be
// adopted or relabeled.
func isManagedVolume(pvc *corev1.PersistentVolumeClaim) bool {
	return pvc.Labels[volumeManagedByLabel] == volumeManagedByValue
}

// buildVolumePVC renders a VolumeSpec into a PersistentVolumeClaim. Size is
// required for a valid PVC (validated upstream); accessModes default to RWO.
func buildVolumePVC(ns string, spec axisml.VolumeSpec, labels map[string]string) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: spec.Name, Namespace: ns, Labels: labels},
	}
	if spec.Description != "" {
		pvc.Annotations = map[string]string{volumeDescriptionAnnotation: spec.Description}
	}
	if len(spec.AccessModes) > 0 {
		pvc.Spec.AccessModes = append([]corev1.PersistentVolumeAccessMode(nil), spec.AccessModes...)
	} else {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	if spec.StorageClass != "" {
		sc := spec.StorageClass
		pvc.Spec.StorageClassName = &sc
	}
	if spec.Size != "" {
		q, err := resource.ParseQuantity(spec.Size)
		if err != nil {
			return nil, fmt.Errorf("volume %q: invalid size %q: %v", spec.Name, spec.Size, err)
		}
		pvc.Spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: q}
	}
	return pvc, nil
}

// syncVolumeMeta reconciles only the catalog labels and description annotation
// onto an existing PVC. It never patches the spec, so a volume's data and size
// are untouched.
func syncVolumeMeta(ctx context.Context, c client.Client, pvc *corev1.PersistentVolumeClaim, labels map[string]string, description string) error {
	base := pvc.DeepCopy()
	changed := false
	if pvc.Labels == nil {
		pvc.Labels = map[string]string{}
	}
	for k, v := range labels {
		if pvc.Labels[k] != v {
			pvc.Labels[k] = v
			changed = true
		}
	}
	if description != "" {
		if pvc.Annotations == nil {
			pvc.Annotations = map[string]string{}
		}
		if pvc.Annotations[volumeDescriptionAnnotation] != description {
			pvc.Annotations[volumeDescriptionAnnotation] = description
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return c.Patch(ctx, pvc, client.MergeFrom(base))
}

// phaseMessage surfaces a non-Bound PVC phase as an informational status
// message; a Bound (or freshly created, unreported) volume reports nothing.
func phaseMessage(phase corev1.PersistentVolumeClaimPhase) string {
	if phase == corev1.ClaimBound || phase == "" {
		return ""
	}
	return fmt.Sprintf("volume phase=%s", phase)
}
