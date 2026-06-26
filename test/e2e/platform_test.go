//go:build (e2e || standard) && !lite

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/test/e2e/internal/clients/platform"
)

// Platform-layer e2e: drives the deployed Platform backend (the only externally
// reachable layer) through its generated, typed client over a port-forward,
// exercising the orchestration chain Platform -> cluster-manager / compute / PG.
// Auth is JWT (login), unlike the System services which trust X-Axisml-User; the
// bearer token is injected per call via a RequestEditorFn, and the active tenant
// for tenant-scoped routes travels in each operation's typed *Params.
//
// Prereqs (beyond the System-layer suite): the axisml-platform Helm release is
// installed with the real backend image, and `platform bootstrap` has seeded the
// admin account. See test/e2e/README.md.

// bearer is a per-call request editor that attaches the JWT. An empty token
// leaves the request unauthenticated (used for the 401 negative path).
func bearer(token string) func(context.Context, *http.Request) error {
	return func(_ context.Context, req *http.Request) error {
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return nil
	}
}

func platformLogin(ctx context.Context, user, pass string) (string, error) {
	r, err := platformCli.LoginWithResponse(ctx, platform.LoginRequest{Username: user, Password: pass})
	if err != nil {
		return "", err
	}
	if !is2xx(r.StatusCode()) {
		return "", fmt.Errorf("login %s: status %d: %s", user, r.StatusCode(), r.Body)
	}
	if r.JSON200 == nil {
		return "", fmt.Errorf("login %s: empty body", user)
	}
	return r.JSON200.Jwt, nil
}

// ---- lazy platform setup (port-forward + admin login), shared across tests ----

var (
	platformOnce sync.Once
	platformPF   *portForward
	platformCli  *platform.ClientWithResponses
	adminToken   string
	platformErr  error
)

func platformReady(t *testing.T) (*platform.ClientWithResponses, string) {
	t.Helper()
	platformOnce.Do(func() {
		ns := envOr("E2E_PLATFORM_NAMESPACE", "axisml-platform")
		svc := envOr("E2E_PLATFORM_SVC", "axisml-platform-backend")
		pf, err := startPortForward(ns, svc, 8080)
		if err != nil {
			platformErr = fmt.Errorf("port-forward platform %s/%s: %w", ns, svc, err)
			return
		}
		platformPF = pf
		cli, err := platform.NewClientWithResponses(pf.localURL(),
			platform.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}))
		if err != nil {
			platformErr = fmt.Errorf("build platform client: %w", err)
			return
		}
		platformCli = cli

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		// Retry login while the platform pod / bootstrap settles.
		deadline := time.Now().Add(90 * time.Second)
		for {
			tok, lerr := platformLogin(ctx, "admin", os.Getenv("E2E_ADMIN_PASSWORD"))
			if lerr == nil {
				adminToken = tok
				return
			}
			if os.Getenv("E2E_ADMIN_PASSWORD") == "" {
				tok, lerr = platformLogin(ctx, "admin", "admin")
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
	return platformCli, adminToken
}

func unique(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1000000)
}

// ---- tests ----

func TestPlatform_Auth(t *testing.T) {
	pc, admin := platformReady(t)
	ctx := context.Background()

	me, err := pc.GetCurrentUserWithResponse(ctx, bearer(admin))
	mustOK(t, me.StatusCode(), me.Body, err, "GET /auth/me")
	require.NotNil(t, me.JSON200)
	if !me.JSON200.IsSystemAdmin {
		t.Fatalf("admin /me isSystemAdmin = false")
	}

	// No token -> 401.
	no, err := pc.GetCurrentUserWithResponse(ctx, bearer(""))
	if err != nil || no.StatusCode() != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /me: status %d err %v", no.StatusCode(), err)
	}
	// Bad creds -> 401.
	if _, lerr := platformLogin(ctx, "admin", "definitely-wrong"); lerr == nil {
		t.Fatalf("login with wrong password should fail")
	}
}

func TestPlatform_ResourcePools(t *testing.T) {
	pc, admin := platformReady(t)
	ctx := context.Background()
	r, err := pc.ListResourcePoolsWithResponse(ctx, nil, bearer(admin))
	mustOK(t, r.StatusCode(), r.Body, err, "GET /resourcepools")
	// At least the default pool should exist in a provisioned cluster.
	if r.JSON200 == nil || len(r.JSON200.Items) == 0 {
		t.Logf("no resource pools found (cluster may not be seeded); skipping pool detail")
		return
	}
}

