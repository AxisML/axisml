import dayjs from "dayjs";

// Shared date / duration formatters. Detail pages previously each defined their
// own copies (with subtly different `:ss` / signature variants); this is the
// single source so timestamps render identically across the app.

/** "YYYY-MM-DD HH:mm" (minute precision) or "—" when absent. */
export function fmtDateTime(v?: string | null): string {
  return v ? dayjs(v).format("YYYY-MM-DD HH:mm") : "—";
}

/** "YYYY-MM-DD HH:mm:ss" (second precision) or "—" when absent. */
export function fmtDateTimeSec(v?: string | null): string {
  return v ? dayjs(v).format("YYYY-MM-DD HH:mm:ss") : "—";
}

/** Humanize a second count into a compact h/m/s string. */
export function fmtDuration(secs: number): string {
  if (!Number.isFinite(secs) || secs < 0) return "—";
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  const s = Math.floor(secs % 60);
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

/** Duration between two ISO timestamps (now if `end` absent). "—" if no start. */
export function fmtRange(start?: string | null, end?: string | null): string {
  if (!start) return "—";
  const secs = (end ? dayjs(end) : dayjs()).diff(dayjs(start), "second");
  return secs >= 0 ? fmtDuration(secs) : "—";
}

/** Humanize a byte count into a binary-unit string (B/KiB/MiB/GiB/TiB). */
export function fmtBytes(n?: number): string {
  if (!n || n <= 0) return "—";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(1)} ${units[i]}`;
}

// Kubernetes quantity suffix → byte multiplier (binary Ki.. and decimal K..).
const QTY_MULT: Record<string, number> = {
  "": 1, Ki: 1024, Mi: 1024 ** 2, Gi: 1024 ** 3, Ti: 1024 ** 4, Pi: 1024 ** 5,
  K: 1e3, M: 1e6, G: 1e9, T: 1e12, P: 1e15,
};

/** Parse a Kubernetes quantity (e.g. "2Ti", "200Gi", "500G") to bytes; 0 when unparseable. */
export function parseQty(s?: string): number {
  if (!s) return 0;
  const m = /^([0-9.]+)\s*([A-Za-z]*)$/.exec(s.trim());
  if (!m) return 0;
  return parseFloat(m[1]) * (QTY_MULT[m[2]] ?? 1);
}
