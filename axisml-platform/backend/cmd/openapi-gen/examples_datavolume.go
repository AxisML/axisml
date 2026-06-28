package main

import "github.com/axisml/axisml/pkg/openapigen"

// exDataVolume holds whole-object examples for the datavolume.go DTOs. A data
// volume is a tenant-scoped durable PVC (e.g. "shared-datasets") the tenant's
// workloads mount; system-admins manage it.
func exDataVolume(g *openapigen.Generator) {
	vol := obj{
		"name":         "shared-datasets",
		"size":         "2Ti",
		"storageClass": "nfs-rwx",
		"accessModes":  []any{"ReadWriteMany"},
		"description":  "Shared raw datasets directory.",
		"labels":       obj{"team": "vision"},
		"status": obj{
			"phase":         "Bound",
			"boundCapacity": "2Ti",
			"mounts": []any{obj{
				"workload":  "ws-jupyter-3",
				"kind":      "Deployment",
				"mountPath": "/data/shared",
				"running":   true,
			}},
		},
		"createdAt": exCreatedAt,
	}
	g.SetExample("DataVolume", vol)
	g.SetExample("DataVolumeList", obj{
		"items": []any{vol},
		"count": 1,
	})
	g.SetExample("DataVolumeCreateRequest", obj{
		"name":         "shared-datasets",
		"size":         "2Ti",
		"storageClass": "nfs-rwx",
		"accessModes":  []any{"ReadWriteMany"},
		"description":  "Shared raw datasets directory.",
	})
	g.SetExample("DataVolumePatchRequest", obj{
		"size":        "4Ti",
		"description": "Shared raw datasets directory (expanded).",
	})
	sc := obj{"name": "nfs-rwx", "provisioner": "nfs.csi.k8s.io", "default": false, "allowVolumeExpansion": true}
	g.SetExample("StorageClass", sc)
	g.SetExample("StorageClassList", obj{"items": []any{sc}, "count": 1})
}
