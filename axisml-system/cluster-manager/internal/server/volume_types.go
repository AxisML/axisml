package server

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

// descriptionAnnotation carries the volume's free-text description on the PVC.
const descriptionAnnotation = "resource.axisml.io/description"

// Volume mirrors the OpenAPI Volume schema — the durable data volume materialised
// by cluster-manager (a PersistentVolumeClaim in Kubernetes, a managed Docker
// volume in Lite). Identified by the (namespace, name) tuple; no CR, no id.
// Status is read-only and populated on get/list from the live PVC and pod scan.
type Volume struct {
	Namespace    string            `json:"namespace" desc:"Physical Kubernetes namespace holding the volume."`
	Name         string            `json:"name" desc:"Volume (PersistentVolumeClaim) name."`
	Size         string            `json:"size,omitempty" desc:"Requested storage size as a Kubernetes quantity (e.g. 50Gi)."`
	StorageClass string            `json:"storageClass,omitempty" desc:"StorageClass backing the volume; cluster default when empty. Immutable after creation."`
	AccessModes  []string          `json:"accessModes,omitempty" desc:"Access modes (ReadWriteOnce/ReadWriteMany/ReadOnlyMany). Immutable after creation."`
	Description  string            `json:"description,omitempty" desc:"Free-text description of the volume."`
	Labels       map[string]string `json:"labels,omitempty" desc:"User-defined labels on the volume."`
	Status       *VolumeStatus     `json:"status,omitempty" desc:"Live status read from the PVC and pod scan; populated on get/list."`
	CreatedAt    time.Time         `json:"createdAt,omitempty" desc:"Volume creation timestamp (RFC3339)."`
}

// VolumeStatus is the live, read-only status of a volume.
type VolumeStatus struct {
	Phase         string        `json:"phase,omitempty" desc:"PVC phase: Pending, Bound, or Lost."`
	BoundCapacity string        `json:"boundCapacity,omitempty" desc:"Actually bound capacity once the volume is Bound."`
	UsedBytes     int64         `json:"usedBytes,omitempty" desc:"Best-effort used bytes from the monitoring stack; omitted when unavailable."`
	Mounts        []VolumeMount `json:"mounts,omitempty" desc:"Workloads currently mounting this volume (populated on detail get)."`
}

// VolumeMount is one workload currently mounting a volume.
type VolumeMount struct {
	Workload  string `json:"workload" desc:"Controlling workload (or pod) name."`
	Kind      string `json:"kind,omitempty" desc:"Kubernetes controller kind (Deployment/StatefulSet/Job/Pod)."`
	MountPath string `json:"mountPath,omitempty" desc:"Mount path inside the pod."`
	Running   bool   `json:"running" desc:"Whether the mounting pod is currently running."`
}

// VolumeList is the LIST response.
type VolumeList struct {
	Items []Volume `json:"items" desc:"Page of volumes."`
	Count int      `json:"count" desc:"Number of volumes in this page."`
}

// StorageClass is a cluster-level storage backend offered for new volumes.
type StorageClass struct {
	Name                 string `json:"name" desc:"StorageClass name."`
	Provisioner          string `json:"provisioner,omitempty" desc:"Provisioner backing the class."`
	Default              bool   `json:"default" desc:"Whether this is the cluster default StorageClass."`
	AllowVolumeExpansion bool   `json:"allowVolumeExpansion" desc:"Whether volumes on this class can be expanded."`
}

// StorageClassList is the LIST response.
type StorageClassList struct {
	Items []StorageClass `json:"items" desc:"Available storage classes."`
	Count int            `json:"count" desc:"Number of storage classes."`
}

// StorageClassesToAPI converts the extension storage classes into API form.
func StorageClassesToAPI(in []extensions.StorageClass) []StorageClass {
	out := make([]StorageClass, 0, len(in))
	for _, s := range in {
		out = append(out, StorageClass{
			Name:                 s.Name,
			Provisioner:          s.Provisioner,
			Default:              s.Default,
			AllowVolumeExpansion: s.AllowVolumeExpansion,
		})
	}
	return out
}

