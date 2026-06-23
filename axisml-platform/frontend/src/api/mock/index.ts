// Entry point for the in-browser mock API. When VITE_USE_MOCK_API=true the
// generated client's `fetch` is swapped for `mockFetch` (see api/client.ts), so
// the frontend serves itself entirely from src/api/mock/data.ts with zero
// backend traffic.
import { route } from "./router";

export const USE_MOCK = import.meta.env.VITE_USE_MOCK_API === "true";

// Simulated latency so loading states still render like the real thing.
const LATENCY_MS = 250;
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Drop-in replacement for window.fetch, matching the shape the @hey-api client
// invokes it with: a Request object, returning a Response. Bodies are JSON
// except pod-log endpoints, which return text/plain.
export async function mockFetch(input: Request | string | URL, init?: RequestInit): Promise<Response> {
  const request = input instanceof Request ? input : new Request(input, init);
  const url = new URL(request.url, window.location.origin);
  const method = request.method.toUpperCase();

  let body: unknown;
  if (method === "POST" || method === "PATCH" || method === "PUT") {
    try {
      body = await request.clone().json();
    } catch {
      body = undefined;
    }
  }

  const result = route(method, url.pathname, url.searchParams, body);
  await sleep(LATENCY_MS);

  const status = result.status ?? 200;
  if (status === 204) return new Response(null, { status: 204 });

  const isText = typeof result.body === "string";
  return new Response(isText ? (result.body as string) : JSON.stringify(result.body ?? {}), {
    status,
    headers: { "Content-Type": isText ? "text/plain" : "application/json" },
  });
}

if (USE_MOCK) {
  console.info("[axisml] VITE_USE_MOCK_API=true — serving API from in-browser mock; backend is not contacted.");
}
