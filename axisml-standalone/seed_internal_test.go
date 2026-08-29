package standalone

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

type recordingVolumeEnsurer struct {
	claims []*corev1.PersistentVolumeClaim
}

func (r *recordingVolumeEnsurer) Ensure(_ context.Context, pvc *corev1.PersistentVolumeClaim) error {
	r.claims = append(r.claims, pvc.DeepCopy())
	return nil
}

func TestMaterializeTenantVolumesUsesLogicalTenantScope(t *testing.T) {
	rt := &recordingVolumeEnsurer{}
	tenant := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "team-a"},
		Spec: tenantv1alpha1.TenantSpec{
			// The physical Kubernetes namespace is metadata in standalone. Docker
			// volume identity follows the logical tenant scope used by workloads.
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "shared-k8s-namespace"},
			InitResources: tenantv1alpha1.InitResources{
				Volumes: []tenantv1alpha1.VolumeSpec{{Name: "dataset"}},
			},
		},
	}

	require.NoError(t, materializeTenantVolumes(context.Background(), rt, tenant))
	require.Len(t, rt.claims, 1)
	assert.Equal(t, "team-a", rt.claims[0].Namespace)
	assert.Equal(t, "dataset", rt.claims[0].Name)
}

func TestMaterializeTenantVolumesRejectsUnavailableResources(t *testing.T) {
	rt := &recordingVolumeEnsurer{}
	tenant := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "team-a"},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: "team-a"},
			InitResources: tenantv1alpha1.InitResources{
				Secrets: []tenantv1alpha1.SecretSpec{{Name: "credentials"}},
			},
		},
	}

	err := materializeTenantVolumes(context.Background(), rt, tenant)
	assert.True(t, apierrors.IsInvalid(err), "want Invalid, got %v", err)
	assert.Empty(t, rt.claims)

	tenant.Spec.InitResources = tenantv1alpha1.InitResources{
		Volumes: []tenantv1alpha1.VolumeSpec{{Name: "dataset", HostPath: "/srv/data"}},
	}
	err = materializeTenantVolumes(context.Background(), rt, tenant)
	assert.True(t, apierrors.IsInvalid(err), "want Invalid, got %v", err)
	assert.Empty(t, rt.claims)
}
