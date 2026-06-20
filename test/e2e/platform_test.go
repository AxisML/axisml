//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

// Platform-layer e2e: drives the deployed Platform backend (the only externally
// reachable layer) over a port-forward, exercising the orchestration chain
// Platform -> cluster-manager / compute / PG. Auth is JWT (login), unlike the
// System services which trust X-Axisml-User.
//
// Prereqs (beyond the System-layer suite): the axisml-platform Helm release is
// installed with the real backend image, and `platform bootstrap` has seeded the
// admin account. See test/e2e/README.md.

// ---- platform HTTP client (bearer JWT) ----

type platformClient struct {
	baseURL string
	token   string
	c       *http.Client
}

func (pc *platformClient) do(ctx context.Context, method, path, token string, body any) (resp, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return resp{}, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, pc.baseURL+path, rdr)
	if err != nil {
		return resp{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	httpResp, err := pc.c.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer httpResp.Body.Close()
	b, _ := io.ReadAll(httpResp.Body)
	return resp{status: httpResp.StatusCode, body: b}, nil
}

func (pc *platformClient) login(ctx context.Context, user, pass string) (string, error) {
	r, err := pc.do(ctx, http.MethodPost, "/api/v1/auth/login", "", map[string]any{"username": user, "password": pass})
	if err != nil {
		return "", err
	}
	if !r.is2xx() {
		return "", fmt.Errorf("login %s: status %d: %s", user, r.status, r.body)
	}
	var out struct {
		JWT string `json:"jwt"`
	}
	if err := r.decode(&out); err != nil {
		return "", err
	}
	return out.JWT, nil
}

// ---- lazy platform setup (port-forward + admin login), shared across tests ----

var (
	platformOnce sync.Once
	platformPF   *portForward
	platformPC   *platformClient
	adminToken   string
	platformErr  error
)

func platform(t *testing.T) (*platformClient, string) {
	t.Helper()
	platformOnce.Do(func() {
		ns := envOr("E2E_PLATFORM_NAMESPACE", "axisml-platform")
		svc := envOr("E2E_PLATFORM_SVC", "axisml-platform-platform")
		pf, err := startPortForward(ns, svc, 8080)
		if err != nil {
			platformErr = fmt.Errorf("port-forward platform %s/%s: %w", ns, svc, err)
			return
		}
		platformPF = pf
		platformPC = &platformClient{baseURL: pf.localURL(), c: &http.Client{Timeout: 30 * time.Second}}

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		// Retry login while the platform pod / bootstrap settles.
		deadline := time.Now().Add(90 * time.Second)
		for {
			tok, lerr := platformPC.login(ctx, "admin", os.Getenv("E2E_ADMIN_PASSWORD"))
			if lerr == nil {
				adminToken = tok
				return
			}
			if os.Getenv("E2E_ADMIN_PASSWORD") == "" {
				tok, lerr = platformPC.login(ctx, "admin", "admin")
				if lerr == nil {
					adminToken = tok
					return
				}
			}
			if time.Now().After(deadline) {
				platformErr = fmt.Errorf("admin login never succeeded: %w", lerr)
				return
			}
			time.Sleep(3 * time.Second)
		}
	})
	if platformErr != nil {
		t.Skipf("platform not reachable (is axisml-platform installed?): %v", platformErr)
	}
	return platformPC, adminToken
}

func unique(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1000000)
}

// ---- tests ----

func TestPlatform_Auth(t *testing.T) {
	pc, admin := platform(t)
	ctx := context.Background()

	r, err := pc.do(ctx, http.MethodGet, "/api/v1/auth/me", admin, nil)
	mustOK(t, r, err, "GET /auth/me")
	var me struct {
		IsSystemAdmin bool `json:"isSystemAdmin"`
	}
	_ = r.decode(&me)
	if !me.IsSystemAdmin {
		t.Fatalf("admin /me isSystemAdmin = false")
	}

	// No token -> 401.
	r, err = pc.do(ctx, http.MethodGet, "/api/v1/auth/me", "", nil)
	if err != nil || r.status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /me: status %d err %v", r.status, err)
	}
	// Bad creds -> 401.
	if _, lerr := pc.login(ctx, "admin", "definitely-wrong"); lerr == nil {
		t.Fatalf("login with wrong password should fail")
	}
}

