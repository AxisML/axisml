package standalone

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

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
// mounts it — the standalone equivalent of the tenant-operator's volume subreconciler.
// Ensure is idempotent and content-preserving (it returns an existing volume and
// never wipes it), so running this on every startup is safe. Failures are logged
// and skipped, mirroring EnsureNetwork, rather than blocking boot. hostPath
// volumes are skipped: they bind-mount a host directory (there is no Docker
// volume to create — the registry in tenantHostPathVolumes drives the mount).
func seedTenantVolumes(ctx context.Context, rt volumeEnsurer, t *tenantv1alpha1.Tenant, log logr.Logger) {
	for _, v := range t.Spec.InitResources.Volumes {
		if v.HostPath != "" {
			continue
		}
		if err := rt.Ensure(ctx, buildSeedPVC(t.Name, v)); err != nil {
			log.Error(err, "ensure predefined tenant volume (continuing)", "volume", v.Name)
		}
	}
}

// materializeTenantVolumes is the synchronous standalone equivalent of the
// tenant-operator volume subreconciler for API-created/updated tenants. Only
// managed data volumes are accepted through the REST API. Host bind mounts and
// credential-copy resources remain trusted startup configuration because the
// standalone process cannot safely materialize arbitrary Kubernetes sources.
func materializeTenantVolumes(ctx context.Context, rt volumeEnsurer, t *tenantv1alpha1.Tenant) error {
	if !credentialInitResourcesEmpty(t.Spec.InitResources) {
		return apierrors.NewInvalid(
			tenantv1alpha1.GroupVersion.WithKind("Tenant").GroupKind(),
			t.Name,
			field.ErrorList{field.Forbidden(field.NewPath("spec", "initResources"), "secrets, configMaps and serviceAccounts are unavailable in standalone")},
		)
	}
	if err := validateTenantVolumes(t.Name, t.Spec.InitResources.Volumes, map[string]string{}); err != nil {
		return apierrors.NewInvalid(
			tenantv1alpha1.GroupVersion.WithKind("Tenant").GroupKind(),
			t.Name,
			field.ErrorList{field.Invalid(field.NewPath("spec", "initResources", "volumes"), t.Spec.InitResources.Volumes, err.Error())},
		)
	}
	for _, v := range t.Spec.InitResources.Volumes {
		if v.HostPath != "" {
			return apierrors.NewInvalid(
				tenantv1alpha1.GroupVersion.WithKind("Tenant").GroupKind(),
				t.Name,
				field.ErrorList{field.Forbidden(field.NewPath("spec", "initResources", "volumes"), "hostPath volumes may only be declared in trusted startup config")},
			)
		}
		if err := rt.Ensure(ctx, buildSeedPVC(t.Name, v)); err != nil {
			return fmt.Errorf("ensure predefined tenant volume %q: %w", v.Name, err)
		}
	}
	return nil
}

// tenantsHostPathVolumes builds the name→host-path registry the runtime consults
// to bind-mount predefined hostPath volumes (keyed by the claim name a workload
// mounts), merged across every tenant. hostPath volume names are validated
// unique across tenants at config load, so the merge never collides. Nil when no
// tenant declares any.
func tenantsHostPathVolumes(tenants []*tenantv1alpha1.Tenant) map[string]string {
	var out map[string]string
	for _, t := range tenants {
		for _, v := range t.Spec.InitResources.Volumes {
			if v.HostPath == "" {
				continue
			}
			if out == nil {
				out = map[string]string{}
			}
			out[v.Name] = v.HostPath
		}
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
