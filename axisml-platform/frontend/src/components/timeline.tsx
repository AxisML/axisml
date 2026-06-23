import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

// Vertical event timeline mirroring the prototype's `.timeline` (run-detail 事件
// tab): a hairline rail with a colored dot per item, a reason/tag/time head and a
// description below. Replaces the flat events table so Run events read as a
// chronological feed. `tone` colors the dot; `tag` is the small NORMAL/WARNING chip.
export interface TimelineItem {
  id: string;
  name: ReactNode;
  tag?: ReactNode;
  tagTone?: "normal" | "warn";
  time?: ReactNode;
  desc?: ReactNode;
  tone?: "info" | "warn" | "muted";
}

const DOT_TONE: Record<NonNullable<TimelineItem["tone"]>, string> = {
  info: "bg-info",
  warn: "bg-warning",
  muted: "bg-muted-foreground",
};

export function Timeline({ items, className }: { items: TimelineItem[]; className?: string }) {
  return (
    <ol className={cn("relative pl-6", className)}>
      <span className="absolute top-3 bottom-3 left-[5px] w-0.5 bg-border" aria-hidden />
      {items.map((it) => (
        <li key={it.id} className="relative pb-6 last:pb-0">
          <span
            className={cn(
              "absolute top-1 -left-[23px] size-[11px] rounded-full ring-[3px] ring-background",
              DOT_TONE[it.tone ?? "info"],
            )}
            aria-hidden
          />
          <div className="flex flex-wrap items-center gap-2.5">
            <span className="text-sm font-semibold text-foreground">{it.name}</span>
            {it.tag != null && (
              <span
                className={cn(
                  "rounded-sm border px-1.5 py-px font-mono text-[10.5px] tracking-wider",
                  it.tagTone === "warn"
                    ? "border-transparent bg-warning/20 text-warning"
                    : "bg-muted text-muted-foreground",
                )}
              >
                {it.tag}
              </span>
            )}
            {it.time != null && (
              <span className="ml-auto font-mono text-xs text-muted-foreground">{it.time}</span>
            )}
          </div>
          {it.desc != null && (
            <div className="mt-1.5 text-[13px] text-muted-foreground">{it.desc}</div>
          )}
        </li>
      ))}
    </ol>
  );
}
