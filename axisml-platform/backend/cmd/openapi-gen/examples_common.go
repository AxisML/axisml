package main

import "github.com/axisml/axisml/pkg/openapigen"

// exCommon holds whole-object examples for the common.go DTOs.
func exCommon(g *openapigen.Generator) {
	g.SetExample("ProblemFieldError", obj{
		"field":   "spec.roles",
		"message": "must contain at least one role",
	})

	g.SetExample("Problem", obj{
		"type":     "https://docs.axisml.io/errors/not-found",
		"title":    "Not Found",
		"status":   404,
		"detail":   "Job \"resnet-train\" was not found in tenant \"team-vision\".",
		"instance": "/api/v1/jobs/resnet-train",
		"code":     "JOB_NOT_FOUND",
	})

	g.SetExample("HealthStatus", obj{
		"status": "ok",
		"components": obj{
			"database":        "ok",
			"compute-service": "ok",
			"artifact-hub":    "ok",
		},
	})

	g.SetExample("EnvVar", obj{
		"name":  "NCCL_DEBUG",
		"value": "INFO",
	})

	g.SetExample("Condition", obj{
		"type":               "Ready",
		"status":             "True",
		"reason":             "AllReplicasReady",
		"message":            "All worker replicas are ready.",
		"lastTransitionTime": exUpdatedAt,
	})
}
