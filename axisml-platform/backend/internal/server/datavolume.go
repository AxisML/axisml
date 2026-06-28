package server

import "time"

// DataVolume is a tenant-scoped, system-admin-managed durable volume (a
// PersistentVolumeClaim materialised by cluster-manager). Addressed by name
// within the active tenant; the physical namespace is resolved server-side.
type DataVolume struct {
	Name         string            `json:"name" desc:"Data volume name, unique within the tenant."`
	Size         string            `json:"size,omitempty" desc:"Requested storage size as a Kubernetes quantity (e.g. 200Gi)."`
	StorageClass string            `json:"storageClass,omitempty" desc:"StorageClass backing the volume; cluster default when empty. Immutable after creation."`
	AccessModes  []string          `json:"accessModes,omitempty" desc:"Access modes (ReadWriteOnce/ReadWriteMany/ReadOnlyMany). Immutable after creation."`
	Description  string            `json:"description,omitempty" desc:"Free-text description of the volume."`
	Labels       StringMap         `json:"labels,omitempty" desc:"User-defined labels on the volume."`
	Status       *DataVolumeStatus `json:"status,omitempty" desc:"Live status read from the backing PVC and pod scan."`
	CreatedAt    time.Time         `json:"createdAt,omitempty" desc:"Volume creation timestamp (RFC3339)."`
}

// DataVolumeStatus is the live, read-only status of a data volume.
type DataVolumeStatus struct {
	Phase         string            `json:"phase,omitempty" desc:"PVC phase: Pending, Bound, or Lost."`
	BoundCapacity string            `json:"boundCapacity,omitempty" desc:"Actually bound capacity once the volume is Bound."`
	UsedBytes     int64             `json:"usedBytes,omitempty" desc:"Best-effort used bytes from the monitoring stack; omitted when unavailable."`
	Mounts        []DataVolumeMount `json:"mounts,omitempty" desc:"Workloads currently mounting this volume (populated on detail get)."`
}

// DataVolumeMount is one workload currently mounting a data volume.
type DataVolumeMount struct {
	Workload  string `json:"workload" desc:"Controlling workload (or pod) name."`
	Kind      string `json:"kind,omitempty" desc:"Kubernetes controller kind (Deployment/StatefulSet/Job/Pod)."`
	MountPath string `json:"mountPath,omitempty" desc:"Mount path inside the pod."`
	Running   bool   `json:"running" desc:"Whether the mounting pod is currently running."`
}

// DataVolumeList is the LIST response.
type DataVolumeList struct {
	Items []DataVolume `json:"items" desc:"Data volumes in this page."`
	Count int          `json:"count" desc:"Number of volumes in this page."`
}

// DataVolumeCreateRequest is the body of POST /api/v1/datavolumes.
type DataVolumeCreateRequest struct {
	Name         string    `json:"name" binding:"required,dns1123,min=1,max=63" desc:"Data volume name, unique within the tenant."`
	Size         string    `json:"size" binding:"required" desc:"Requested storage size as a Kubernetes quantity (e.g. 200Gi)."`
	StorageClass string    `json:"storageClass,omitempty" desc:"StorageClass backing the volume; cluster default when empty."`
	AccessModes  []string  `json:"accessModes,omitempty" desc:"Access modes; defaults to [ReadWriteOnce] when empty."`
	Description  string    `json:"description,omitempty" binding:"max=1000" desc:"Free-text description of the volume."`
	Labels       StringMap `json:"labels,omitempty" desc:"User-defined labels to set on the volume."`
}

// DataVolumePatchRequest is the body of PATCH /api/v1/datavolumes/{name}. Size
// is expand-only; storageClass and accessModes are immutable.
type DataVolumePatchRequest struct {
	Size        *string   `json:"size,omitempty" desc:"New storage size; expand-only (must be >= current)."`
	Description *string   `json:"description,omitempty" desc:"Replacement free-text description."`
	Labels      StringMap `json:"labels,omitempty" desc:"Replacement user-defined label set."`
}

// StorageClass is a cluster-level storage backend offered when creating a volume.
type StorageClass struct {
	Name                 string `json:"name" desc:"StorageClass name."`
	Provisioner          string `json:"provisioner,omitempty" desc:"Provisioner backing the class."`
	Default              bool   `json:"default" desc:"Whether this is the cluster default StorageClass."`
	AllowVolumeExpansion bool   `json:"allowVolumeExpansion" desc:"Whether volumes on this class can be expanded."`
}

// StorageClassList is the LIST response for storage classes.
type StorageClassList struct {
	Items []StorageClass `json:"items" desc:"Available storage classes."`
	Count int            `json:"count" desc:"Number of storage classes."`
}
