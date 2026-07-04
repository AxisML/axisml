import type { ResourceMap, ResourceUnit } from "@/api/generated";

// Format a resource unit's requests into a one-line spec, e.g.
// "1×GPU · 8 vCPU · 64 GiB". Shared by the resource-pool manager and the
// training / workspace create drawers so units read consistently everywhere.

const GiB = 1024 ** 3;
const MEM_SUFFIX: Record<string, number> = {
  Ki: 1024, Mi: 1024 ** 2, Gi: 1024 ** 3, Ti: 1024 ** 4, Pi: 1024 ** 5,
  k: 1e3, M: 1e6, G: 1e9, T: 1e12, P: 1e15,
};

function scalar(m: ResourceMap | undefined, k: string): number | undefined {
  const v = m?.[k];
  if (v == null) return undefined;
  const n = parseFloat(String(v));
  return Number.isFinite(n) ? n : undefined;
}

// Parse a Kubernetes memory quantity (with binary/decimal suffix) into GiB.
export function memGiB(m: ResourceMap | undefined, k: string): number | undefined {
  const v = m?.[k];
  if (v == null) return undefined;
  const match = String(v).trim().match(/^([0-9.]+)\s*([A-Za-z]+)?$/);
  if (!match) return undefined;
  const n = parseFloat(match[1]);
  if (!Number.isFinite(n)) return undefined;
  const bytes = match[2] ? n * (MEM_SUFFIX[match[2]] ?? GiB) : n;
  return Math.round((bytes / GiB) * 1000) / 1000;
}

export function unitSpecLine(u: ResourceUnit, gpuLabel = "GPU"): string {
  const cpu = scalar(u.requests, "cpu");
  const mem = memGiB(u.requests, "memory");
  const gpu = scalar(u.requests, "nvidia.com/gpu");
  const parts: string[] = [];
  if (gpu) parts.push(`${gpu}×${gpuLabel}`);
  if (cpu != null) parts.push(`${cpu} vCPU`);
  if (mem != null) parts.push(`${mem} GiB`);
  return parts.join(" · ") || "—";
}
