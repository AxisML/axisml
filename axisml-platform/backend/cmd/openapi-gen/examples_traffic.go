package main

import "github.com/axisml/axisml/pkg/openapigen"

// exTraffic holds whole-object examples for the traffic.go DTOs. A TrafficPolicy
// fans one stable external entry across member online services by weight; the
// example shows a canary split (90 stable / 10 canary, summing to 100).
func exTraffic(g *openapigen.Generator) {
	endpoint := obj{
		"path":     "/services/team-vision/resnet-serving/",
		"hostname": "infer.axisml.io",
	}
	g.SetExample("TrafficPolicyEndpoint", endpoint)

	stableBackend := obj{
		"serviceName": "resnet-serving-v1",
		"role":        "stable",
		"weight":      90,
		"actualPct":   90,
		"ready":       true,
	}
	canaryBackend := obj{
		"serviceName": "resnet-serving-v2",
		"role":        "canary",
		"weight":      10,
		"actualPct":   10,
		"ready":       true,
	}
	g.SetExample("TrafficPolicyBackend", stableBackend)

	policy := obj{
		"id":            "d4e5f6a7-8b9c-0d1e-2f3a-4b5c6d7e8f90",
		"namespace":     "team-vision",
		"tenantName":    "team-vision",
		"name":          "resnet-serving",
		"displayName":   "ResNet inference traffic",
		"description":   "Canary traffic split for the ResNet-50 online inference service.",
		"owner":         "li.wei",
		"ownerId":       "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"mode":          "canary",
		"endpoint":      endpoint,
		"accessUrl":     "https://infer.axisml.io/services/team-vision/resnet-serving/",
		"backends":      []any{stableBackend, canaryBackend},
		"canaryPercent": 10,
		"phase":         "Ready",
		"message":       "Routing 90/10 between stable and canary.",
		"createdAt":     exCreatedAt,
		"updatedAt":     exUpdatedAt,
	}
	g.SetExample("TrafficPolicy", policy)
	g.SetExample("TrafficPolicyList", obj{
		"items":         []any{policy},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})

	backendSpec := obj{
		"serviceName": "resnet-serving-v2",
		"role":        "canary",
		"weight":      10,
	}
	g.SetExample("TrafficPolicyBackendSpec", backendSpec)

	g.SetExample("TrafficPolicyCreateRequest", obj{
		"name":        "resnet-serving",
		"displayName": "ResNet inference traffic",
		"description": "Canary traffic split for the ResNet-50 online inference service.",
		"mode":        "canary",
		"endpoint":    endpoint,
		"backends": []any{
			obj{"serviceName": "resnet-serving-v1", "role": "stable", "weight": 90},
			obj{"serviceName": "resnet-serving-v2", "role": "canary", "weight": 10},
		},
		"canaryPercent": 10,
	})

	g.SetExample("TrafficPolicyPatchRequest", obj{
		"displayName": "ResNet inference traffic (v2)",
		"description": "Updated description.",
	})

	g.SetExample("TrafficPolicySplitRequest", obj{
		"backends": []any{
			obj{"serviceName": "resnet-serving-v1", "role": "stable", "weight": 75},
			obj{"serviceName": "resnet-serving-v2", "role": "canary", "weight": 25},
		},
		"canaryPercent": 25,
	})
}
