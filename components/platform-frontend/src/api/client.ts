import type { CreateClientConfig } from "./generated/client.gen";

// Runtime configuration injected into the generated @hey-api/client-fetch client.
// Same-origin in production (served behind the Envoy Gateway); the Vite dev server
// proxies /api to the backend.
//
// Auth is cookie + JWT only — the frontend does not derive or send any tenant
// scope. `credentials: "include"` carries the session cookie; when a bearer JWT
// is also held client-side it is attached via the `auth` resolver.
export const createClientConfig: CreateClientConfig = (config) => ({
  ...config,
  baseUrl: "",
  credentials: "include",
  auth: () => localStorage.getItem("axisml.token") ?? undefined,
});
