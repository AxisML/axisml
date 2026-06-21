// Best-effort human message out of a backend RFC 7807 Problem / Error / unknown
// throw. The backend's `detail`/`title` are already meaningful; callers that need
// localized copy map the problem `type`/`code` themselves (frontend.md §6).
export function errorText(err: unknown): string {
  if (!err) return "Unknown error";
  if (typeof err === "string") return err;
  const e = err as { detail?: string; title?: string; message?: string };
  return e.detail || e.title || e.message || "Request failed";
}
