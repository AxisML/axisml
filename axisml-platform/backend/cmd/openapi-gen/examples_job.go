package main

import "github.com/axisml/axisml/pkg/openapigen"

// exJob holds whole-object examples for the job.go DTOs (Job templates + Runs).
func exJob(g *openapigen.Generator) {
	backend := obj{"name": "native", "engine": "pytorchjob"}

	roleTemplate := obj{
		"image":        "registry.axisml.io/training/resnet:1.4.0",
		"command":      []any{"python", "train.py"},
		"args":         []any{"--epochs", "90", "--batch-size", "256"},
		"env":          []any{obj{"name": "NCCL_DEBUG", "value": "INFO"}},
		"ports":        []any{obj{"name": "http", "containerPort": 8080, "protocol": "TCP"}},
		"resources":    obj{"cpu": "8", "memory": "64Gi", "nvidia.com/gpu": "2"},
		"volumes":      []any{obj{"name": "data", "persistentVolumeClaim": obj{"claimName": "resnet-imagenet"}}},
		"volumeMounts": []any{obj{"name": "data", "mountPath": "/data"}},
	}

	role := obj{
		"name":          "worker",
		"replicas":      4,
		"restartPolicy": "OnFailure",
		"template":      roleTemplate,
	}

	runPolicy := obj{
		"activeDeadlineSeconds":   86400,
		"ttlSecondsAfterFinished": 3600,
		"backoffLimit":            2,
		"progressDeadlineSeconds": 600,
	}

	jobSpec := obj{
		"backend":   backend,
		"poolName":  "gpu-a100",
		"unitName":  "a100-2x",
		"roles":     []any{role},
		"runPolicy": runPolicy,
		"artifacts": []any{obj{"kind": "model", "name": "resnet50", "version": "1.4.0"}},
	}

	g.SetExample("Backend", backend)
	g.SetExample("RoleTemplate", roleTemplate)
	g.SetExample("MLRunRole", role)
	g.SetExample("RunPolicy", runPolicy)
	g.SetExample("ArtifactRef", obj{"kind": "model", "name": "resnet50", "version": "1.4.0"})
	g.SetExample("JobSpec", jobSpec)

	runSummary := obj{
		"count":       7,
		"active":      1,
		"recent":      []any{"Succeeded", "Failed", "Succeeded", "Succeeded", "Running"},
		"latestPhase": "Running",
		"latestRunAt": exStartedAt,
	}
	g.SetExample("RunSummary", runSummary)

	job := obj{
		"id":          "8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10",
		"namespace":   "team-vision",
		"tenantName":  "team-vision",
		"name":        "resnet-train",
		"displayName": "ResNet-50 Training",
		"description": "Distributed ResNet-50 training job on ImageNet.",
		"owner":       "li.wei",
		"ownerId":     "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"labels":      obj{"team": "vision"},
		"annotations": obj{"axisml.io/created-by": "li.wei", "git-commit": "8c1f4e2"},
		"spec":        jobSpec,
		"runSummary":  runSummary,
		"createdAt":   exCreatedAt,
		"updatedAt":   exUpdatedAt,
	}
	g.SetExample("Job", job)
	g.SetExample("JobList", obj{
		"items":         []any{job},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("JobCreateRequest", obj{
		"name":        "resnet-train",
		"displayName": "ResNet-50 Training",
		"description": "Distributed ResNet-50 training job on ImageNet.",
		"labels":      obj{"team": "vision"},
		"spec":        jobSpec,
	})
	g.SetExample("JobPatchRequest", obj{
		"displayName": "ResNet-50 Training (v2)",
		"description": "Updated description.",
	})

	roleStatus := obj{
		"name":              "worker",
		"replicas":          4,
		"activeReplicas":    4,
		"readyReplicas":     4,
		"succeededReplicas": 0,
		"failedReplicas":    0,
		"template":          roleTemplate,
		"restartPolicy":     "OnFailure",
	}
	g.SetExample("MLRunRoleStatus", roleStatus)

	mlRunSpec := obj{
		"backend":    backend,
		"scheduling": obj{"quota": "axisml-team-vision-gpu-a100", "priorityClass": "high-priority", "minMember": 4},
		"roles":      []any{role},
		"runPolicy":  runPolicy,
	}
	g.SetExample("MLRunSpec", mlRunSpec)

	run := obj{
		"id":                "b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5",
		"namespace":         "team-vision",
		"tenantName":        "team-vision",
		"tenantDisplayName": "Vision Team",
		"computeNamespace":  "axisml-team-vision",
		"name":              "resnet-train-7",
		"jobName":           "resnet-train",
		"runNumber":         7,
		"displayName":       "ResNet-50 Training #7",
		"description":       "Distributed ResNet-50 training run on ImageNet.",
		"owner":             "li.wei",
		"ownerId":           "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"backend":           backend,
		"poolName":          "gpu-a100",
		"unitName":          "a100-2x",
		"resources":         obj{"cpu": "32", "memory": "256Gi", "nvidia.com/gpu": "8"},
		"roles":             []any{roleStatus},
		"runPolicy":         runPolicy,
		"spec":              mlRunSpec,
		"phase":             "Running",
		"message":           "All worker replicas ready.",
		"scheduledAt":       exStartedAt,
		"startedAt":         exStartedAt,
		"createdAt":         exStartedAt,
		"updatedAt":         exUpdatedAt,
	}
	g.SetExample("Run", run)
	g.SetExample("RunList", obj{
		"items":         []any{run},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("RunTriggerRequest", obj{
		"displayName": "ResNet-50 Training #8",
		"poolName":    "gpu-a100",
		"unitName":    "a100-2x",
		"roles": []any{obj{
			"name": "worker",
			"args": []any{"--epochs", "120"},
		}},
	})
}
