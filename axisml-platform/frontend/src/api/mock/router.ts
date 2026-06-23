// Mock request router for VITE_USE_MOCK_API mode. It is installed as the custom
// `fetch` of the generated @hey-api/client-fetch client (see api/client.ts), so
// every SDK call resolves here without any network I/O. The router returns real
// Response objects; the client wrapper handles parsing + the {data,error} unwrap
// exactly as it would for a live backend.
import * as db from "./data";

interface Result {
  status?: number;
  body?: unknown;
}

// A route is a method + path-pattern (":param" segments) + handler.
type Handler = (params: Record<string, string>, query: URLSearchParams, body: unknown) => Result;
interface Route {
  method: string;
  pattern: string;
  handler: Handler;
}

const routes: Route[] = [];
const on = (method: string, pattern: string, handler: Handler) =>
  routes.push({ method, pattern: `/api/v1${pattern}`, handler });

const list = (items: unknown[]): Result => ({ body: { count: items.length, items } });
const ok = (body: unknown = {}): Result => ({ body });
const created = (body: unknown): Result => ({ status: 201, body });
const noContent = (): Result => ({ status: 204 });

// ── auth ─────────────────────────────────────────────────────────────────────
on("POST", "/auth/login", () => ok({ jwt: "mock.jwt.token", expiresAt: db.ago(-24) }));
on("POST", "/auth/logout", () => noContent());
on("POST", "/auth/refresh", () => ok({ jwt: "mock.jwt.token", expiresAt: db.ago(-24) }));
on("GET", "/auth/me", () => ok(db.me));

