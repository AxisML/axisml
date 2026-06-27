package server

import "github.com/axisml/axisml/components/cluster-manager/pkg/extensions"

// Volume mirrors the OpenAPI Volume schema — the durable volume materialised by
// cluster-manager (a PersistentVolumeClaim in Kubernetes, a managed Docker
// volume in Lite). Identified by the (namespace, name) tuple; no CR, no id.
type Volume struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Size         string `json:"size,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
}

// CreateVolumeRequest is the body for POST /api/v1/volumes. The caller supplies
// the deterministic name (e.g. a workspace's axisml-ws-<svc>-data) and the
// physical namespace; cluster-manager materialises the backing volume.
type CreateVolumeRequest struct {
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Size         string `json:"size"`
	StorageClass string `json:"storageClass,omitempty"`
}

// APIToVolume converts the create request into the neutral store value.
func APIToVolume(req CreateVolumeRequest) extensions.Volume {
	return extensions.Volume{
		Namespace:    req.Namespace,
		Name:         req.Name,
		Size:         req.Size,
		StorageClass: req.StorageClass,
	}
}

// VolumeToAPI converts the neutral store value into the API response.
func VolumeToAPI(v extensions.Volume) Volume {
	return Volume{
		Namespace:    v.Namespace,
		Name:         v.Name,
		Size:         v.Size,
		StorageClass: v.StorageClass,
	}
}