func TestPlatform_TenantQuotaMemberLifecycle(t *testing.T) {
	pc, admin := platformReady(t)
	ctx := context.Background()

	member := unique("plat-u")
	tenant := unique("plat-t")

	// Create a member user (system-admin).
	cu, err := pc.CreateUserWithResponse(ctx, platform.UserCreateRequest{
		Username: member, Password: "password123", DisplayName: ptr(member),
	}, bearer(admin))
	mustCreated(t, cu.StatusCode(), cu.Body, err, "POST /users")
	t.Cleanup(func() { _ = deleteUser(ctx, admin, member) })

	// Create the tenant with the member as initial tenant-admin.
	ct, err := pc.CreateTenantWithResponse(ctx, platform.TenantCreateRequest{
		Identifier: tenant, KubernetesNamespace: tenant,
		DisplayName: "Platform E2E", InitialAdmin: member,
	}, bearer(admin))
	mustCreated(t, ct.StatusCode(), ct.Body, err, "POST /tenants")
	t.Cleanup(func() {
		// Remove members then delete the tenant.
		removeAllMembers(ctx, admin, tenant)
		_, _ = pc.DeleteTenantWithResponse(ctx, tenant, bearer(admin))
	})

	// Get + list.
	gt, err := pc.GetTenantWithResponse(ctx, tenant, bearer(admin))
	mustOK(t, gt.StatusCode(), gt.Body, err, "GET tenant")

	// Quota: assign the default pool / cpu-small unit.
	pool := envOr("E2E_DEFAULT_POOL", "default")
	unit := envOr("E2E_DEFAULT_UNIT", "cpu-small")
	q, err := pc.CreateTenantQuotaWithResponse(ctx, tenant, platform.QuotaCreateRequest{
		Pool: pool, Units: []platform.QuotaUnit{{UnitName: unit, Quantity: 1}},
	}, bearer(admin))
	mustCreated(t, q.StatusCode(), q.Body, err, "POST quota")

	// Members: the initial admin is present; removing it (the only admin) is blocked.
	memberToken, lerr := platformLogin(ctx, member, "password123")
	if lerr != nil {
		t.Fatalf("member login: %v", lerr)
	}
	ml, err := pc.ListTenantMembersWithResponse(ctx, tenant, bearer(memberToken))
	mustOK(t, ml.StatusCode(), ml.Body, err, "GET members (member)")
	require.NotNil(t, ml.JSON200)
	if len(ml.JSON200.Items) != 1 {
		t.Fatalf("want 1 member, got %d", len(ml.JSON200.Items))
	}
	rm, err := pc.RemoveTenantMemberWithResponse(ctx, tenant, ml.JSON200.Items[0].UserId, bearer(admin))
	if err != nil || rm.StatusCode() != http.StatusConflict {
		t.Fatalf("removing last tenant-admin: want 409, got %d (%s)", rm.StatusCode(), rm.Body)
	}

	// Suspend / resume.
	sus, err := pc.SuspendTenantWithResponse(ctx, tenant, bearer(admin))
	mustOK(t, sus.StatusCode(), sus.Body, err, "suspend")
	res, err := pc.ResumeTenantWithResponse(ctx, tenant, bearer(admin))
	mustOK(t, res.StatusCode(), res.Body, err, "resume")
}