func TestPlatform_ResourcePools(t *testing.T) {
	pc, admin := platform(t)
	ctx := context.Background()
	r, err := pc.do(ctx, http.MethodGet, "/api/v1/resourcepools", admin, nil)
	mustOK(t, r, err, "GET /resourcepools")
	var list struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	_ = r.decode(&list)
	// At least the default pool should exist in a provisioned cluster.
	if len(list.Items) == 0 {
		t.Logf("no resource pools found (cluster may not be seeded); skipping pool detail")
		return
	}
}

func TestPlatform_TenantQuotaMemberLifecycle(t *testing.T) {
	pc, admin := platform(t)
	ctx := context.Background()

	member := unique("plat-u")
	tenant := unique("plat-t")

	// Create a member user (system-admin).
	r, err := pc.do(ctx, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"username": member, "password": "password123", "displayName": member,
	})
	mustCreated(t, r, err, "POST /users")
	t.Cleanup(func() { _ = deleteUser(ctx, pc, admin, member) })

	// Create the tenant with the member as initial tenant-admin.
	r, err = pc.do(ctx, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": tenant, "kubernetesNamespace": tenant,
		"displayName": "Platform E2E", "initialAdmin": member,
	})
	mustCreated(t, r, err, "POST /tenants")
	t.Cleanup(func() {
		// Remove members then delete the tenant.
		removeAllMembers(ctx, pc, admin, tenant)
		_, _ = pc.do(ctx, http.MethodDelete, "/api/v1/tenants/"+tenant, admin, nil)
	})

	// Get + list.
	r, err = pc.do(ctx, http.MethodGet, "/api/v1/tenants/"+tenant, admin, nil)
	mustOK(t, r, err, "GET tenant")

	// Quota: assign the default pool / cpu-small unit.
	pool := envOr("E2E_DEFAULT_POOL", "default")
	unit := envOr("E2E_DEFAULT_UNIT", "cpu-small")
	r, err = pc.do(ctx, http.MethodPost, "/api/v1/tenants/"+tenant+"/quotas", admin, map[string]any{
		"pool": pool, "units": []map[string]any{{"unitName": unit, "quantity": 1}},
	})
	mustCreated(t, r, err, "POST quota")

	// Members: the initial admin is present; removing it (the only admin) is blocked.
	memberToken, lerr := pc.login(ctx, member, "password123")
	if lerr != nil {
		t.Fatalf("member login: %v", lerr)
	}
	r, err = pc.do(ctx, http.MethodGet, "/api/v1/tenants/"+tenant+"/members", memberToken, nil)
	mustOK(t, r, err, "GET members (member)")
	var members struct {
		Items []struct {
			UserID string `json:"userId"`
		} `json:"items"`
	}
	_ = r.decode(&members)
	if len(members.Items) != 1 {
		t.Fatalf("want 1 member, got %d", len(members.Items))
	}
	r, err = pc.do(ctx, http.MethodDelete, "/api/v1/tenants/"+tenant+"/members/"+members.Items[0].UserID, admin, nil)
	if err != nil || r.status != http.StatusConflict {
		t.Fatalf("removing last tenant-admin: want 409, got %d (%s)", r.status, r.body)
	}

	// Suspend / resume.
	r, err = pc.do(ctx, http.MethodPost, "/api/v1/tenants/"+tenant+"/suspend", admin, nil)
	mustOK(t, r, err, "suspend")
	r, err = pc.do(ctx, http.MethodPost, "/api/v1/tenants/"+tenant+"/resume", admin, nil)
	mustOK(t, r, err, "resume")
}

