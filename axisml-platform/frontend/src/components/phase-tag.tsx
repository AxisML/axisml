import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { phaseTone, TONE_DOT, TONE_TEXT, type PhaseTone } from "@/lib/phase";
import { cn } from "@/lib/utils";

// A small coloured dot followed by a label — the prototype's `.status` indicator.
// `tone` drives the dot + text colour; `pulse` adds the breathing ring used for
// live/in-progress states (running, uploading). This is the shared primitive
// behind both PhaseTag (workload phases) and artifact version-status rows.
export function StatusDot({
  tone,
  children,
  pulse,
  className,
}: {
  tone: PhaseTone;
  children: ReactNode;
  pulse?: boolean;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 text-xs font-medium whitespace-nowrap",
        TONE_TEXT[tone],
        className,
      )}
    >
      <span
        className={cn(
          "size-[7px] shrink-0 rounded-full",
          TONE_DOT[tone],
          (pulse ?? tone === "running") && "status-pulse",
        )}
      />
      {children}
    </span>
  );
}

// Maps a backend phase enum to the StatusDot indicator. The enum value is never
// translated (it's machine-readable); only the display label is localized via the
// shared `phase.*` catalog. Reused across every list/detail page surfacing state.
export function PhaseTag({ phase }: { phase?: string | null }) {
  const { t } = useTranslation();
  if (!phase) return <span className="text-muted-foreground">—</span>;
  const label = t(`phase.${phase}`, { defaultValue: phase });
  return <StatusDot tone={phaseTone(phase)}>{label}</StatusDot>;
}