// CreateVolumeRequest is the body for POST /api/v1/volumes. The caller supplies
// the deterministic name and the physical namespace; cluster-manager
// materialises the backing volume.
type CreateVolumeRequest struct {
	Namespace    string            `json:"namespace" desc:"Physical Kubernetes namespace to materialise the volume in."`
	Name         string            `json:"name" desc:"Deterministic volume name supplied by the caller."`
	Size         string            `json:"size" desc:"Requested storage size as a Kubernetes quantity (e.g. 50Gi)."`
	StorageClass string            `json:"storageClass,omitempty" desc:"StorageClass to back the volume; cluster default when empty."`
	AccessModes  []string          `json:"accessModes,omitempty" desc:"Access modes; defaults to [ReadWriteOnce] when empty."`
	Description  string            `json:"description,omitempty" desc:"Free-text description of the volume."`
	Labels       map[string]string `json:"labels,omitempty" desc:"User-defined labels to set on the volume."`
}

// PatchVolumeRequest is the body for PATCH /api/v1/volumes/{namespace}/{name}.
// Size is expand-only; storageClass and accessModes are immutable.
type PatchVolumeRequest struct {
	Size        *string           `json:"size,omitempty" desc:"New storage size; expand-only (must be >= current)."`
	Description *string           `json:"description,omitempty" desc:"Replacement free-text description."`
	Labels      map[string]string `json:"labels,omitempty" desc:"Replacement user-defined label set."`
}

// VolumePatch converts the request into the extensions patch form.
func (p PatchVolumeRequest) VolumePatch() extensions.VolumePatch {
	return extensions.VolumePatch{Size: p.Size, Description: p.Description, Labels: p.Labels}
}

// APIToPVC builds the PersistentVolumeClaim the VolumeManager materialises from
// the create request. Size must be a valid Kubernetes Quantity; an invalid value
// is returned as an error the handler maps to 400. Ownership labels are the
// store's concern; access modes default there when empty.
func APIToPVC(req CreateVolumeRequest) (*corev1.PersistentVolumeClaim, error) {
	q, err := resource.ParseQuantity(req.Size)
	if err != nil {
		return nil, err
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: q},
			},
		},
	}
	if req.StorageClass != "" {
		sc := req.StorageClass
		pvc.Spec.StorageClassName = &sc
	}
	for _, m := range req.AccessModes {
		pvc.Spec.AccessModes = append(pvc.Spec.AccessModes, corev1.PersistentVolumeAccessMode(m))
	}
	if len(req.Labels) > 0 {
		pvc.Labels = map[string]string{}
		for k, v := range req.Labels {
			pvc.Labels[k] = v
		}
	}
	if req.Description != "" {
		pvc.Annotations = map[string]string{descriptionAnnotation: req.Description}
	}
	return pvc, nil
}

// VolumeToAPI projects the materialised PersistentVolumeClaim into the API
// response, including the live status read from the PVC. Mounts are attached
// separately (a pod scan) by the handler for the detail view.
func VolumeToAPI(pvc *corev1.PersistentVolumeClaim) Volume {
	v := Volume{
		Namespace:   pvc.Namespace,
		Name:        pvc.Name,
		Description: pvc.Annotations[descriptionAnnotation],
		Labels:      userLabels(pvc.Labels),
		CreatedAt:   pvc.CreationTimestamp.Time,
	}
	if q, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		v.Size = q.String()
	}
	if pvc.Spec.StorageClassName != nil {
		v.StorageClass = *pvc.Spec.StorageClassName
	}
	for _, m := range pvc.Spec.AccessModes {
		v.AccessModes = append(v.AccessModes, string(m))
	}
	v.Status = &VolumeStatus{Phase: string(pvc.Status.Phase)}
	if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		v.Status.BoundCapacity = q.String()
	}
	return v
}

// MountsToAPI converts extension mounts into the API form.
func MountsToAPI(mounts []extensions.VolumeMount) []VolumeMount {
	if len(mounts) == 0 {
		return nil
	}
	out := make([]VolumeMount, 0, len(mounts))
	for _, m := range mounts {
		out = append(out, VolumeMount{Workload: m.Workload, Kind: m.Kind, MountPath: m.MountPath, Running: m.Running})
	}
	return out
}

// userLabels returns the caller-facing labels, hiding cluster-manager's own
// ownership label so a round-trip create→get→patch doesn't echo internals.
func userLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range labels {
		if k == "app.kubernetes.io/managed-by" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
