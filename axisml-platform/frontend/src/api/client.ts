import type { CreateClientConfig } from "./generated/client.gen";
import { USE_MOCK, mockFetch } from "./mock";

// Runtime configuration injected into the generated @hey-api/client-fetch client.
// Same-origin in production (served behind the Envoy Gateway); the Vite dev server
// proxies /api to the backend.
//
// The bearer JWT is attached by a request interceptor (see api/setup.ts) rather
// than the `auth` resolver, because the platform OpenAPI spec declares no security
// scheme on its operations — without one the resolver never fires.
//
// When VITE_USE_MOCK_API=true we swap in `mockFetch`, which resolves every call
// from the in-browser fixtures (src/api/mock) — the frontend then never touches
// the network. The client's request/response interceptors still run, but they're
// no-ops against the mock (no real token / no 401).
export const createClientConfig: CreateClientConfig = (config) => ({
  ...config,
  baseUrl: "",
  credentials: "include",
  ...(USE_MOCK ? { fetch: mockFetch } : {}),
});
