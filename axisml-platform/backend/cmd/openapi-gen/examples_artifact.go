package main

import "github.com/axisml/axisml/pkg/openapigen"

// exArtifact holds whole-object examples for the artifact.go DTOs
// (Platform-owned Model / Image definitions + their version upload handshake).
func exArtifact(g *openapigen.Generator) {
	// ---- Models ----
	model := obj{
		"id":          "c4a7f1e9-3d2b-4a6c-8e1f-9b0d5a2c7f31",
		"namespace":   "team-vision",
		"tenantName":  "team-vision",
		"name":        "resnet50",
		"version":     "1.4.0",
		"displayName": "ResNet-50 v1.4.0",
		"description": "ResNet-50 weights fine-tuned on ImageNet.",
		"status":      "Ready",
		"source":      "webUpload",
		"digest":      "sha256:9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a",
		"spec": obj{
			"framework":  "pytorch",
			"task":       "image-classification",
			"format":     "safetensors",
			"parameters": "25.6M",
		},
		"owner":     "li.wei",
		"ownerId":   "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"uri":       "oci://registry.axisml.io/team-vision/resnet50:1.4.0",
		"sizeBytes": 102457600,
		"createdAt": exCreatedAt,
		"readyAt":   exUpdatedAt,
		"updatedAt": exUpdatedAt,
	}
	g.SetExample("Model", model)
	g.SetExample("ModelList", obj{
		"items":         []any{model},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("ModelInitiateRequest", obj{
		"version":     "1.4.0",
		"displayName": "ResNet-50 v1.4.0",
		"description": "ResNet-50 weights fine-tuned on ImageNet.",
		"source":      "webUpload",
	})
	g.SetExample("ModelInitiateResponse", obj{
		"id":          "c4a7f1e9-3d2b-4a6c-8e1f-9b0d5a2c7f31",
		"uri":         "oci://registry.axisml.io/team-vision/resnet50:1.4.0",
		"storageKind": "oci",
		"uploadCredentials": obj{
			"username": "upload-token",
			"password": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		"expiresAt": exUpdatedAt,
	})
	g.SetExample("ModelCompleteRequest", obj{
		"digest": "sha256:9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a",
	})

	// ---- Artifacts (shared) ----
	g.SetExample("ArtifactUpdateRequest", obj{
		"displayName": "ResNet-50 v1.4.0 (calibrated)",
		"description": "Updated version description.",
		"labels":      obj{"team": "vision"},
	})
	g.SetExample("ArtifactDefinitionCreateRequest", obj{
		"name":        "resnet50",
		"displayName": "ResNet-50",
		"description": "ResNet-50 image-classification model.",
		"labels":      obj{"team": "vision"},
	})
	g.SetExample("ArtifactDefinitionPatchRequest", obj{
		"displayName": "ResNet-50 (v2)",
		"description": "Updated definition description.",
	})

	definition := obj{
		"id":          "1f2e3d4c-5b6a-7980-abcd-ef0123456789",
		"namespace":   "team-vision",
		"tenantName":  "team-vision",
		"name":        "resnet50",
		"kind":        "model",
		"displayName": "ResNet-50",
		"description": "ResNet-50 image-classification model.",
		"owner":       "li.wei",
		"ownerId":     "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"labels":      obj{"team": "vision"},
		"annotations": obj{"git-commit": "8c1f4e2"},
		"spec": obj{
			"framework":  "pytorch",
			"task":       "image-classification",
			"format":     "safetensors",
			"parameters": "25.6M",
		},
		"versionCount":    3,
		"latestVersion":   "1.4.0",
		"latestVersionAt": exUpdatedAt,
		"createdAt":       exCreatedAt,
		"updatedAt":       exUpdatedAt,
	}
	g.SetExample("ArtifactDefinition", definition)
	g.SetExample("ArtifactDefinitionList", obj{
		"items":         []any{definition},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("ArtifactResolveResponse", obj{
		"storageKind": "oci",
		"uri":         "oci://registry.axisml.io/team-vision/resnet50:1.4.0",
		"digest":      "sha256:9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a",
		"pullCredentials": obj{
			"username": "pull-token",
			"password": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		"expiresAt": exUpdatedAt,
	})

	// ---- Images ----
	image := obj{
		"id":          "a1b2c3d4-5e6f-7081-92a3-b4c5d6e7f809",
		"namespace":   "team-vision",
		"tenantName":  "team-vision",
		"name":        "pytorch-train",
		"version":     "2.3.0",
		"displayName": "PyTorch Training 2.3.0",
		"description": "Training image with PyTorch 2.3 and CUDA 12.",
		"status":      "Ready",
		"source":      "dockerPush",
		"digest":      "sha256:7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b",
		"spec": obj{
			"baseImage": "nvidia/cuda:12.1.0-runtime",
			"python":    "3.11",
			"purpose":   "training",
		},
		"owner":     "li.wei",
		"ownerId":   "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"uri":       "oci://registry.axisml.io/team-vision/pytorch-train:2.3.0",
		"sizeBytes": 5368709120,
		"createdAt": exCreatedAt,
		"readyAt":   exUpdatedAt,
		"updatedAt": exUpdatedAt,
	}
	g.SetExample("Image", image)
	g.SetExample("ImageList", obj{
		"items":         []any{image},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("ImageInitiateRequest", obj{
		"version":     "2.3.0",
		"displayName": "PyTorch Training 2.3.0",
		"description": "Training image with PyTorch 2.3 and CUDA 12.",
		"source":      "dockerPush",
		"spec":        obj{"purpose": "training"},
	})
	g.SetExample("ImageInitiateResponse", obj{
		"id":          "a1b2c3d4-5e6f-7081-92a3-b4c5d6e7f809",
		"uri":         "oci://registry.axisml.io/team-vision/pytorch-train:2.3.0",
		"storageKind": "oci",
		"uploadCredentials": obj{
			"username": "upload-token",
			"password": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		"expiresAt": exUpdatedAt,
	})
	g.SetExample("ImageCompleteRequest", obj{
		"digest": "sha256:7f3148e1f4a6c8e3d2b4a6c8e1f9b0d5a2c7f3148e1f4a6c8e3d2b4a6c8e1f9b",
	})
}
