package apidoc

import "github.com/axisml/axisml/pkg/openapigen"

// obj is a terse alias for authoring whole-object schema examples.
type obj = map[string]any

const (
	exCreatedAt = "2026-06-20T08:00:00Z"
	exUpdatedAt = "2026-06-28T09:30:00Z"
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
		"resourceVersion": "184729",
		"createdAt":       exCreatedAt,
		"updatedAt":       exUpdatedAt,
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

	tenant := obj{
		"name":      "team-vision",
		"namespace": obj{"name": "team-vision"},
		"quotas":    []any{quota},
		"labels":    obj{"displayName": "Vision Team"},
		"annotations": obj{
			"axisml.io/last-modified-by": "li.wei",
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
		"units": []any{obj{
			"unitName": "a100-2x",
			"quantity": 6,
		}},
	})

	// --- Volumes ---

	g.SetExample("CreateVolumeRequest", obj{
		"namespace":    "team-vision",
		"name":         "axisml-ws-notebook-data",
		"size":         "50Gi",
		"storageClass": "fast-ssd",
	})
	g.SetExample("Volume", obj{
		"namespace":    "team-vision",
		"name":         "axisml-ws-notebook-data",
		"size":         "50Gi",
		"storageClass": "fast-ssd",
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
