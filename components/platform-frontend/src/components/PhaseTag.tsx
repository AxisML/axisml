import { useTranslation } from "react-i18next";

// Maps a backend phase/status enum to the prototype's dot + label `.status`
// indicator (a small coloured dot followed by a localized label) rather than a
// filled tag. The enum value itself is never translated (it's machine-readable);
// only the display label is localized via the shared `phase.*` catalog. Reused
// across every list/detail page that surfaces workload state.
//
// tone drives the text + dot colour; `pulse` adds the breathing ring used for
// live/running states, matching the prototype's `.status-running .dot`.
type Tone = "running" | "success" | "pending" | "failed" | "stopped";

const TONE: Record<string, Tone> = {
  Running: "running",
  Starting: "running",
  Creating: "running",
  Stopping: "running",
  Deleting: "running",
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

const TEXT: Record<Tone, string> = {
  running: "text-success",
  success: "text-fg-2",
  pending: "text-warn",
  failed: "text-danger",
  stopped: "text-muted",
};

const DOT: Record<Tone, string> = {
  running: "bg-success",
  success: "bg-success",
  pending: "bg-warn",
  failed: "bg-danger",
  stopped: "bg-muted",
};

export function PhaseTag({ phase }: { phase?: string | null }) {
  const { t } = useTranslation();
  if (!phase) return <span className="text-muted">—</span>;
  const tone = TONE[phase] ?? "stopped";
  const label = t(`phase.${phase}`, { defaultValue: phase });
  return (
    <span className={`inline-flex items-center gap-1.5 whitespace-nowrap text-xs font-medium ${TEXT[tone]}`}>
      <span className={`h-[7px] w-[7px] shrink-0 rounded-full ${DOT[tone]} ${tone === "running" ? "status-pulse" : ""}`} />
      {label}
    </span>
  );
}