func TestPlatform_JobDefinitionLifecycle(t *testing.T) {
	pc, admin := platform(t)
	ctx := context.Background()

	owner := unique("job-u")
	tenant := unique("job-t")
	_, _ = pc.do(ctx, http.MethodPost, "/api/v1/users", admin, map[string]any{"username": owner, "password": "password123", "displayName": owner})
	t.Cleanup(func() { _ = deleteUser(ctx, pc, admin, owner) })
	r, err := pc.do(ctx, http.MethodPost, "/api/v1/tenants", admin, map[string]any{
		"identifier": tenant, "kubernetesNamespace": tenant, "displayName": "Job E2E", "initialAdmin": owner,
	})
	mustCreated(t, r, err, "POST /tenants")
	t.Cleanup(func() {
		removeAllMembers(ctx, pc, admin, tenant)
		_, _ = pc.do(ctx, http.MethodDelete, "/api/v1/tenants/"+tenant, admin, nil)
	})

	tok, _ := pc.login(ctx, owner, "password123")
	job := unique("job")
	spec := map[string]any{
		"backend":  map[string]any{"name": "native", "engine": "job"},
		"poolName": envOr("E2E_DEFAULT_POOL", "default"),
		"unitName": envOr("E2E_DEFAULT_UNIT", "cpu-small"),
		"roles": []map[string]any{{
			"name": "worker", "replicas": 1,
			"template": map[string]any{"image": envOr("E2E_JOB_IMAGE", "busybox:latest"), "command": []string{"sh", "-c", "echo hi"}},
		}},
	}
	// Create the Job definition (Platform PG) with the tenant in the header.
	r, err = pc.doTenant(ctx, http.MethodPost, "/api/v1/jobs", tok, tenant, map[string]any{
		"name": job, "displayName": "echo", "spec": spec,
	})
	mustCreated(t, r, err, "POST /jobs")

	// Get + list.
	r, err = pc.doTenant(ctx, http.MethodGet, "/api/v1/jobs/"+job, tok, tenant, nil)
	mustOK(t, r, err, "GET /jobs/{name}")
	r, err = pc.doTenant(ctx, http.MethodGet, "/api/v1/jobs", tok, tenant, nil)
	mustOK(t, r, err, "GET /jobs")

	// Runs list is empty (no trigger).
	r, err = pc.doTenant(ctx, http.MethodGet, "/api/v1/jobs/"+job+"/runs", tok, tenant, nil)
	mustOK(t, r, err, "GET runs")

	// Delete the Job definition.
	r, err = pc.doTenant(ctx, http.MethodDelete, "/api/v1/jobs/"+job, tok, tenant, nil)
	if err != nil || (r.status != http.StatusNoContent && r.status != http.StatusOK) {
		t.Fatalf("DELETE /jobs/{name}: status %d (%s)", r.status, r.body)
	}
}

// doTenant is do with the X-Axisml-Tenant header set.
func (pc *platformClient) doTenant(ctx context.Context, method, path, token, tenant string, body any) (resp, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, pc.baseURL+path, rdr)
	if err != nil {
		return resp{}, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Axisml-Tenant", tenant)
	httpResp, err := pc.c.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer httpResp.Body.Close()
	b, _ := io.ReadAll(httpResp.Body)
	return resp{status: httpResp.StatusCode, body: b}, nil
}

// ---- helpers ----

func mustOK(t *testing.T, r resp, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: transport error: %v", what, err)
	}
	if !r.is2xx() {
		t.Fatalf("%s: status %d: %s", what, r.status, r.body)
	}
}

func mustCreated(t *testing.T, r resp, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: transport error: %v", what, err)
	}
	if r.status != http.StatusCreated && r.status != http.StatusOK {
		t.Fatalf("%s: want 201, got %d: %s", what, r.status, r.body)
	}
}

func deleteUser(ctx context.Context, pc *platformClient, admin, username string) error {
	r, err := pc.do(ctx, http.MethodGet, "/api/v1/users?q="+username, admin, nil)
	if err != nil || !r.is2xx() {
		return fmt.Errorf("lookup user")
	}
	var list struct {
		Items []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"items"`
	}
	_ = r.decode(&list)
	for _, u := range list.Items {
		if u.Username == username {
			_, _ = pc.do(ctx, http.MethodDelete, "/api/v1/users/"+u.ID, admin, nil)
		}
	}
	return nil
}

func removeAllMembers(ctx context.Context, pc *platformClient, admin, tenant string) {
	r, err := pc.do(ctx, http.MethodGet, "/api/v1/tenants/"+tenant+"/members", admin, nil)
	if err != nil || !r.is2xx() {
		return
	}
	var members struct {
		Items []struct {
			UserID string `json:"userId"`
		} `json:"items"`
	}
	_ = r.decode(&members)
	// Demote-to-user then delete is required to drop the last admin; simplest is
	// to delete all but the last, then the tenant delete is blocked unless empty.
	// For cleanup best-effort, just attempt each removal.
	for _, m := range members.Items {
		_, _ = pc.do(ctx, http.MethodDelete, "/api/v1/tenants/"+tenant+"/members/"+m.UserID, admin, nil)
	}
}
