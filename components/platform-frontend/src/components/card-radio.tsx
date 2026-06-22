import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

// Controlled radio-card group — the prototype's `.pick-grid`. Each option is a
// clickable card; the selected one gets an accent ring. Drives a simple
// `value` / `onChange` contract so it slots into form controllers. An optional
// `icon` renders to the left of the text (the prototype's image picker layout).
export interface CardOption {
  value: string;
  title: string;
  desc?: string;
  icon?: ReactNode;
}

export function CardRadio({
  options,
  value,
  onChange,
  columns = 2,
  disabled,
}: {
  options: CardOption[];
  value?: string;
  onChange?: (value: string) => void;
  columns?: number;
  disabled?: boolean;
}) {
  return (
    <div className="grid gap-2.5" style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}>
      {options.map((o) => {
        const selected = o.value === value;
        return (
          <button
            type="button"
            key={o.value}
            disabled={disabled}
            onClick={() => onChange?.(o.value)}
            className={cn(
              "rounded-lg border px-3 py-2.5 text-left transition-colors outline-none",
              "focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50",
              "disabled:cursor-not-allowed disabled:opacity-60",
              o.icon && "flex items-center gap-2.5",
              selected
                ? "border-foreground bg-accent ring-1 ring-foreground"
                : "border-border bg-card hover:border-foreground/30",
            )}
          >
            {o.icon && <span className="grid size-[22px] shrink-0 place-items-center">{o.icon}</span>}
            <span className="min-w-0">
              <span className="block truncate font-mono text-sm font-medium">{o.title}</span>
              {o.desc && <span className="mt-0.5 block truncate text-xs text-muted-foreground">{o.desc}</span>}
            </span>
          </button>
        );
      })}
    </div>
  );
}