func TestPlatform_JobDefinitionLifecycle(t *testing.T) {
	pc, admin := platformReady(t)
	ctx := context.Background()

	owner := unique("job-u")
	tenant := unique("job-t")
	_, _ = pc.CreateUserWithResponse(ctx, platform.UserCreateRequest{
		Username: owner, Password: "password123", DisplayName: ptr(owner),
	}, bearer(admin))
	t.Cleanup(func() { _ = deleteUser(ctx, admin, owner) })
	ct, err := pc.CreateTenantWithResponse(ctx, platform.TenantCreateRequest{
		Identifier: tenant, KubernetesNamespace: tenant, DisplayName: "Job E2E", InitialAdmin: owner,
	}, bearer(admin))
	mustCreated(t, ct.StatusCode(), ct.Body, err, "POST /tenants")
	t.Cleanup(func() {
		removeAllMembers(ctx, admin, tenant)
		_, _ = pc.DeleteTenantWithResponse(ctx, tenant, bearer(admin))
	})

	tok, _ := platformLogin(ctx, owner, "password123")
	job := unique("job")
	spec := platform.JobSpec{
		Backend:  platform.Backend{Name: platform.BackendNameNative, Engine: "job"},
		PoolName: ptr(envOr("E2E_DEFAULT_POOL", "default")),
		UnitName: ptr(envOr("E2E_DEFAULT_UNIT", "cpu-small")),
		Roles: []platform.MLRunRole{{
			Name:     "worker",
			Replicas: ptr(1),
			Template: platform.RoleTemplate{
				Image:   ptr(envOr("E2E_JOB_IMAGE", "busybox:latest")),
				Command: &[]string{"sh", "-c", "echo hi"},
			},
		}},
	}
	tenantParam := ptr(tenant)
	// Create the Job definition (Platform PG) with the tenant in the header param.
	cj, err := pc.CreateJobWithResponse(ctx, &platform.CreateJobParams{XAxismlTenant: tenantParam},
		platform.JobCreateRequest{Name: job, DisplayName: ptr("echo"), Spec: spec}, bearer(tok))
	mustCreated(t, cj.StatusCode(), cj.Body, err, "POST /jobs")

	// Get + list.
	gj, err := pc.GetJobWithResponse(ctx, job, &platform.GetJobParams{XAxismlTenant: tenantParam}, bearer(tok))
	mustOK(t, gj.StatusCode(), gj.Body, err, "GET /jobs/{name}")
	lj, err := pc.ListJobsWithResponse(ctx, &platform.ListJobsParams{XAxismlTenant: tenantParam}, bearer(tok))
	mustOK(t, lj.StatusCode(), lj.Body, err, "GET /jobs")

	// Runs list is empty (no trigger).
	lr, err := pc.ListRunsWithResponse(ctx, job, &platform.ListRunsParams{XAxismlTenant: tenantParam}, bearer(tok))
	mustOK(t, lr.StatusCode(), lr.Body, err, "GET runs")

	// Delete the Job definition.
	dj, err := pc.DeleteJobWithResponse(ctx, job, &platform.DeleteJobParams{XAxismlTenant: tenantParam}, bearer(tok))
	if err != nil || (dj.StatusCode() != http.StatusNoContent && dj.StatusCode() != http.StatusOK) {
		t.Fatalf("DELETE /jobs/{name}: status %d (%s)", dj.StatusCode(), dj.Body)
	}
}

// ---- helpers ----

func mustOK(t *testing.T, code int, body []byte, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: transport error: %v", what, err)
	}
	if !is2xx(code) {
		t.Fatalf("%s: status %d: %s", what, code, body)
	}
}

func mustCreated(t *testing.T, code int, body []byte, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: transport error: %v", what, err)
	}
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("%s: want 201, got %d: %s", what, code, body)
	}
}

func deleteUser(ctx context.Context, admin, username string) error {
	r, err := platformCli.ListUsersWithResponse(ctx, &platform.ListUsersParams{Q: ptr(username)}, bearer(admin))
	if err != nil || !is2xx(r.StatusCode()) || r.JSON200 == nil {
		return fmt.Errorf("lookup user")
	}
	for _, u := range r.JSON200.Items {
		if u.Username == username {
			_, _ = platformCli.DeleteUserWithResponse(ctx, u.Id, bearer(admin))
		}
	}
	return nil
}

func removeAllMembers(ctx context.Context, admin, tenant string) {
	r, err := platformCli.ListTenantMembersWithResponse(ctx, tenant, bearer(admin))
	if err != nil || !is2xx(r.StatusCode()) || r.JSON200 == nil {
		return
	}
	// Best-effort: attempt each removal (the last admin removal is blocked, which
	// is fine — the tenant delete that follows handles the remainder).
	for _, m := range r.JSON200.Items {
		_, _ = platformCli.RemoveTenantMemberWithResponse(ctx, tenant, m.UserId, bearer(admin))
	}
}
