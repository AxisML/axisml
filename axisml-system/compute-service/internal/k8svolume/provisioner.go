// Package k8svolume is the Kubernetes implementation of
// extensions.WorkspaceVolumeProvisioner: it backs a kind=workspace MLService with
// a PersistentVolumeClaim. The Lite form provides its own provisioner that
// creates a managed Docker volume instead.
package k8svolume

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/internal/mlservice"
	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
	"github.com/axisml/axisml/components/compute-service/pkg/extensions"
)

// Provisioner manages workspace PVCs via a controller-runtime client.
type Provisioner struct {
	c client.Client
}

var _ extensions.WorkspaceVolumeProvisioner = (*Provisioner)(nil)

// New builds a Provisioner.
func New(c client.Client) *Provisioner { return &Provisioner{c: c} }

// EnsureWorkspaceVolume creates the backing PVC. Idempotent: AlreadyExists is
// treated as success so reconcile retries don't fail the Create.
func (p *Provisioner) EnsureWorkspaceVolume(ctx context.Context, namespace, name, size, storageClass string) error {
	if size == "" {
		return apperrors.New(apperrors.CodeValidation,
			"workspaceStorage.size is required when kind=workspace")
	}
	q, err := resource.ParseQuantity(size)
	if err != nil {
		return apperrors.Newf(apperrors.CodeValidation,
			"workspaceStorage.size %q is not a valid Quantity: %v", size, err)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mlservice.WorkspacePVCName(name),
			Namespace: namespace,
			Labels: map[string]string{
				mlservicev1alpha1.LabelServiceKind: mlservicev1alpha1.ServiceKindWorkspace,
				"axisml.io/workspace":              name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: q},
			},
		},
	}
	if storageClass != "" {
		sc := storageClass
		pvc.Spec.StorageClassName = &sc
	}
	if err := p.c.Create(ctx, pvc); err != nil && !apierrors.IsAlreadyExists(err) {
		return apperrors.Wrap(apperrors.CodeUnavailable, "create workspace pvc", err)
	}
	return nil
}

// DeleteWorkspaceVolume removes the backing PVC. A missing PVC is not an error.
func (p *Provisioner) DeleteWorkspaceVolume(ctx context.Context, namespace, name string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mlservice.WorkspacePVCName(name),
			Namespace: namespace,
		},
	}
	if err := p.c.Delete(ctx, pvc); err != nil && !apierrors.IsNotFound(err) {
		return apperrors.Wrap(apperrors.CodeUnavailable, "delete workspace pvc", err)
	}
	return nil
}
