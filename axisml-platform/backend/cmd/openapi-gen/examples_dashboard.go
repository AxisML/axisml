package main

import "github.com/axisml/axisml/pkg/openapigen"

// exDashboard holds whole-object examples for the dashboard.go DTOs
// (cluster resource usage + recent-activity feed).
func exDashboard(g *openapigen.Generator) {
	gpuMeter := obj{"resource": "gpu", "used": 36, "total": 48, "unit": "cards"}
	cpuMeter := obj{"resource": "cpu", "used": 740, "total": 1152, "unit": "cores"}
	memMeter := obj{"resource": "memory", "used": 3481.6, "total": 5632, "unit": "GiB"}
	g.SetExample("ClusterMeter", gpuMeter)

	poolUsage := obj{
		"pool":   "gpu-a100",
		"meters": []any{obj{"resource": "gpu", "used": 22, "total": 32, "unit": "cards"}, obj{"resource": "cpu", "used": 240, "total": 384, "unit": "cores"}, obj{"resource": "memory", "used": 1228.8, "total": 2048, "unit": "GiB"}},
	}
	g.SetExample("ClusterPoolUsage", poolUsage)

	g.SetExample("ClusterUsage", obj{
		"aggregate": []any{gpuMeter, cpuMeter, memMeter},
		"pools": []any{
			poolUsage,
			obj{"pool": "h100-pool", "meters": []any{obj{"resource": "gpu", "used": 14, "total": 16, "unit": "cards"}, obj{"resource": "cpu", "used": 180, "total": 256, "unit": "cores"}, obj{"resource": "memory", "used": 1433.6, "total": 2048, "unit": "GiB"}}},
			obj{"pool": "cpu-pool", "meters": []any{obj{"resource": "gpu", "used": 0, "total": 0, "unit": "cards"}, obj{"resource": "cpu", "used": 320, "total": 512, "unit": "cores"}, obj{"resource": "memory", "used": 819.2, "total": 1536, "unit": "GiB"}}},
		},
		"updatedAt": exUpdatedAt,
	})

	activity := obj{
		"id":        "act-9f3a2e5c",
		"kind":      "run",
		"name":      "resnet-train-7",
		"action":    "succeeded",
		"phase":     "Succeeded",
		"actor":     "li.wei",
		"timestamp": exFinishedAt,
	}
	g.SetExample("ActivityItem", activity)
	g.SetExample("ActivityList", obj{
		"items": []any{
			activity,
			obj{"id": "act-7b4f2a1c", "kind": "mlservice", "name": "llama3-chat", "action": "started", "phase": "Ready", "actor": "zhang.san", "timestamp": exUpdatedAt},
			obj{"id": "act-3e8f4a1c", "kind": "workspace", "name": "notebook-dev", "action": "created", "phase": "Running", "actor": "li.wei", "timestamp": exCreatedAt},
		},
		"count": 3,
	})
}
