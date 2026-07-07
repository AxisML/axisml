package apidoc

import "github.com/axisml/axisml/pkg/openapigen"

// obj is a terse alias for building whole-object example values.
type obj = map[string]any

// registerExamples attaches a realistic, internally-consistent example to every
// registered schema. The narrative mirrors the platform spec for cross-spec
// consistency: a "resnet50" model artifact version "1.4.0" in the "team-vision"
// namespace, registered through the two-phase upload handshake (Initiate hands
// back an upload URI + credentials; Complete finalizes with a content digest).
func registerExamples(g *openapigen.Generator) {
	const (
		exNamespace = "team-vision"
		exName      = "resnet50"
		exVersion   = "1.4.0"
		exDigest    = "sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10"
		exReadyAt   = "2026-06-28T09:30:00Z"
		exCreatedAt = "2026-06-20T08:00:00Z"
		exUpdatedAt = "2026-06-28T09:30:00Z"
		exExpiresAt = "2026-06-28T10:30:00Z"
	)

	g.SetExample("ArtifactHubError", obj{
		"type":     "https://axisml.io/errors/not_found",
		"title":    "artifact not found",
		"status":   404,
		"detail":   "artifact team-vision/resnet50@1.4.0 not found",
		"instance": "/api/v1/namespaces/team-vision/artifacts/resnet50/1.4.0",
		"code":     "not_found",
	})

	artifactSpec := obj{
		"framework":  "pytorch",
		"task":       "image-classification",
		"format":     "safetensors",
		"parameters": "25.6M",
	}

	g.SetExample("ArtifactInitiateRequest", obj{
		"version":     exVersion,
		"spec":        artifactSpec,
		"source":      "webUpload",
		"visibility":  "tenant",
		"displayName": "ResNet-50",
		"description": "ResNet-50 image-classification model pretrained on ImageNet.",
		"labels":      obj{"team": "vision", "stage": "production"},
		"annotations": obj{"git-commit": "8c1f4e2"},
	})

	credentials := obj{
		"username":   "team-vision",
		"password":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.upload-token",
		"expires_at": exExpiresAt,
	}

	g.SetExample("ArtifactInitiateResponse", obj{
		"artifact": obj{
			"id":          "8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10",
			"namespace":   exNamespace,
			"kind":        "model",
			"name":        exName,
			"version":     exVersion,
			"visibility":  "tenant",
			"displayName": "ResNet-50",
			"description": "ResNet-50 image-classification model pretrained on ImageNet.",
			"labels":      obj{"team": "vision", "stage": "production"},
			"owner":       "li.wei",
			"spec":        artifactSpec,
			"status":      "Uploading",
			"source":      "webUpload",
			"createdAt":   exCreatedAt,
			"updatedAt":   exCreatedAt,
		},
		"upload": obj{
			"storageKind": "oci",
			"uri":         "oci://registry.axisml.io/team-vision/resnet50:1.4.0",
			"credentials": credentials,
		},
	})

	g.SetExample("ArtifactCompleteRequest", obj{
		"digest": exDigest,
	})

	g.SetExample("ArtifactPatchRequest", obj{
		"displayName": "ResNet-50 (production)",
		"description": "Updated description.",
		"labels":      obj{"team": "vision", "stage": "production"},
		"annotations": obj{"reviewed-by": "zhang.san"},
	})

	g.SetExample("ArtifactResolveResponse", obj{
		"storageKind":     "oci",
		"uri":             "oci://registry.axisml.io/team-vision/resnet50:1.4.0",
		"digest":          exDigest,
		"visibility":      "tenant",
		"pullCredentials": credentials,
	})

	artifact := obj{
		"id":          "8c1f4e22-2b7a-4c0e-9b1d-7f3a2e5c9a10",
		"namespace":   exNamespace,
		"kind":        "model",
		"name":        exName,
		"version":     exVersion,
		"visibility":  "tenant",
		"displayName": "ResNet-50",
		"description": "ResNet-50 image-classification model pretrained on ImageNet.",
		"labels":      obj{"team": "vision", "stage": "production"},
		"annotations": obj{"git-commit": "8c1f4e2"},
		"owner":       "li.wei",
		"spec":        artifactSpec,
		"status":      "Ready",
		"source":      "webUpload",
		"digest":      exDigest,
		"readyAt":     exReadyAt,
		"createdAt":   exCreatedAt,
		"updatedAt":   exUpdatedAt,
	}
	g.SetExample("Artifact", artifact)

	// ArtifactList ({items, total}) is registered in Document() after this call,
	// so it can reuse the Artifact item example above.
	g.SetExample("ArtifactList", obj{"items": []any{artifact}, "total": 1})

	g.SetExample("Capabilities", obj{
		"kinds":  []any{"model", "dataset", "image"},
		"upload": true,
	})
}
