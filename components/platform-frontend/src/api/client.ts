import type { CreateClientConfig } from "./generated/client.gen";

// Runtime configuration injected into the generated @hey-api/client-fetch client.
// Same-origin in production (served behind the Envoy Gateway); the Vite dev server
// proxies /api to the backend.
//
// The bearer JWT is attached by a request interceptor (see api/setup.ts) rather
// than the `auth` resolver, because the platform OpenAPI spec declares no security
// scheme on its operations — without one the resolver never fires.
export const createClientConfig: CreateClientConfig = (config) => ({
  ...config,
  baseUrl: "",
  credentials: "include",
});
