package core

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

// volumeDescriptionAnnotation carries a data volume's description; the
// Standalone Runtime mirrors it onto the Docker volume for read-back. Kept in
// sync with the runtime's own constant.
const volumeDescriptionAnnotation = "resource.axisml.io/description"

// volumeEnsurer is the subset of the cluster-manager VolumeManager the seed
// needs; the Standalone Runtime satisfies it.
type volumeEnsurer interface {
	Ensure(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error
}

// seedTenantVolumes ensures each predefined managed data volume declared on the
// tenant (spec.initResources.volumes[]) exists before any workload reconcile
// mounts it — the Lite equivalent of the tenant-operator's volume subreconciler.
// Ensure is idempotent and content-preserving (it returns an existing volume and
// never wipes it), so running this on every startup is safe. Failures are logged
// and skipped, mirroring EnsureNetwork, rather than blocking boot. hostPath
// volumes are skipped: they bind-mount a host directory (there is no Docker
// volume to create — the registry in tenantHostPathVolumes drives the mount).
func seedTenantVolumes(ctx context.Context, rt volumeEnsurer, t *tenantv1alpha1.Tenant, log logr.Logger) {
	ns := t.Spec.Namespace.Name
	for _, v := range t.Spec.InitResources.Volumes {
		if v.HostPath != "" {
			continue
		}
		if err := rt.Ensure(ctx, buildSeedPVC(ns, v)); err != nil {
			log.Error(err, "ensure predefined tenant volume (continuing)", "volume", v.Name)
		}
	}
}

// tenantHostPathVolumes builds the name→host-path registry the runtime consults
// to bind-mount predefined hostPath volumes (keyed by the claim name a workload
// mounts). Nil when the tenant declares none.
func tenantHostPathVolumes(t *tenantv1alpha1.Tenant) map[string]string {
	var out map[string]string
	for _, v := range t.Spec.InitResources.Volumes {
		if v.HostPath == "" {
			continue
		}
		if out == nil {
			out = map[string]string{}
		}
		out[v.Name] = v.HostPath
	}
	return out
}

// buildSeedPVC renders a VolumeSpec into the PVC shape the Runtime's Ensure
// consumes. The single-host Docker runtime keys the volume on (namespace, name)
// and ignores size/class/accessModes, but they are populated so the same
// builder holds against a real PVC store.
func buildSeedPVC(ns string, v tenantv1alpha1.VolumeSpec) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: v.Name},
	}
	if v.Description != "" {
		pvc.Annotations = map[string]string{volumeDescriptionAnnotation: v.Description}
	}
	if len(v.AccessModes) > 0 {
		pvc.Spec.AccessModes = append([]corev1.PersistentVolumeAccessMode(nil), v.AccessModes...)
	}
	if v.StorageClass != "" {
		sc := v.StorageClass
		pvc.Spec.StorageClassName = &sc
	}
	if v.Size != "" {
		if q, err := resource.ParseQuantity(v.Size); err == nil {
			pvc.Spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: q}
		}
	}
	return pvc
}