// ── jobs + runs ───────────────────────────────────────────────────────────────
on("GET", "/jobs", () => list(db.jobs));
on("POST", "/jobs", (_p, _q, body) => created({ ...db.jobs[0], ...(body as object) }));
on("GET", "/jobs/:name", (p) => ok(db.jobs.find((j) => j.name === p.name) ?? db.jobs[0]));
on("PATCH", "/jobs/:name", (p, _q, body) => ok({ ...(db.jobs.find((j) => j.name === p.name) ?? db.jobs[0]), ...(body as object) }));
on("DELETE", "/jobs/:name", () => noContent());
on("GET", "/jobs/:name/runs", (p) => list(db.runsFor(p.name, "job")));
on("POST", "/jobs/:name/runs", (p) => created(db.runsFor(p.name, "job")[0]));
on("GET", "/jobs/:name/runs/:run", (p) => ok(db.runsFor(p.name, "job").find((r) => r.name === p.run) ?? db.runsFor(p.name, "job")[0]));
on("DELETE", "/jobs/:name/runs/:run", () => noContent());
on("POST", "/jobs/:name/runs/:run/cancel", (p) => ok(db.runsFor(p.name, "job")[0]));
on("GET", "/jobs/:name/runs/:run/events", (p) => list(db.eventsFor(p.run)));
on("GET", "/jobs/:name/runs/:run/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "gpu_util")));
on("GET", "/jobs/:name/runs/:run/pods", (p) => list(db.podsFor(p.run)));
on("GET", "/jobs/:name/runs/:run/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/jobs/:name/runs/:run/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));

// ── experiments + runs (mirror jobs) ────────────────────────────────────────────
on("GET", "/experiments", () => list(db.experiments));
on("POST", "/experiments", (_p, _q, body) => created({ ...db.experiments[0], ...(body as object) }));
on("GET", "/experiments/:name", (p) => ok(db.experiments.find((e) => e.name === p.name) ?? db.experiments[0]));
on("PATCH", "/experiments/:name", (p, _q, body) => ok({ ...(db.experiments.find((e) => e.name === p.name) ?? db.experiments[0]), ...(body as object) }));
on("DELETE", "/experiments/:name", () => noContent());
on("GET", "/experiments/:name/runs", (p) => list(db.runsFor(p.name, "experiment")));
on("POST", "/experiments/:name/runs", (p) => created(db.runsFor(p.name, "experiment")[0]));
on("GET", "/experiments/:name/runs/:run", (p) => ok(db.runsFor(p.name, "experiment").find((r) => r.name === p.run) ?? db.runsFor(p.name, "experiment")[0]));
on("DELETE", "/experiments/:name/runs/:run", () => noContent());
on("POST", "/experiments/:name/runs/:run/cancel", (p) => ok(db.runsFor(p.name, "experiment")[0]));
on("GET", "/experiments/:name/runs/:run/events", (p) => list(db.eventsFor(p.run)));
on("GET", "/experiments/:name/runs/:run/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "gpu_util")));
on("GET", "/experiments/:name/runs/:run/pods", (p) => list(db.podsFor(p.run)));
on("GET", "/experiments/:name/runs/:run/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/experiments/:name/runs/:run/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));
on("GET", "/experiments/:name/tensorboard", (p) => ok({ name: `tb-${p.name}`, phase: "Ready", url: `https://tb-${p.name}.llm-lab.axisml.io`, createdAt: db.ago(2) }));
on("POST", "/experiments/:name/tensorboard", (p) => ok({ name: `tb-${p.name}`, phase: "Ready", url: `https://tb-${p.name}.llm-lab.axisml.io`, createdAt: db.now() }));
on("DELETE", "/experiments/:name/tensorboard", () => noContent());

// ── workspaces ──────────────────────────────────────────────────────────────────
on("GET", "/workspaces", () => list(db.workspaces));
on("POST", "/workspaces", (_p, _q, body) => created({ ...db.workspaces[0], ...(body as object) }));
on("GET", "/workspaces/:name", (p) => ok(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]));
on("PATCH", "/workspaces/:name", (p, _q, body) => ok({ ...(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]), ...(body as object) }));
on("DELETE", "/workspaces/:name", () => noContent());
on("POST", "/workspaces/:name/start", (p) => ok({ ...(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]), phase: "Starting" }));
on("POST", "/workspaces/:name/stop", (p) => ok({ ...(db.workspaces.find((w) => w.name === p.name) ?? db.workspaces[0]), phase: "Stopped" }));
on("GET", "/workspaces/:name/events", (p) => list(db.eventsFor(p.name)));
on("GET", "/workspaces/:name/pods", (p) => list(db.podsFor(p.name).slice(0, 1)));
on("GET", "/workspaces/:name/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/workspaces/:name/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));

// ── services ────────────────────────────────────────────────────────────────────
on("GET", "/mlservices", () => list(db.services));
on("POST", "/mlservices", (_p, _q, body) => created({ ...db.services[0], ...(body as object) }));
on("GET", "/mlservices/:name", (p) => ok(db.services.find((s) => s.name === p.name) ?? db.services[0]));
on("PATCH", "/mlservices/:name", (p, _q, body) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), ...(body as object) }));
on("DELETE", "/mlservices/:name", () => noContent());
on("POST", "/mlservices/:name/scale", (p, _q, body) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), ...(body as object) }));
on("POST", "/mlservices/:name/start", (p) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), phase: "Ready" }));
on("POST", "/mlservices/:name/stop", (p) => ok({ ...(db.services.find((s) => s.name === p.name) ?? db.services[0]), phase: "Stopped" }));
on("GET", "/mlservices/:name/events", (p) => list(db.eventsFor(p.name)));
on("GET", "/mlservices/:name/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "request_rate")));
on("GET", "/mlservices/:name/pods", (p) => list(db.podsFor(p.name).slice(0, 2)));
on("GET", "/mlservices/:name/pods/:pod/events", (p) => list(db.eventsFor(p.pod)));
on("GET", "/mlservices/:name/pods/:pod/logs", (p) => ({ body: db.podLogs(p.pod) }));

// ── traffic policies ──────────────────────────────────────────────────────────
on("GET", "/trafficpolicies", () => list(db.trafficPolicies));
on("POST", "/trafficpolicies", (_p, _q, body) => created({ ...db.trafficPolicies[0], ...(body as object) }));
on("GET", "/trafficpolicies/:name", (p) => ok(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]));
on("PATCH", "/trafficpolicies/:name", (p, _q, body) => ok({ ...(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]), ...(body as object) }));
on("DELETE", "/trafficpolicies/:name", () => noContent());
on("GET", "/trafficpolicies/:name/events", (p) => list(db.eventsFor(p.name)));
on("GET", "/trafficpolicies/:name/metrics", (_p, q) => ok(db.metricSeries(q.get("metric") ?? "request_rate")));
on("POST", "/trafficpolicies/:name/promote", (p) => ok(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]));
on("POST", "/trafficpolicies/:name/rollback", (p) => ok(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]));
on("POST", "/trafficpolicies/:name/split", (p, _q, body) => ok({ ...(db.trafficPolicies.find((t) => t.name === p.name) ?? db.trafficPolicies[0]), ...(body as object) }));

