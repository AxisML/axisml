package apidoc

import "github.com/axisml/axisml/pkg/openapigen"

// obj is a terse alias for the example object literals below.
type obj = map[string]any

// Example timestamps. Kept consistent across schemas so a reader can follow one
// run/service/policy through its create → status timeline.
const (
	exCreatedAt = "2026-06-28T09:25:00Z"
	exStartedAt = "2026-06-28T09:30:00Z"
	exUpdatedAt = "2026-06-28T09:45:00Z"
)

// registerExamples attaches whole-object examples to every schema registered in
// Document(). It reuses the platform narrative (tenant "team-vision", pool
// "gpu-a100"/unit "a100-2x", an MLRun "resnet-train-7", an MLService
// "llama3-8b", a 90/10 canary policy) so the compute-service spec stays
// internally consistent with the platform spec. Each *List reuses its item.
func registerExamples(g *openapigen.Generator) {
	// ---- ComputeServiceError (RFC7807) -------------------------------------
	g.SetExample("ComputeServiceError", obj{
		"type":     "https://axisml.io/errors/quota_exceeded",
		"title":    "ElasticQuota team-vision exhausted",
		"status":   422,
		"detail":   "requested 8 nvidia.com/gpu exceeds the remaining quota of 2",
		"instance": "/api/v1/namespaces/team-vision/mlruns",
		"code":     "quota_exceeded",
		"details":  obj{"requested": "8", "available": "2", "resource": "nvidia.com/gpu"},
	})

	// ---- MLRun -------------------------------------------------------------
	mlrunBackend := obj{"name": "kubeflow-trainer", "engine": "pytorchjob"}
	mlrunRole := obj{
		"name":          "worker",
		"replicas":      4,
		"restartPolicy": "OnFailure",
		"template": obj{
			"image":   "registry.axisml.io/training/resnet:1.4.0",
			"command": []any{"python", "train.py"},
			"args":    []any{"--epochs", "90", "--batch-size", "256"},
			"env":     []any{obj{"name": "NCCL_DEBUG", "value": "INFO"}},
			"resources": obj{
				"requests": obj{"cpu": "8", "memory": "64Gi", "nvidia.com/gpu": "2"},
				"limits":   obj{"cpu": "8", "memory": "64Gi", "nvidia.com/gpu": "2"},
			},
		},
	}
	mlrunRunPolicy := obj{
		"activeDeadlineSeconds":   86400,
		"ttlSecondsAfterFinished": 3600,
		"backoffLimit":            2,
	}
	mlrunSpec := obj{
		"backend": mlrunBackend,
		"scheduling": obj{
			"quota":         "axisml-team-vision-gpu-a100-default",
			"priorityClass": "high-priority",
		},
		"roles":     []any{mlrunRole},
		"runPolicy": mlrunRunPolicy,
	}

	g.SetExample("MLRunCreateRequest", obj{
		"name":        "resnet-train-7",
		"displayName": "ResNet-50 Training #7",
		"description": "Distributed ResNet-50 training on ImageNet.",
		"labels":      obj{"team": "vision"},
		"poolName":    "gpu-a100",
		"unitName":    "a100-2x",
		"quota":       "axisml-team-vision-gpu-a100-default",
		"backend":     mlrunBackend,
		"roles":       []any{mlrunRole},
		"runPolicy":   mlrunRunPolicy,
	})
	g.SetExample("MLRunPatchRequest", obj{
		"displayName": "ResNet-50 Training #7 (rerun)",
		"description": "Updated description.",
	})
	mlRun := obj{
		"id":          "b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5",
		"namespace":   "team-vision",
		"name":        "resnet-train-7",
		"displayName": "ResNet-50 Training #7",
		"description": "Distributed ResNet-50 training on ImageNet.",
		"owner":       "li.wei",
		"labels":      obj{"team": "vision"},
		"phase":       "Running",
		"spec":        mlrunSpec,
		"status": obj{
			"message":   "All worker replicas ready.",
			"startedAt": exStartedAt,
			"conditions": []any{obj{
				"type":               "Scheduled",
				"status":             "True",
				"reason":             "PodGroupScheduled",
				"message":            "Gang scheduled onto gpu-a100.",
				"lastTransitionTime": exStartedAt,
			}},
		},
		"createdAt": exCreatedAt,
		"updatedAt": exUpdatedAt,
	}
	g.SetExample("MLRun", mlRun)

	// ---- MLService ---------------------------------------------------------
	mlsvcBackend := obj{"name": "kserve", "engine": "llminference"}
	mlsvcRole := obj{
		"name":     "predictor",
		"replicas": 2,
		"template": obj{
			"image": "registry.axisml.io/serving/vllm:0.6.2",
			"args":  []any{"--model", "meta-llama/Llama-3-8b", "--max-model-len", "8192"},
			"ports": []any{obj{"name": "http", "containerPort": 8080, "protocol": "TCP"}},
			"resources": obj{
				"requests": obj{"cpu": "8", "memory": "48Gi", "nvidia.com/gpu": "1"},
				"limits":   obj{"cpu": "8", "memory": "48Gi", "nvidia.com/gpu": "1"},
			},
		},
	}
	mlsvcRoute := obj{
		"enabled":    true,
		"targetRole": "predictor",
		"portName":   "http",
		"hostname":   "llama3-8b.team-vision.axisml.io",
		"path":       "/v1",
		"auth":       obj{"type": "jwt", "jwt": obj{"issuer": "https://auth.axisml.io", "jwksUri": "https://auth.axisml.io/.well-known/jwks.json"}},
	}
	mlsvcSpec := obj{
		"backend": mlsvcBackend,
		"scheduling": obj{
			"quota": "axisml-team-vision-gpu-a100-default",
		},
		"roles":     []any{mlsvcRole},
		"runPolicy": obj{"progressDeadlineSeconds": 600},
		"route":     mlsvcRoute,
	}

	g.SetExample("MLServiceCreateRequest", obj{
		"name":        "llama3-8b",
		"kind":        "service",
		"displayName": "Llama-3 8B inference service",
		"description": "Llama-3 8B online inference on the vLLM backend.",
		"labels":      obj{"team": "vision"},
		"poolName":    "gpu-a100",
		"unitName":    "a100-2x",
		"quota":       "axisml-team-vision-gpu-a100-default",
		"backend":     mlsvcBackend,
		"roles":       []any{mlsvcRole},
		"runPolicy":   obj{"progressDeadlineSeconds": 600},
		"route":       mlsvcRoute,
	})
	g.SetExample("MLServicePatchRequest", obj{
		"displayName": "Llama-3 8B inference service (production)",
		"description": "Updated description.",
	})
	g.SetExample("MLServiceScaleRequest", obj{"replicas": 4})
	mlService := obj{
		"id":                 "c2e1a0b9-8d7c-6b5a-4f3e-2d1c0b9a8f7e",
		"namespace":          "team-vision",
		"name":               "llama3-8b",
		"kind":               "service",
		"displayName":        "Llama-3 8B inference service",
		"description":        "Llama-3 8B online inference on the vLLM backend.",
		"owner":              "li.wei",
		"labels":             obj{"team": "vision"},
		"generation":         3,
		"observedGeneration": 3,
		"phase":              "Ready",
		"spec":               mlsvcSpec,
		"status": obj{
			"message":       "2/2 replicas ready.",
			"readyReplicas": 2,
			"endpoint":      "https://llama3-8b.team-vision.axisml.io/v1",
			"conditions": []any{obj{
				"type":               "Available",
				"status":             "True",
				"reason":             "MinimumReplicasAvailable",
				"message":            "Deployment has minimum availability.",
				"lastTransitionTime": exStartedAt,
			}},
		},
		"createdAt": exCreatedAt,
		"updatedAt": exUpdatedAt,
	}
	g.SetExample("MLService", mlService)

	// ---- TrafficPolicy (90/10 canary) --------------------------------------
	tpBackend := obj{"name": "kserve", "engine": "inference"}
	tpEndpoint := obj{
		"path":     "/v1",
		"hostname": "llama3-8b.team-vision.axisml.io",
		"auth":     obj{"type": "jwt", "jwt": obj{"issuer": "https://auth.axisml.io", "jwksUri": "https://auth.axisml.io/.well-known/jwks.json", "audience": "axisml-inference"}},
	}
	tpMembers := []any{
		obj{"serviceName": "llama3-8b", "role": "stable", "weight": 90},
		obj{"serviceName": "llama3-8b-v2", "role": "canary", "weight": 10},
	}
	tpSpec := obj{
		"backend":  tpBackend,
		"mode":     "canary",
		"endpoint": tpEndpoint,
		"backends": tpMembers,
	}

	g.SetExample("TrafficPolicyCreateRequest", obj{
		"name":        "llama3-canary",
		"displayName": "Llama-3 canary release",
		"description": "Canary 10% traffic to v2.",
		"labels":      obj{"team": "vision"},
		"mode":        "canary",
		"endpoint":    tpEndpoint,
		"backends":    tpMembers,
	})
	g.SetExample("TrafficPolicyPatchRequest", obj{
		"displayName": "Llama-3 canary release (under evaluation)",
		"description": "Updated description.",
	})
	g.SetExample("TrafficPolicySplitRequest", obj{
		"backends": []any{
			obj{"serviceName": "llama3-8b", "weight": 80},
			obj{"serviceName": "llama3-8b-v2", "weight": 20},
		},
	})
	trafficPolicy := obj{
		"id":                 "d3f2b1c0-9e8d-7c6b-5a4f-3e2d1c0b9a8f",
		"namespace":          "team-vision",
		"name":               "llama3-canary",
		"mode":               "canary",
		"displayName":        "Llama-3 canary release",
		"description":        "Canary 10% traffic to v2.",
		"owner":              "li.wei",
		"labels":             obj{"team": "vision"},
		"generation":         2,
		"observedGeneration": 2,
		"phase":              "Ready",
		"spec":               tpSpec,
		"status": obj{
			"message":  "Route programmed; weights applied.",
			"endpoint": "https://llama3-8b.team-vision.axisml.io/v1",
			"backends": []any{
				obj{"serviceName": "llama3-8b", "weight": 90, "ready": true},
				obj{"serviceName": "llama3-8b-v2", "weight": 10, "ready": true},
			},
			"conditions": []any{obj{
				"type":               "Ready",
				"status":             "True",
				"reason":             "Programmed",
				"message":            "HTTPRoute programmed with weighted backends.",
				"lastTransitionTime": exStartedAt,
			}},
		},
		"createdAt": exCreatedAt,
		"updatedAt": exUpdatedAt,
	}
	g.SetExample("TrafficPolicy", trafficPolicy)

	// ---- Pod / Event -------------------------------------------------------
	pod := obj{
		"name":      "resnet-train-7-worker-0",
		"namespace": "team-vision",
		"phase":     "Running",
		"nodeName":  "gpu-node-a100-03",
		"labels":    obj{"axisml.io/run-id": "b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5", "axisml.io/role": "worker"},
	}
	g.SetExample("Pod", pod)
	event := obj{
		"reason":              "Scheduled",
		"note":                "Successfully assigned team-vision/resnet-train-7-worker-0 to gpu-node-a100-03",
		"type":                "Normal",
		"object":              "Pod/resnet-train-7-worker-0",
		"reportingController": "koord-scheduler",
		"eventTime":           exStartedAt,
	}
	g.SetExample("Event", event)

	// ---- Capabilities ------------------------------------------------------
	g.SetExample("Capabilities", obj{
		"runtime":          "kubernetes",
		"quotaEnforcement": true,
	})

	// ---- List envelopes ({items, total}) — registered in Document() after
	// this call, so each reuses its item example above. ----------------------
	g.SetExample("MLRunList", obj{"items": []any{mlRun}, "total": 1})
	g.SetExample("MLServiceList", obj{"items": []any{mlService}, "total": 1})
	g.SetExample("TrafficPolicyList", obj{"items": []any{trafficPolicy}, "total": 1})
	g.SetExample("PodList", obj{"items": []any{pod}, "total": 1})
	g.SetExample("EventList", obj{"items": []any{event}, "total": 1})
}
