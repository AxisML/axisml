package main

import "github.com/axisml/axisml/pkg/openapigen"

// exProxy holds whole-object examples for the proxy.go DTOs (Pod/Event surfaces).
func exProxy(g *openapigen.Generator) {
	pod := obj{
		"name":         "resnet-train-7-worker-0",
		"role":         "worker",
		"replicaIndex": 0,
		"phase":        "Running",
		"nodeName":     "gpu-node-03",
		"startedAt":    exStartedAt,
		"restartCount": 0,
	}
	g.SetExample("Pod", pod)
	g.SetExample("PodList", obj{
		"items": []any{pod},
		"count": 1,
	})

	event := obj{
		"type":           "Normal",
		"reason":         "Scheduled",
		"message":        "Successfully assigned resnet-train-7-worker-0 to gpu-node-03.",
		"source":         "default-scheduler",
		"firstTimestamp": exStartedAt,
		"lastTimestamp":  exStartedAt,
		"count":          1,
		"involvedObject": obj{
			"kind":      "Pod",
			"name":      "resnet-train-7-worker-0",
			"namespace": "axisml-team-vision",
		},
	}
	g.SetExample("Event", event)
	g.SetExample("EventList", obj{
		"items": []any{event},
		"count": 1,
	})
}
