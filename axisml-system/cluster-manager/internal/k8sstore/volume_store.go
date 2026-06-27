package k8sstore

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/axisml/axisml/components/cluster-manager/pkg/extensions"
)

// VolumeStore backs extensions.VolumeStore with a controller-runtime client,
// materialising each Volume as a namespace-scoped PersistentVolumeClaim.
type VolumeStore struct {
	c client.Client
}

var _ extensions.VolumeStore = (*VolumeStore)(nil)

// NewVolumeStore builds a VolumeStore.
func NewVolumeStore(c client.Client) *VolumeStore { return &VolumeStore{c: c} }

// managedByLabel/Value mark PVCs cluster-manager owns, distinguishing them from
// PVCs created by other actors. The volume's purpose (workspace, dataset, …) is
// the caller's concern and is deliberately not encoded here.
const (
	managedByLabel = "app.kubernetes.io/managed-by"
	managedByValue = "axisml-cluster-manager"
)

// Ensure creates the backing PVC. Idempotent: AlreadyExists is treated as
// success so a caller retry doesn't fail the Create. Size must be a valid
// Kubernetes Quantity (the handler pre-validates it for the 400 path).
func (s *VolumeStore) Ensure(ctx context.Context, v extensions.Volume) error {
	q, err := resource.ParseQuantity(v.Size)
	if err != nil {
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v.Name,
			Namespace: v.Namespace,
			Labels:    map[string]string{managedByLabel: managedByValue},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: q},
			},
		},
	}
	if v.StorageClass != "" {
		sc := v.StorageClass
		pvc.Spec.StorageClassName = &sc
	}
	if err := s.c.Create(ctx, pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Delete removes the backing PVC. A missing PVC is not an error.
func (s *VolumeStore) Delete(ctx context.Context, namespace, name string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	if err := s.c.Delete(ctx, pvc); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
