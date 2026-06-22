// Single source of truth for how a backend phase / status enum maps to a visual
// tone. Both the workload PhaseTag and the artifact (model/image) version rows
// read from here so a status never renders two different colours across pages.
export type PhaseTone = "running" | "success" | "pending" | "failed" | "stopped";

// Phase/status enum value → tone. The enum values themselves are machine-readable
// and never translated; only their display labels are (via the `phase.*` i18n
// catalog). Covers workload phases plus the artifact statuses Ready/Uploading/Failed.
const PHASE_TONE: Record<string, PhaseTone> = {
  Running: "running",
  Starting: "running",
  Creating: "running",
  Stopping: "running",
  Deleting: "running",
  Uploading: "pending",
  Ready: "success",
  Succeeded: "success",
  Completed: "success",
  Active: "success",
  Pending: "pending",
  Degraded: "pending",
  Suspended: "pending",
  Failed: "failed",
  Error: "failed",
  Stopped: "stopped",
  Deleted: "stopped",
};

export function phaseTone(phase?: string | null): PhaseTone {
  return phase ? (PHASE_TONE[phase] ?? "stopped") : "stopped";
}

export const TONE_TEXT: Record<PhaseTone, string> = {
  running: "text-success",
  success: "text-foreground",
  pending: "text-warning",
  failed: "text-destructive",
  stopped: "text-muted-foreground",
};

export const TONE_DOT: Record<PhaseTone, string> = {
  running: "bg-success",
  success: "bg-success",
  pending: "bg-warning",
  failed: "bg-destructive",
  stopped: "bg-muted-foreground",
};
