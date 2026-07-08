package apidoc

import "github.com/axisml/axisml/pkg/openapigen"

// obj is a terse alias for authoring whole-object schema examples.
type obj = map[string]any

const (
	exCreatedAt = "2026-06-20T08:00:00Z"
)

// registerExamples attaches a realistic, internally-consistent whole-object
// example to every schema registered in Document. The narrative mirrors the
// platform examples for cross-spec consistency: tenant "team-vision" /
// "Vision Team", pool "gpu-a100" with unit "a100-2x" (2 GPUs), quota quantity 4.
func registerExamples(g *openapigen.Generator) {
	// --- Resource pools & units ---

	unit := obj{
		"name":        "a100-2x",
		"description": "2× A100 GPU compute unit.",
		"requests": obj{
			"cpu":            "16",
			"memory":         "128Gi",
			"nvidia.com/gpu": "2",
		},
		"limits": obj{
			"cpu":            "16",
			"memory":         "128Gi",
			"nvidia.com/gpu": "2",
		},
		"nodeSelector": obj{"axisml.io/gpu": "a100"},
		"annotations":  obj{"tenant.axisml.io/managed-by": "platform"},
	}
	g.SetExample("ResourceUnit", unit)
	g.SetExample("ResourceUnitList", obj{
		"items": []any{unit},
		"count": 1,
	})

	createUnit := obj{
		"name":        "a100-2x",
		"description": "2× A100 GPU compute unit.",
		"requests": obj{
			"cpu":            "16",
			"memory":         "128Gi",
			"nvidia.com/gpu": "2",
		},
		"limits": obj{
			"cpu":            "16",
			"memory":         "128Gi",
			"nvidia.com/gpu": "2",
		},
		"nodeSelector": obj{"axisml.io/gpu": "a100"},
	}
	g.SetExample("CreateResourceUnitRequest", createUnit)
	g.SetExample("PatchResourceUnitRequest", obj{
		"description": "2× A100 GPU compute unit (updated).",
		"limits": obj{
			"cpu":            "24",
			"memory":         "192Gi",
			"nvidia.com/gpu": "2",
		},
	})

	pool := obj{
		"name":         "gpu-a100",
		"description":  "A100 GPU resource pool.",
		"nodeSelector": obj{"axisml.io/gpu": "a100"},
		"tolerations": []any{obj{
			"key":      "nvidia.com/gpu",
			"operator": "Exists",
			"effect":   "NoSchedule",
		}},
		"units":           []any{unit},
		"labels":          obj{"tier": "gpu"},
		"annotations":     obj{"tenant.axisml.io/managed-by": "platform"},
		"resourceVersion": "184729",
		"createdAt":       exCreatedAt,
	}
	g.SetExample("ResourcePool", pool)
	g.SetExample("ResourcePoolList", obj{
		"items":         []any{pool},
		"count":         1,
		"continueToken": "",
	})

	g.SetExample("CreateResourcePoolRequest", obj{
		"name":         "gpu-a100",
		"description":  "A100 GPU resource pool.",
		"nodeSelector": obj{"axisml.io/gpu": "a100"},
		"tolerations": []any{obj{
			"key":      "nvidia.com/gpu",
			"operator": "Exists",
			"effect":   "NoSchedule",
		}},
		"units":  []any{createUnit},
		"labels": obj{"tier": "gpu"},
	})
	g.SetExample("PatchResourcePoolRequest", obj{
		"description": "A100 GPU resource pool (updated).",
		"labels":      obj{"tier": "gpu", "region": "cn-east"},
	})

	// --- Tenants & quotas ---

	quota := obj{
		"pool": "gpu-a100",
		"units": []any{obj{
			"unitName": "a100-2x",
			"quantity": 4,
		}},
	}
	g.SetExample("Quota", quota)
	g.SetExample("QuotaList", obj{
		"items": []any{quota},
		"count": 1,
	})

	initResources := obj{
		"configMaps": []any{obj{
			"name": "shared-config",
			"sourceConfigMapRef": obj{
				"namespace": "axisml-system",
				"name":      "tenant-shared-config",
			},
		}},
		"imagePullSecrets": []any{obj{
			"name": "registry-pull",
			"sourceSecretRef": obj{
				"namespace": "axisml-system",
				"name":      "registry-pull-credentials",
			},
		}},
		"secrets": []any{obj{
			"name": "wandb-api-key",
			"type": "Opaque",
			"sourceSecretRef": obj{
				"namespace": "axisml-system",
				"name":      "wandb-api-key",
			},
		}},
		"serviceAccounts": []any{obj{
			"name":             "trainer",
			"imagePullSecrets": []any{"registry-pull"},
			"rbac": obj{
				"rules": []any{obj{
					"apiGroups": []any{""},
					"resources": []any{"pods", "pods/log"},
					"verbs":     []any{"get", "list", "watch"},
				}},
			},
		}},
	}

	tenant := obj{
		"name":          "team-vision",
		"namespace":     obj{"name": "team-vision"},
		"quotas":        []any{quota},
		"initResources": initResources,
		"labels":        obj{"displayName": "Vision Team"},
		"annotations": obj{
			"resource.axisml.io/last-modified-by": "li.wei",
		},
		"resourceVersion": "184730",
		"phase":           "Ready",
		"status": obj{
			"observedGeneration": 3,
			"phase":              "Ready",
			"message":            "Namespace and quotas provisioned.",
			"namespaceReady":     true,
			"quotas": []any{obj{
				"pool":  "gpu-a100",
				"ready": true,
				"used": obj{
					"cpu":            "32",
					"memory":         "256Gi",
					"nvidia.com/gpu": "4",
				},
			}},
		},
		"createdAt": exCreatedAt,
	}
	g.SetExample("Tenant", tenant)
	g.SetExample("TenantList", obj{
		"items":         []any{tenant},
		"count":         1,
		"continueToken": "",
	})

	g.SetExample("CreateTenantRequest", obj{
		"name":      "team-vision",
		"namespace": obj{"name": "team-vision"},
		"quotas":    []any{quota},
		"labels":    obj{"displayName": "Vision Team"},
	})
	g.SetExample("PatchTenantRequest", obj{
		"namespaceLabels": obj{"team": "vision"},
		"labels":          obj{"displayName": "Vision Team", "region": "cn-east"},
	})

	g.SetExample("SetQuotaRequest", quota)
	g.SetExample("PatchQuotaRequest", obj{
		"quota": obj{
			"min": obj{"cpu": "32", "memory": "256Gi"},
			"max": obj{"nvidia.com/gpu": "8"},
		},
	})

	// --- Volumes ---

	g.SetExample("CreateVolumeRequest", obj{
		"namespace":    "team-vision",
		"name":         "shared-datasets",
		"size":         "2Ti",
		"storageClass": "nfs-rwx",
		"accessModes":  []any{"ReadWriteMany"},
		"description":  "Shared raw datasets directory",
	})
	g.SetExample("PatchVolumeRequest", obj{
		"size":        "4Ti",
		"description": "Shared raw datasets directory (expanded)",
	})
	g.SetExample("Volume", obj{
		"namespace":    "team-vision",
		"name":         "shared-datasets",
		"size":         "2Ti",
		"storageClass": "nfs-rwx",
		"accessModes":  []any{"ReadWriteMany"},
		"description":  "Shared raw datasets directory",
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
		"createdAt": "2026-02-11T08:00:00Z",
	})

	g.SetExample("StorageClass", obj{
		"name":                 "nfs-rwx",
		"provisioner":          "nfs.csi.k8s.io",
		"default":              false,
		"allowVolumeExpansion": true,
	})

	// --- Capabilities ---

	g.SetExample("Capabilities", obj{
		"multiTenant":           true,
		"resourcePoolsWritable": true,
	})

	// --- Errors ---

	g.SetExample("ClusterManagerError", obj{
		"type":   "about:blank",
		"title":  "Not found",
		"status": 404,
		"detail": "ResourcePool \"gpu-a100\" not found.",
		"code":   "NotFound",
	})
}
