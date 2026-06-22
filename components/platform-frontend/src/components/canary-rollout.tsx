import { useTranslation } from "react-i18next";
import { Slider } from "@/components/ui/slider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

// Standard canary ramp steps: a few quick presets plus 100% for a manual full cutover.
const CANARY_PRESETS = [5, 10, 25, 50, 100] as const;

/**
 * Visual canary rollout control: quick ramp presets above a plain slider that sets
 * the canary share. Shared by the traffic detail "config" tab and the row-level
 * adjust drawer.
 */
export function CanaryRollout({
  value,
  onChange,
  className,
}: {
  /** Canary percent being edited. */
  value: number;
  onChange: (v: number) => void;
  className?: string;
}) {
  const { t } = useTranslation();

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {/* ramp presets */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="mr-1 text-xs text-muted-foreground">{t("traffic.canaryPresets")}</span>
        {CANARY_PRESETS.map((p) => (
          <Button
            key={p}
            type="button"
            variant={value === p ? "secondary" : "outline"}
            size="xs"
            className="font-mono"
            onClick={() => onChange(p)}
          >
            {p}%
          </Button>
        ))}
      </div>

      {/* plain slider + editable percentage input */}
      <div className="flex items-center gap-4">
        <Slider
          min={0}
          max={100}
          step={1}
          value={[value]}
          onValueChange={(v) => onChange(v[0])}
          className="flex-1"
        />
        <div className="relative w-20 shrink-0">
          <Input
            type="number"
            min={0}
            max={100}
            value={value}
            onChange={(e) => {
              const n = Number(e.target.value);
              onChange(Number.isNaN(n) ? 0 : Math.max(0, Math.min(100, Math.round(n))));
            }}
            className="pr-7 text-right font-mono tabular-nums"
          />
          <span className="pointer-events-none absolute top-1/2 right-2.5 -translate-y-1/2 text-sm text-muted-foreground">
            %
          </span>
        </div>
      </div>
    </div>
  );
}
