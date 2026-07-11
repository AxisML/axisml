package main

import "github.com/axisml/axisml/pkg/openapigen"

// exWorkspace holds whole-object examples for the workspace.go DTOs.
func exWorkspace(g *openapigen.Generator) {
	lifecycle := obj{"idleTimeoutSeconds": 3600}
	g.SetExample("WorkspaceLifecycle", lifecycle)

	volume := obj{
		"name":      "notebook-data",
		"mountPath": "/home/jovyan/work",
		"used":      "12Gi",
	}
	g.SetExample("WorkspaceVolume", volume)

	tool := obj{"name": "jupyter", "label": "JupyterLab", "url": "https://axisml.example.com/ws/team-vision/notebook-dev/lab"}
	g.SetExample("WorkspaceTool", tool)

	endpoint := obj{
		"accessUrl":   "https://axisml.example.com/ws/team-vision/notebook-dev/",
		"internalDns": "notebook-dev.axisml-team-vision.svc.cluster.local",
		"tools": []any{
			tool,
			obj{"name": "vscode", "label": "VS Code", "url": "https://axisml.example.com/ws/team-vision/notebook-dev/vscode/"},
			obj{"name": "terminal", "label": "Terminal", "url": "https://axisml.example.com/ws/team-vision/notebook-dev/terminals/1"},
		},
	}
	g.SetExample("WorkspaceEndpoint", endpoint)

	workspace := obj{
		"id":                "f1e2d3c4-5b6a-4798-8c0d-1e2f3a4b5c6d",
		"namespace":         "team-vision",
		"tenantName":        "team-vision",
		"tenantDisplayName": "Vision Team",
		"computeNamespace":  "axisml-team-vision",
		"name":              "notebook-dev",
		"displayName":       "Vision team dev environment",
		"description":       "JupyterLab interactive development environment.",
		"owner":             "li.wei",
		"ownerId":           "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"image":             "registry.axisml.io/dev/jupyter:3.0.0",
		"command":           []any{"start-notebook.sh"},
		"args":              []any{"--NotebookApp.token="},
		"env":               []any{obj{"name": "JUPYTER_ENABLE_LAB", "value": "yes"}},
		"containerPort":     8888,
		"poolName":          "gpu-a100",
		"unitName":          "a100-1x",
		"quota":             "team-vision",
		"resources":         obj{"cpu": "4", "memory": "32Gi", "nvidia.com/gpu": "1"},
		"volumes":           []any{volume},
		"lifecycle":         lifecycle,
		"replicas":          1,
		"readyReplicas":     1,
		"desiredState":      "Running",
		"phase":             "Running",
		"message":           "Workspace is ready.",
		"endpoint":          endpoint,
		"lastStartedAt":     exStartedAt,
		"createdAt":         exCreatedAt,
		"updatedAt":         exUpdatedAt,
	}
	g.SetExample("Workspace", workspace)
	g.SetExample("WorkspaceList", obj{
		"items":         []any{workspace},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})

	g.SetExample("WorkspaceCreateRequest", obj{
		"name":          "notebook-dev",
		"displayName":   "Vision team dev environment",
		"description":   "JupyterLab interactive development environment.",
		"image":         "registry.axisml.io/dev/jupyter:3.0.0",
		"containerPort": 8888,
		"poolName":      "gpu-a100",
		"unitName":      "a100-1x",
		"quota":         "team-vision",
		"volumes":       []any{obj{"name": "notebook-data", "mountPath": "/home/jovyan/work"}},
		"lifecycle":     lifecycle,
	})
	g.SetExample("WorkspacePatchRequest", obj{
		"displayName": "Vision team dev environment (v2)",
		"description": "Updated description.",
		"lifecycle":   obj{"idleTimeoutSeconds": 7200},
	})

	wsImage := obj{
		"ref":         "registry.axisml.io/dev/jupyter:3.0.0",
		"displayName": "JupyterLab",
		"description": "JupyterLab data-science environment (public).",
		"kind":        "jupyter",
		"defaultPort": 8888,
		"public":      true,
	}
	g.SetExample("WorkspaceImage", wsImage)
	g.SetExample("WorkspaceImageList", obj{
		"items": []any{
			wsImage,
			obj{"ref": "registry.axisml.io/dev/pytorch:2.3.0", "displayName": "PyTorch", "description": "CUDA 12.1 + PyTorch 2.3 training image.", "kind": "jupyter", "defaultPort": 8888, "public": true},
			obj{"ref": "registry.axisml.io/dev/code-server:4.22", "displayName": "VS Code", "description": "VS Code (code-server) development environment (public).", "kind": "vscode", "defaultPort": 8080, "public": true},
		},
		"count": 3,
	})
}
