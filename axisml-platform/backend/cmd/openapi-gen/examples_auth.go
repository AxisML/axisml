package main

import "github.com/axisml/axisml/pkg/openapigen"

// exAuth holds whole-object examples for the auth.go DTOs.
func exAuth(g *openapigen.Generator) {
	user := obj{
		"id":                 "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"username":           "li.wei",
		"displayName":        "Li Wei",
		"email":              "li.wei@axisml.io",
		"disabled":           false,
		"mustChangePassword": false,
		"createdAt":          exCreatedAt,
		"updatedAt":          exUpdatedAt,
	}
	g.SetExample("User", user)

	tenantRoles := []any{
		obj{"tenantName": "team-vision", "roleName": "admin"},
		obj{"tenantName": "team-nlp", "roleName": "member"},
	}
	g.SetExample("UserTenantRole", obj{"tenantName": "team-vision", "roleName": "admin"})

	g.SetExample("LoginRequest", obj{
		"username": "li.wei",
		"password": "S3cure-pass",
	})

	g.SetExample("LoginResponse", obj{
		"jwt":         "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJsaS53ZWkifQ.sig",
		"expiresAt":   exFinishedAt,
		"user":        user,
		"tenantRoles": tenantRoles,
	})

	g.SetExample("RefreshResponse", obj{
		"jwt":       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJsaS53ZWkifQ.sig2",
		"expiresAt": exFinishedAt,
	})

	g.SetExample("MeResponse", obj{
		"user":          user,
		"tenantRoles":   tenantRoles,
		"permissions":   []any{"job:read", "job:write", "run:read", "run:trigger"},
		"isSystemAdmin": false,
	})

	userSummary := obj{
		"id":          "3a2b1c0d-4e5f-6789-abcd-ef0123456789",
		"username":    "li.wei",
		"displayName": "Li Wei",
		"email":       "li.wei@axisml.io",
	}
	g.SetExample("UserSummary", userSummary)
	g.SetExample("UserSummaryList", obj{
		"items":         []any{userSummary},
		"count":         1,
		"continueToken": "",
	})

	g.SetExample("UserCreateRequest", obj{
		"username":    "li.wei",
		"displayName": "Li Wei",
		"email":       "li.wei@axisml.io",
		"password":    "S3cure-pass",
	})

	g.SetExample("UserPatchRequest", obj{
		"displayName": "Li Wei (Vision Lead)",
		"email":       "li.wei@axisml.io",
		"disabled":    false,
	})

	g.SetExample("SetPasswordRequest", obj{
		"currentPassword": "S3cure-pass",
		"newPassword":     "S3cure-pass2",
	})
}
