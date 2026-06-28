package main

import "github.com/axisml/axisml/pkg/openapigen"

// exTenant holds whole-object examples for the tenant.go DTOs.
func exTenant(g *openapigen.Generator) {
	quotaUnit := obj{"unitName": "a100-2x", "quantity": 4}
	quota := obj{
		"pool":  "gpu-a100",
		"units": []any{quotaUnit},
	}
	g.SetExample("QuotaUnit", quotaUnit)
	g.SetExample("Quota", quota)

	quotaUnitStatus := obj{"unitName": "a100-2x", "quantity": 4, "used": 3}
	quotaStatus := obj{
		"pool":  "gpu-a100",
		"units": []any{quotaUnitStatus},
	}
	g.SetExample("QuotaUnitStatus", quotaUnitStatus)
	g.SetExample("QuotaStatus", quotaStatus)
	g.SetExample("QuotaList", obj{
		"items":    []any{quota},
		"statuses": []any{quotaStatus},
		"count":    1,
	})
	g.SetExample("QuotaCreateRequest", obj{
		"pool":  "gpu-a100",
		"units": []any{quotaUnit},
	})
	g.SetExample("QuotaPatchRequest", obj{
		"units": []any{obj{"unitName": "a100-2x", "quantity": 6}},
	})

	secretSourceRef := obj{"namespace": "axisml-system", "name": "registry-pull"}
	configMapSourceRef := obj{"namespace": "axisml-system", "name": "shared-config"}
	g.SetExample("SecretSourceRef", secretSourceRef)
	g.SetExample("ConfigMapSourceRef", configMapSourceRef)

	imagePullSecretInit := obj{
		"name":            "registry-pull",
		"sourceSecretRef": secretSourceRef,
	}
	secretInit := obj{
		"name":            "wandb-api-key",
		"type":            "Opaque",
		"sourceSecretRef": obj{"namespace": "axisml-system", "name": "wandb-api-key"},
	}
	configMapInit := obj{
		"name":               "shared-config",
		"sourceConfigMapRef": configMapSourceRef,
	}
	serviceAccountInit := obj{
		"name":             "trainer",
		"imagePullSecrets": []any{"registry-pull"},
	}
	g.SetExample("ImagePullSecretInit", imagePullSecretInit)
	g.SetExample("SecretInit", secretInit)
	g.SetExample("ConfigMapInit", configMapInit)
	g.SetExample("ServiceAccountInit", serviceAccountInit)

	initResources := obj{
		"imagePullSecrets": []any{imagePullSecretInit},
		"secrets":          []any{secretInit},
		"configMaps":       []any{configMapInit},
		"serviceAccounts":  []any{serviceAccountInit},
	}
	g.SetExample("InitResources", initResources)

	tenantStatus := obj{
		"message": "Tenant is active.",
		"conditions": []any{obj{
			"type":               "Ready",
			"status":             "True",
			"reason":             "Provisioned",
			"message":            "Namespace and quota provisioned.",
			"lastTransitionTime": exUpdatedAt,
		}},
		"quotas": []any{quotaStatus},
	}
	g.SetExample("TenantStatus", tenantStatus)

	tenant := obj{
		"identifier":           "team-vision",
		"kubernetesNamespace":  "axisml-team-vision",
		"displayName":          "Vision Team",
		"description":          "Computer-vision model training and inference team.",
		"owner":                "li.wei",
		"labels":               obj{"team": "vision"},
		"quotas":               []any{quota},
		"initResources":        initResources,
		"phase":                "Active",
		"status":               tenantStatus,
		"suspended":            false,
		"memberCount":          8,
		"activeJobRuns":        3,
		"activeExperimentRuns": 2,
		"onlineServices":       1,
		"createdAt":            exCreatedAt,
		"updatedAt":            exUpdatedAt,
	}
	g.SetExample("Tenant", tenant)
	g.SetExample("TenantList", obj{
		"items":         []any{tenant},
		"count":         1,
		"continueToken": "",
		"partial":       false,
	})
	g.SetExample("TenantCreateRequest", obj{
		"identifier":          "team-vision",
		"kubernetesNamespace": "axisml-team-vision",
		"displayName":         "Vision Team",
		"description":         "Computer-vision model training and inference team.",
		"initialAdmin":        "li.wei@example.com",
		"labels":              obj{"team": "vision"},
		"quotas":              []any{quota},
	})
	g.SetExample("TenantPatchRequest", obj{
		"displayName": "Vision Team (core)",
		"description": "Updated team description.",
	})

	member := obj{
		"userId":      "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"username":    "li.wei",
		"displayName": "Li Wei",
		"email":       "li.wei@example.com",
		"roleName":    "tenant-admin",
		"addedAt":     exCreatedAt,
	}
	g.SetExample("Member", member)
	g.SetExample("MemberList", obj{
		"items":         []any{member},
		"count":         1,
		"continueToken": "",
	})
	g.SetExample("MemberCreateRequest", obj{
		"account":  "zhang.san@example.com",
		"roleName": "user",
	})
	g.SetExample("MemberPatchRequest", obj{
		"roleName": "tenant-admin",
	})
}
