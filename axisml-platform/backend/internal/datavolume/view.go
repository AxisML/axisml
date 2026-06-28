package datavolume

import (
	"github.com/axisml/axisml/components/platform/internal/clients/clustermanager"
	"github.com/axisml/axisml/components/platform/internal/server"
)

// toDataVolume bridges the cluster-manager wire Volume into the Platform DTO,
// dereferencing the generated pointers and dropping the physical namespace
// (tenant scope is implicit in the request).
func toDataVolume(v *clustermanager.Volume) server.DataVolume {
	out := server.DataVolume{Name: v.Name}
	if v.Size != nil {
		out.Size = *v.Size
	}
	if v.StorageClass != nil {
		out.StorageClass = *v.StorageClass
	}
	if v.AccessModes != nil {
		out.AccessModes = *v.AccessModes
	}
	if v.Description != nil {
		out.Description = *v.Description
	}
	if v.Labels != nil {
		out.Labels = server.StringMap(*v.Labels)
	}
	if v.CreatedAt != nil {
		out.CreatedAt = *v.CreatedAt
	}
	if v.Status != nil {
		st := &server.DataVolumeStatus{}
		if v.Status.Phase != nil {
			st.Phase = *v.Status.Phase
		}
		if v.Status.BoundCapacity != nil {
			st.BoundCapacity = *v.Status.BoundCapacity
		}
		if v.Status.UsedBytes != nil {
			st.UsedBytes = *v.Status.UsedBytes
		}
		if v.Status.Mounts != nil {
			for _, m := range *v.Status.Mounts {
				dm := server.DataVolumeMount{Workload: m.Workload, Running: m.Running}
				if m.Kind != nil {
					dm.Kind = *m.Kind
				}
				if m.MountPath != nil {
					dm.MountPath = *m.MountPath
				}
				st.Mounts = append(st.Mounts, dm)
			}
		}
		out.Status = st
	}
	return out
}

// createBody builds the cluster-manager create payload from the request and the
// resolved physical namespace.
func createBody(namespace string, req server.DataVolumeCreateRequest) clustermanager.VolumeCreate {
	body := clustermanager.VolumeCreate{
		Namespace: &namespace,
		Name:      &req.Name,
		Size:      &req.Size,
	}
	if req.StorageClass != "" {
		body.StorageClass = &req.StorageClass
	}
	if len(req.AccessModes) > 0 {
		modes := req.AccessModes
		body.AccessModes = &modes
	}
	if req.Description != "" {
		body.Description = &req.Description
	}
	if len(req.Labels) > 0 {
		m := map[string]string(req.Labels)
		body.Labels = &m
	}
	return body
}

// toStorageClass bridges the cluster-manager wire StorageClass into the DTO.
func toStorageClass(s *clustermanager.StorageClass) server.StorageClass {
	out := server.StorageClass{
		Name:                 s.Name,
		Default:              s.Default,
		AllowVolumeExpansion: s.AllowVolumeExpansion,
	}
	if s.Provisioner != nil {
		out.Provisioner = *s.Provisioner
	}
	return out
}

// patchBody builds the cluster-manager patch payload.
func patchBody(req server.DataVolumePatchRequest) clustermanager.VolumePatch {
	body := clustermanager.VolumePatch{Size: req.Size, Description: req.Description}
	if req.Labels != nil {
		m := map[string]string(req.Labels)
		body.Labels = &m
	}
	return body
}
