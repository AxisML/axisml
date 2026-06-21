import { Tag } from "antd";
import { useTranslation } from "react-i18next";

// Maps a backend phase/status enum to a colored AntD Tag with a localized label.
// The enum value itself is never translated (it's machine-readable); only the
// display label is localized via the shared `phase.*` catalog. Reused across all
// list/detail pages that surface workload state.
const COLOR: Record<string, string> = {
  Running: "processing",
  Starting: "processing",
  Creating: "processing",
  Pending: "default",
  Ready: "success",
  Succeeded: "success",
  Completed: "success",
  Degraded: "warning",
  Stopping: "warning",
  Stopped: "default",
  Deleted: "default",
  Deleting: "default",
  Failed: "error",
};

export function PhaseTag({ phase }: { phase?: string | null }) {
  const { t } = useTranslation();
  if (!phase) return <span className="text-muted">—</span>;
  const label = t(`phase.${phase}`, { defaultValue: phase });
  return (
    <Tag color={COLOR[phase] ?? "default"} className="!m-0">
      {label}
    </Tag>
  );
}
