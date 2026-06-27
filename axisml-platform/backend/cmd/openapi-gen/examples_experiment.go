package main

import "github.com/axisml/axisml/pkg/openapigen"

// exExperiment holds whole-object examples for the experiment.go DTOs
// (training-experiment templates + on-demand TensorBoard).
func exExperiment(g *openapigen.Generator) {
	backend := obj{"name": "kubeflow-trainer", "engine": "pytorchjob"}

	roleTemplate := obj{
		"image":     "registry.axisml.io/training/bert:2.1.0",
		"command":   []any{"python", "train.py"},
		"args":      []any{"--lr", "0.001", "--epochs", "10", "--batch-size", "64"},
		"env":       []any{obj{"name": "WANDB_MODE", "value": "offline"}},
		"resources": obj{"cpu": "8", "memory": "64Gi", "nvidia.com/gpu": "2"},
	}

	role := obj{
		"name":          "worker",
		"replicas":      2,
		"restartPolicy": "OnFailure",
		"template":      roleTemplate,
	}

	runPolicy := obj{
		"activeDeadlineSeconds":   86400,
		"ttlSecondsAfterFinished": 3600,
		"backoffLimit":            2,
	}

	jobSpec := obj{
		"backend":   backend,
		"poolName":  "gpu-a100",
		"unitName":  "a100-2x",
		"quota":     "team-nlp",
		"roles":     []any{role},
		"runPolicy": runPolicy,
		"artifacts": []any{obj{"kind": "model", "name": "bert-base", "version": "2.1.0"}},
	}

	experiment := obj{
		"id":          "d4f8a1b2-3c5e-4a7b-9c0d-1e2f3a4b5c6d",
		"namespace":   "team-nlp",
		"tenantName":  "team-nlp",
		"name":        "bert-finetune",
		"displayName": "BERT 微调实验",
		"description": "在中文语料上微调 BERT 的训练实验。",
		"owner":       "zhang.san",
		"ownerId":     "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"labels":      obj{"team": "nlp"},
		"spec":        jobSpec,
		"createdAt":   exCreatedAt,
		"updatedAt":   exUpdatedAt,
	}
	g.SetExample("Experiment", experiment)
	g.SetExample("ExperimentList", obj{
		"items":         []any{experiment},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("ExperimentCreateRequest", obj{
		"name":        "bert-finetune",
		"displayName": "BERT 微调实验",
		"description": "在中文语料上微调 BERT 的训练实验。",
		"labels":      obj{"team": "nlp"},
		"spec":        jobSpec,
	})
	g.SetExample("ExperimentPatchRequest", obj{
		"displayName": "BERT 微调实验 (v2)",
		"description": "更新后的描述。",
	})

	g.SetExample("TensorBoard", obj{
		"name":      "bert-finetune-tb",
		"url":       "https://tb.axisml.io/team-nlp/bert-finetune",
		"phase":     "Running",
		"message":   "TensorBoard is serving.",
		"createdAt": exCreatedAt,
	})
	g.SetExample("TensorBoardRequest", obj{
		"runs": []any{"bert-finetune-3", "bert-finetune-5"},
	})
}
