package server

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Volume mirrors the OpenAPI Volume schema — the durable volume materialised by
// cluster-manager (a PersistentVolumeClaim in Kubernetes, a managed Docker
// volume in Lite). Identified by the (namespace, name) tuple; no CR, no id.
type Volume struct {
	Namespace    string `json:"namespace" desc:"Physical Kubernetes namespace holding the volume."`
	Name         string `json:"name" desc:"Volume (PersistentVolumeClaim) name."`
	Size         string `json:"size,omitempty" desc:"Requested storage size as a Kubernetes quantity (e.g. 50Gi)."`
	StorageClass string `json:"storageClass,omitempty" desc:"StorageClass backing the volume; cluster default when empty."`
}

// CreateVolumeRequest is the body for POST /api/v1/volumes. The caller supplies
// the deterministic name (e.g. a workspace's axisml-ws-<svc>-data) and the
// physical namespace; cluster-manager materialises the backing volume.
type CreateVolumeRequest struct {
	Namespace    string `json:"namespace" desc:"Physical Kubernetes namespace to materialise the volume in."`
	Name         string `json:"name" desc:"Deterministic volume name supplied by the caller."`
	Size         string `json:"size" desc:"Requested storage size as a Kubernetes quantity (e.g. 50Gi)."`
	StorageClass string `json:"storageClass,omitempty" desc:"StorageClass to back the volume; cluster default when empty."`
}

// APIToPVC builds the PersistentVolumeClaim the VolumeManager materialises from
// the create request. Size must be a valid Kubernetes Quantity; an invalid value
// is returned as an error the handler maps to 400. Ownership labels and access
// modes are the store's concern.
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
	return pvc, nil
}

// VolumeToAPI projects the materialised PersistentVolumeClaim into the API
// response, reading back only the (namespace, name, size, storageClass) the
// caller declared.
func VolumeToAPI(pvc *corev1.PersistentVolumeClaim) Volume {
	v := Volume{Namespace: pvc.Namespace, Name: pvc.Name}
	if q, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		v.Size = q.String()
	}
	if pvc.Spec.StorageClassName != nil {
		v.StorageClass = *pvc.Spec.StorageClassName
	}
	return v
}
