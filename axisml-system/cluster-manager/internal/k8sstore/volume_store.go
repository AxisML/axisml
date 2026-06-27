package k8sstore

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/axisml/axisml/components/cluster-manager/pkg/extensions"
)

// VolumeStore backs extensions.VolumeManager with a controller-runtime client,
// materialising each Volume as a namespace-scoped PersistentVolumeClaim.
type VolumeStore struct {
	c client.Client
}

var _ extensions.VolumeManager = (*VolumeStore)(nil)

// NewVolumeStore builds a VolumeStore.
func NewVolumeStore(c client.Client) *VolumeStore { return &VolumeStore{c: c} }

// managedByLabel/Value mark PVCs cluster-manager owns, distinguishing them from
// PVCs created by other actors. The volume's purpose (workspace, dataset, …) is
// the caller's concern and is deliberately not encoded here.
const (
	managedByLabel = "app.kubernetes.io/managed-by"
	managedByValue = "axisml-cluster-manager"
)

// Ensure creates the backing PVC. It stamps the cluster-manager ownership label
// and defaults the access mode (K8s specifics the caller need not supply), then
// Creates. Idempotent: AlreadyExists is treated as success so a caller retry
// doesn't fail the Create.
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
