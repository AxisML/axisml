import { client } from "./generated/client.gen";

// Wires auth onto the generated client. Imported once at app startup (main.tsx).
//
// The platform OpenAPI spec declares no per-operation security, so the generated
// SDK never invokes the `auth` resolver. We instead attach the bearer JWT via a
// request interceptor, and centralize session-expiry handling in a response
// interceptor (401 → drop the token and bounce to /login).

const TOKEN_KEY = "axisml.token";
const TENANT_KEY = "axisml.tenant";

client.interceptors.request.use((request) => {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token) request.headers.set("Authorization", `Bearer ${token}`);
  // Active-tenant scope for name-addressed endpoints. The backend reads
  // X-Axisml-Tenant to scope tenant-partitioned resources (services, workspaces,
  // traffic policies, jobs/experiments, their detail & mutations). There is no
  // "all-tenants" view — exactly one tenant is always selected.
  const tenant = localStorage.getItem(TENANT_KEY);
  if (tenant && tenant !== "all") request.headers.set("X-Axisml-Tenant", tenant);
  return request;
});

client.interceptors.response.use((response, request) => {
  // /auth/login itself legitimately 401s on bad credentials — let the caller
  // surface that inline rather than redirecting.
  const isLogin = request.url.endsWith("/api/v1/auth/login");
  if (response.status === 401 && !isLogin) {
    localStorage.removeItem(TOKEN_KEY);
    if (window.location.pathname !== "/login") {
      window.location.assign("/login");
    }
  }
  return response;
});
