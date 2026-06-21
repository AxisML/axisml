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
  // Active-tenant scope is carried by the `axisml.tenant` cookie (the backend
  // reads it cookie-first, header-fallback). Sync the cookie from the selected
  // tenant here — synchronous, so it's present on this very request regardless of
  // React timing. There is no "all-tenants" view: exactly one tenant is selected.
  const tenant = localStorage.getItem(TENANT_KEY);
  if (tenant && tenant !== "all") {
    document.cookie = `${TENANT_KEY}=${tenant}; path=/; SameSite=Lax`;
  }
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
