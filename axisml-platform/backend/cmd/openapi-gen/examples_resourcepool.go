package main

import "github.com/axisml/axisml/pkg/openapigen"

// exResourcePool holds whole-object examples for the resourcepool.go DTOs. A pool
// is cluster-scoped (e.g. "gpu-a100"); a unit is a shape within it (e.g. "a100-2x"
// = 2 GPUs + cpu/mem).
func exResourcePool(g *openapigen.Generator) {
	unit := obj{
		"name":        "a100-2x",
		"description": "2x A100 GPU compute unit.",
		"requests":    obj{"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "2"},
		"limits":      obj{"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "2"},
	}
	g.SetExample("ResourceUnit", unit)
	g.SetExample("ResourceUnitList", obj{
		"items":         []any{unit},
		"count":         1,
		"continueToken": "",
	})
	g.SetExample("ResourceUnitCreateRequest", obj{
		"name":        "a100-2x",
		"description": "2x A100 GPU compute unit.",
		"requests":    obj{"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "2"},
		"limits":      obj{"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "2"},
	})
	g.SetExample("ResourceUnitPatchRequest", obj{
		"description": "Updated 2x A100 GPU compute unit.",
		"limits":      obj{"cpu": "24", "memory": "192Gi", "nvidia.com/gpu": "2"},
	})

	pool := obj{
		"name":            "gpu-a100",
		"description":     "A100 GPU resource pool.",
		"nodeSelector":    obj{"axisml.io/gpu": "a100"},
		"units":           []any{unit},
		"labels":          obj{"tier": "gpu"},
		"nodeCount":       8,
		"resourceVersion": "184321",
		"createdAt":       exCreatedAt,
		"updatedAt":       exUpdatedAt,
	}
	g.SetExample("ResourcePool", pool)
	g.SetExample("ResourcePoolList", obj{
		"items":         []any{pool},
		"count":         1,
		"continueToken": "",
	})
	g.SetExample("ResourcePoolCreateRequest", obj{
		"name":         "gpu-a100",
		"description":  "A100 GPU resource pool.",
		"nodeSelector": obj{"axisml.io/gpu": "a100"},
		"units": []any{obj{
			"name":        "a100-2x",
			"description": "2x A100 GPU compute unit.",
			"requests":    obj{"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "2"},
			"limits":      obj{"cpu": "16", "memory": "128Gi", "nvidia.com/gpu": "2"},
		}},
		"labels": obj{"tier": "gpu"},
	})
	g.SetExample("ResourcePoolPatchRequest", obj{
		"description":  "Updated A100 GPU resource pool.",
		"nodeSelector": obj{"axisml.io/gpu": "a100", "axisml.io/zone": "cn-east-1"},
	})
}
