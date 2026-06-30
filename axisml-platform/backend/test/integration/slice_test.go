//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// do drives the in-process engine and returns status + decoded JSON body.
func do(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	testEngine.ServeHTTP(rec, req)
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// tokenCache memoises one session token per credential pair across the whole
// suite. /auth/login is rate-limited per client IP (burst 10), and httptest
// makes every request share one IP, so re-authenticating on every call would
// exhaust the bucket and 429. Reusing the session token also mirrors real
// clients, which log in once and carry the token.
var (
	tokenMu    sync.Mutex
	tokenCache = map[string]string{}
)

func loginAdmin(t *testing.T) string {
	t.Helper()
	return loginAs(t, "admin", "admin")
}

func TestProbes(t *testing.T) {
	code, _ := do(t, http.MethodGet, "/healthz", "", nil)
	require.Equal(t, http.StatusOK, code)
}

func TestAuthFlow(t *testing.T) {
	tok := loginAdmin(t)

	// /me reflects the bootstrapped system-admin.
	code, me := do(t, http.MethodGet, "/api/v1/auth/me", tok, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, true, me["isSystemAdmin"])

	// Unauthenticated access is rejected.
	code, _ = do(t, http.MethodGet, "/api/v1/auth/me", "", nil)
	require.Equal(t, http.StatusUnauthorized, code)

	// Bad credentials.
	code, _ = do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{"username": "admin", "password": "wrong"})
	require.Equal(t, http.StatusUnauthorized, code)
}

func TestTenantAndMemberLifecycle(t *testing.T) {
	admin := loginAdmin(t)

	// Seed two member users.
	for _, u := range []string{"alice", "bob"} {
		code, body := do(t, http.MethodPost, "/api/v1/users", admin, map[string]any{
			"username": u, "password": "password123", "displayName": u,
		})
		require.Equal(t, http.StatusCreated, code, "create user %s: %v", u, body)
	}

	// Non-admin cannot create a tenant.
	aliceTok := loginAs(t, "alice", "password123")
	code, _ := do(t, http.MethodPost, "/api/v1/tenants", aliceTok, map[string]any{
		"identifier": "acme-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Acme", "initialAdmin": "alice",
	})
	require.Equal(t, http.StatusForbidden, code)

	// system-admin creates a tenant with alice as the initial tenant-admin.
	code, tn := do(t, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": "acme-team", "kubernetesNamespace": "axisml-tenant",
		"displayName": "Acme", "initialAdmin": "alice",
	})
	require.Equal(t, http.StatusCreated, code, "create tenant: %v", tn)
	require.Equal(t, "acme-team", tn["identifier"])

	// Get + list.
	code, _ = do(t, http.MethodGet, "/api/v1/tenants/acme-team", admin, nil)
	require.Equal(t, http.StatusOK, code)
	code, list := do(t, http.MethodGet, "/api/v1/tenants", admin, nil)
	require.Equal(t, http.StatusOK, code)
	require.GreaterOrEqual(t, int(list["count"].(float64)), 2) // default + acme-team

	// alice (tenant-admin) lists members and adds bob.
	code, members := do(t, http.MethodGet, "/api/v1/tenants/acme-team/members", aliceTok, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 1, int(members["count"].(float64)))

	code, _ = do(t, http.MethodPost, "/api/v1/tenants/acme-team/members", aliceTok, map[string]any{
		"account": "bob", "roleName": "user",
	})
	require.Equal(t, http.StatusCreated, code)

	// bob (role=user) cannot manage members.
	bobTok := loginAs(t, "bob", "password123")
	code, _ = do(t, http.MethodPost, "/api/v1/tenants/acme-team/members", bobTok, map[string]any{
		"account": "admin", "roleName": "user",
	})
	require.Equal(t, http.StatusForbidden, code)

	// Removing the only tenant-admin (alice) is blocked.
	aliceID := userID(t, admin, "alice")
	code, body := do(t, http.MethodDelete, "/api/v1/tenants/acme-team/members/"+aliceID, admin, nil)
	require.Equal(t, http.StatusConflict, code)
	require.Equal(t, "last-tenant-admin", body["code"])

	// Suspend / resume gate.
	code, susp := do(t, http.MethodPost, "/api/v1/tenants/acme-team/suspend", admin, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, true, susp["suspended"])
	require.Equal(t, "Suspended", susp["phase"])
	code, res := do(t, http.MethodPost, "/api/v1/tenants/acme-team/resume", admin, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, false, res["suspended"])

	// Quota set + list (system-admin).
	code, _ = do(t, http.MethodPost, "/api/v1/tenants/acme-team/quotas", admin, map[string]any{
		"pool": "default", "units": []map[string]any{{"unitName": "gpu-small", "quantity": 2}},
	})
	require.Equal(t, http.StatusCreated, code)
	code, quotas := do(t, http.MethodGet, "/api/v1/tenants/acme-team/quotas", admin, nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 1, int(quotas["count"].(float64)))
}

func loginAs(t *testing.T, user, pass string) string {
	t.Helper()
	key := user + "\x00" + pass
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if tok, ok := tokenCache[key]; ok {
		return tok
	}
	code, body := do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]any{"username": user, "password": pass})
	require.Equal(t, http.StatusOK, code, "login %s: %v", user, body)
	tok, _ := body["jwt"].(string)
	require.NotEmpty(t, tok)
	tokenCache[key] = tok
	return tok
}

func userID(t *testing.T, admin, username string) string {
	t.Helper()
	code, body := do(t, http.MethodGet, "/api/v1/users?q="+username, admin, nil)
	require.Equal(t, http.StatusOK, code)
	items, _ := body["items"].([]any)
	for _, it := range items {
		m := it.(map[string]any)
		if m["username"] == username {
			return m["id"].(string)
		}
	}
	t.Fatalf("user %s not found", username)
	return ""
}
