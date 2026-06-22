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