// ── models ──────────────────────────────────────────────────────────────────────
on("GET", "/models", () => list(db.models));
on("GET", "/models/:tenant/:name", (p) => ok(db.models.find((m) => m.name === p.name) ?? db.models[0]));
on("POST", "/models/:tenant/:name", (_p, _q, body) => created({ ...db.models[0], ...(body as object) }));
on("PATCH", "/models/:tenant/:name", (p, _q, body) => ok({ ...(db.models.find((m) => m.name === p.name) ?? db.models[0]), ...(body as object) }));
on("DELETE", "/models/:tenant/:name", () => noContent());
on("GET", "/models/:tenant/:name/versions", (p) => list(db.modelVersions(p.name)));
on("POST", "/models/:tenant/:name/versions", (p, _q, body) => created({ id: `${p.name}-init`, uri: `s3://axisml-models/${p.tenant}/${p.name}`, storageKind: "s3", uploadCredentials: {}, ...(body as object) }));
on("GET", "/models/:tenant/:name/versions/:version", (p) => ok(db.modelVersions(p.name).find((v) => v.version === p.version) ?? db.modelVersions(p.name)[0]));
on("DELETE", "/models/:tenant/:name/versions/:version", () => noContent());
on("POST", "/models/:tenant/:name/versions/:version/complete", (p) => ok(db.modelVersions(p.name)[0]));
on("GET", "/models/:tenant/:name/versions/:version/resolve", (p) => ok({ uri: `s3://axisml-models/${p.tenant}/${p.name}/${p.version}`, expiresAt: db.ago(-1) }));

// ── images ────────────────────────────────────────────────────────────────────
on("GET", "/images", () => list(db.images));
on("GET", "/images/:tenant/:name", (p) => ok(db.images.find((m) => m.name === p.name) ?? db.images[0]));
on("POST", "/images/:tenant/:name", (_p, _q, body) => created({ ...db.images[0], ...(body as object) }));
on("PATCH", "/images/:tenant/:name", (p, _q, body) => ok({ ...(db.images.find((m) => m.name === p.name) ?? db.images[0]), ...(body as object) }));
on("DELETE", "/images/:tenant/:name", () => noContent());
on("GET", "/images/:tenant/:name/versions", (p) => list(db.imageVersions(p.name)));
on("POST", "/images/:tenant/:name/versions", (p, _q, body) => created({ id: `${p.name}-init`, uri: `harbor.axisml.io/${p.tenant}/${p.name}`, storageKind: "oci", uploadCredentials: {}, ...(body as object) }));
on("GET", "/images/:tenant/:name/versions/:version", (p) => ok(db.imageVersions(p.name).find((v) => v.version === p.version) ?? db.imageVersions(p.name)[0]));
on("DELETE", "/images/:tenant/:name/versions/:version", () => noContent());
on("POST", "/images/:tenant/:name/versions/:version/complete", (p) => ok(db.imageVersions(p.name)[0]));
on("GET", "/images/:tenant/:name/versions/:version/resolve", (p) => ok({ uri: `harbor.axisml.io/${p.tenant}/${p.name}:${p.version}`, expiresAt: db.ago(-1) }));

