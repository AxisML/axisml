package k8sstore

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

const defaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

// VolumeStore backs extensions.VolumeManager with a controller-runtime client,
// materialising each Volume as a namespace-scoped PersistentVolumeClaim and
// computing occupancy by scanning pods in the namespace.
type VolumeStore struct {
	c client.Client
}

var _ extensions.VolumeManager = (*VolumeStore)(nil)

// NewVolumeStore builds a VolumeStore.
func NewVolumeStore(c client.Client) *VolumeStore { return &VolumeStore{c: c} }

// managedByLabel/Value mark PVCs cluster-manager owns, distinguishing them from
// PVCs created by other actors. List filters on this so the data-volume catalog
// shows only managed volumes. The volume's purpose is the caller's concern and
// is deliberately not encoded here.
const (
	managedByLabel = "app.kubernetes.io/managed-by"
	managedByValue = "axisml-cluster-manager"

	descriptionAnnotation = "resource.axisml.io/description"
)

// Ensure creates the backing PVC. It stamps the cluster-manager ownership label
// and defaults the access mode (a K8s specific the caller need not supply), then
// Creates. Idempotent: AlreadyExists is treated as success.
func (s *VolumeStore) Ensure(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	pvc = pvc.DeepCopy()
	if pvc.Labels == nil {
		pvc.Labels = map[string]string{}
	}
	pvc.Labels[managedByLabel] = managedByValue
	if len(pvc.Spec.AccessModes) == 0 {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	if err := s.c.Create(ctx, pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Get reads back the backing PVC.
func (s *VolumeStore) Get(ctx context.Context, key types.NamespacedName) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := s.c.Get(ctx, key, pvc); err != nil {
		return nil, err
	}
	return pvc, nil
}

// List returns the managed PVCs in a namespace. The cluster-manager ownership
// label is always required; an extra caller selector is ANDed on top.
func (s *VolumeStore) List(ctx context.Context, namespace, labelSelector string) ([]corev1.PersistentVolumeClaim, error) {
	sel := labels.SelectorFromSet(labels.Set{managedByLabel: managedByValue})
	if labelSelector != "" {
		extra, err := labels.Parse(labelSelector)
		if err != nil {
			return nil, err
		}
		reqs, _ := extra.Requirements()
		sel = sel.Add(reqs...)
	}
	list := &corev1.PersistentVolumeClaimList{}
	opts := []client.ListOption{client.MatchingLabelsSelector{Selector: sel}}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if err := s.c.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// Patch expands the volume and/or updates its description / labels. Size is
// expand-only (the K8s API rejects a shrink); storageClass and accessModes are
// immutable and untouched here.
func (s *VolumeStore) Patch(ctx context.Context, key types.NamespacedName, patch extensions.VolumePatch) (*corev1.PersistentVolumeClaim, error) {
	base := &corev1.PersistentVolumeClaim{}
	if err := s.c.Get(ctx, key, base); err != nil {
		return nil, err
	}
	obj := base.DeepCopy()

	if patch.Size != nil {
		q, err := resource.ParseQuantity(*patch.Size)
		if err != nil {
			return nil, err
		}
		if obj.Spec.Resources.Requests == nil {
			obj.Spec.Resources.Requests = corev1.ResourceList{}
		}
		obj.Spec.Resources.Requests[corev1.ResourceStorage] = q
	}
	if patch.Description != nil {
		if *patch.Description == "" {
			delete(obj.Annotations, descriptionAnnotation)
		} else {
			if obj.Annotations == nil {
				obj.Annotations = map[string]string{}
			}
			obj.Annotations[descriptionAnnotation] = *patch.Description
		}
	}
	if patch.Labels != nil {
		merged := map[string]string{managedByLabel: managedByValue}
		for k, v := range patch.Labels {
			if k == managedByLabel {
				continue
			}
			merged[k] = v
		}
		obj.Labels = merged
	}

	// Optimistic lock so a concurrent edit returns 409 instead of silently
	// clobbering the peer's change; the handler maps it to OptimisticLockConflict.
	if err := s.c.Patch(ctx, obj, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
		return nil, err
	}
	return obj, nil
}

// Mounts scans pods in the volume's namespace and reports those referencing the
// PVC by claim name. Used for delete-time occupancy checks and the detail view.
func (s *VolumeStore) Mounts(ctx context.Context, key types.NamespacedName) ([]extensions.VolumeMount, error) {
	pods := &corev1.PodList{}
	if err := s.c.List(ctx, pods, client.InNamespace(key.Namespace)); err != nil {
		return nil, err
	}
	var out []extensions.VolumeMount
	for i := range pods.Items {
		p := &pods.Items[i]
		volName := ""
		for _, v := range p.Spec.Volumes {
			if v.PersistentVolumeClaim != nil && v.PersistentVolumeClaim.ClaimName == key.Name {
				volName = v.Name
				break
			}
		}
		if volName == "" {
			continue
		}
		out = append(out, extensions.VolumeMount{
			Workload:  workloadName(p),
			Kind:      workloadKind(p),
			MountPath: mountPathFor(p, volName),
			Running:   p.Status.Phase == corev1.PodRunning,
		})
	}
	return out, nil
}

// ListStorageClasses returns the cluster's storage classes for the new-volume
// picker, flagging the default and expansion capability.
func (s *VolumeStore) ListStorageClasses(ctx context.Context) ([]extensions.StorageClass, error) {
	list := &storagev1.StorageClassList{}
	if err := s.c.List(ctx, list); err != nil {
		return nil, err
	}
	out := make([]extensions.StorageClass, 0, len(list.Items))
	for i := range list.Items {
		sc := &list.Items[i]
		item := extensions.StorageClass{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
			Default:     sc.Annotations[defaultStorageClassAnnotation] == "true",
		}
		if sc.AllowVolumeExpansion != nil {
			item.AllowVolumeExpansion = *sc.AllowVolumeExpansion
		}
		out = append(out, item)
	}
	return out, nil
}

// Delete removes the backing PVC. A missing PVC is not an error.
func (s *VolumeStore) Delete(ctx context.Context, key types.NamespacedName) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
	}
	if err := s.c.Delete(ctx, pvc); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// workloadName returns the controlling workload name, falling back to the pod
// name when the pod has no controller.
func workloadName(p *corev1.Pod) string {
	if owner := metav1.GetControllerOf(p); owner != nil {
		return owner.Name
	}
	return p.Name
}

// workloadKind returns the controller kind, or Pod when uncontrolled.
func workloadKind(p *corev1.Pod) string {
	if owner := metav1.GetControllerOf(p); owner != nil {
		return owner.Kind
	}
	return "Pod"
}

// mountPathFor finds the first mount path for the named volume across the pod's
// containers (init and regular).
func mountPathFor(p *corev1.Pod, volName string) string {
	containers := append([]corev1.Container{}, p.Spec.InitContainers...)
	containers = append(containers, p.Spec.Containers...)
	for _, ctr := range containers {
		for _, vm := range ctr.VolumeMounts {
			if vm.Name == volName {
				return vm.MountPath
			}
		}
	}
	return ""
}
