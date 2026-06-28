package main

import "github.com/axisml/axisml/pkg/openapigen"

// exMLService holds whole-object examples for the mlservice.go DTOs.
func exMLService(g *openapigen.Generator) {
	port := obj{"name": "http", "port": 8080}
	g.SetExample("ServicePort", port)

	route := obj{"enabled": true, "path": "/v1/models/llama3-8b"}
	g.SetExample("MLServiceRoute", route)

	service := obj{
		"id":               "5d2c9b41-3e8f-4a1c-9d7e-6b4f2a1c8e90",
		"namespace":        "team-nlp",
		"tenantName":       "team-nlp",
		"computeNamespace": "axisml-team-nlp",
		"name":             "llama3-chat",
		"displayName":      "Llama3 chat service",
		"description":      "Llama3-8B online inference service.",
		"owner":            "zhang.san",
		"ownerId":          "9f8e7d6c-5b4a-3210-fedc-ba9876543210",
		"backend":          obj{"name": "kserve", "engine": "llminference"},
		"modelName":        "llama3-8b",
		"modelVersion":     "1.2.0",
		"image":            "registry.axisml.io/serving/vllm:0.6.0",
		"env":              []any{obj{"name": "MAX_TOKENS", "value": "4096"}},
		"ports":            []any{port},
		"poolName":         "gpu-a100",
		"unitName":         "a100-1x",
		"quota":            "team-nlp",
		"resources":        obj{"cpu": "8", "memory": "64Gi", "nvidia.com/gpu": "1"},
		"replicas":         3,
		"readyReplicas":    3,
		"route":            route,
		"accessUrl":        "https://gateway.axisml.io/v1/models/llama3-8b",
		"phase":            "Running",
		"message":          "All replicas ready.",
		"createdAt":        exCreatedAt,
		"updatedAt":        exUpdatedAt,
	}
	g.SetExample("MLService", service)
	g.SetExample("MLServiceList", obj{
		"items":         []any{service},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})

	g.SetExample("MLServiceCreateRequest", obj{
		"name":         "llama3-chat",
		"displayName":  "Llama3 chat service",
		"description":  "Llama3-8B online inference service.",
		"backend":      obj{"name": "kserve", "engine": "llminference"},
		"modelName":    "llama3-8b",
		"modelVersion": "1.2.0",
		"image":        "registry.axisml.io/serving/vllm:0.6.0",
		"env":          []any{obj{"name": "MAX_TOKENS", "value": "4096"}},
		"ports":        []any{port},
		"poolName":     "gpu-a100",
		"unitName":     "a100-1x",
		"quota":        "team-nlp",
		"replicas":     3,
		"route":        route,
	})
	g.SetExample("MLServicePatchRequest", obj{
		"displayName": "Llama3 chat service (v2)",
		"description": "Updated description.",
	})
	g.SetExample("MLServiceScaleRequest", obj{"replicas": 5})

	point := obj{"timestamp": exStartedAt, "value": 12.5}
	g.SetExample("MetricPoint", point)
	g.SetExample("MetricSeries", obj{
		"metric":     "qps",
		"range":      "1h",
		"step":       "1m",
		"percentile": "p95",
		"unit":       "req/s",
		"series": []any{
			obj{"timestamp": exStartedAt, "value": 12.5},
			obj{"timestamp": exUpdatedAt, "value": 18.3},
			obj{"timestamp": exFinishedAt, "value": 15.1},
		},
	})
}
