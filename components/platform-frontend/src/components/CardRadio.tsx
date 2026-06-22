import { cn } from "@/lib/utils";

// Controlled radio-card group — the prototype's `.pick-grid`. Each option is a
// clickable card; the selected one gets an accent ring. Drives a simple
// `value` / `onChange` contract so it slots into form controllers.
export interface CardOption {
  value: string;
  title: string;
  desc?: string;
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
              selected
                ? "border-foreground bg-accent ring-1 ring-foreground"
                : "border-border bg-card hover:border-foreground/30",
            )}
          >
            <div className="font-mono text-sm font-medium">{o.title}</div>
            {o.desc && <div className="mt-0.5 text-xs text-muted-foreground">{o.desc}</div>}
          </button>
        );
      })}
    </div>
  );
}