// ── resource pools + units ──────────────────────────────────────────────────────
on("GET", "/resourcepools", () => list(db.pools));
on("POST", "/resourcepools", (_p, _q, body) => created({ ...db.pools[0], ...(body as object) }));
on("GET", "/resourcepools/:pool", (p) => ok(db.pools.find((x) => x.name === p.pool) ?? db.pools[0]));
on("PATCH", "/resourcepools/:pool", (p, _q, body) => ok({ ...(db.pools.find((x) => x.name === p.pool) ?? db.pools[0]), ...(body as object) }));
on("DELETE", "/resourcepools/:pool", () => noContent());
on("GET", "/resourcepools/:pool/units", (p) => list(db.pools.find((x) => x.name === p.pool)?.units ?? []));
on("POST", "/resourcepools/:pool/units", (_p, _q, body) => created(body));
on("GET", "/resourcepools/:pool/units/:unit", (p) => ok((db.pools.find((x) => x.name === p.pool)?.units ?? []).find((u) => u.name === p.unit) ?? {}));
on("PATCH", "/resourcepools/:pool/units/:unit", (_p, _q, body) => ok(body));
on("DELETE", "/resourcepools/:pool/units/:unit", () => noContent());

// ── tenants, members, quotas ────────────────────────────────────────────────────
on("GET", "/tenants", () => list(db.tenants));
on("POST", "/tenants", (_p, _q, body) => created({ ...db.tenants[0], ...(body as object) }));
on("GET", "/tenants/:name", (p) => ok(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]));
on("PATCH", "/tenants/:name", (p, _q, body) => ok({ ...(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]), ...(body as object) }));
on("DELETE", "/tenants/:name", () => noContent());
on("POST", "/tenants/:name/suspend", (p) => ok({ ...(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]), suspended: true, phase: "Suspended" }));
on("POST", "/tenants/:name/resume", (p) => ok({ ...(db.tenants.find((t) => t.identifier === p.name) ?? db.tenants[0]), suspended: false, phase: "Active" }));
on("GET", "/tenants/:name/members", (p) => list(db.membersByTenant[p.name] ?? []));
on("POST", "/tenants/:name/members", (_p, _q, body) => created(body));
on("PATCH", "/tenants/:name/members/:userId", (_p, _q, body) => ok(body));
on("DELETE", "/tenants/:name/members/:userId", () => noContent());
on("GET", "/tenants/:name/quotas", (p) => ({ body: { count: (db.quotasByTenant[p.name] ?? []).length, items: db.quotasByTenant[p.name] ?? [] } }));
on("POST", "/tenants/:name/quotas", (_p, _q, body) => created(body));
on("PATCH", "/tenants/:name/quotas/:pool", (_p, _q, body) => ok(body));
on("DELETE", "/tenants/:name/quotas/:pool", () => noContent());

// ── health ──────────────────────────────────────────────────────────────────────
on("GET", "/healthz", () => ok({ status: "ok" }));
on("GET", "/readyz", () => ok({ status: "ok" }));

// ── matcher ─────────────────────────────────────────────────────────────────────
function match(routePattern: string, path: string): Record<string, string> | null {
  const rp = routePattern.split("/");
  const pp = path.split("/");
  if (rp.length !== pp.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < rp.length; i++) {
    if (rp[i].startsWith(":")) params[rp[i].slice(1)] = decodeURIComponent(pp[i]);
    else if (rp[i] !== pp[i]) return null;
  }
  return params;
}

export function route(method: string, path: string, query: URLSearchParams, body: unknown): Result {
  for (const r of routes) {
    if (r.method !== method) continue;
    const params = match(r.pattern, path);
    if (params) return r.handler(params, query, body);
  }
  // Unknown route: stay safe rather than hang. GETs look like an empty list;
  // writes succeed with an empty body.
  if (method === "GET") return { body: { count: 0, items: [] } };
  return { body: {} };
}
