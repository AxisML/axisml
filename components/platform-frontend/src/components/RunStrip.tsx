import { Link } from "react-router-dom";

// Recent-run status strip — the prototype's `.runbar`: up to 5 small circular
// glyphs (✓ success / ✗ failure / ◔ running / dashed = no run) showing a job or
// experiment's latest run outcomes, oldest→newest, padded on the left with empty
// slots. Optionally links through to the detail page's run history.
//
// Run roll-ups aren't carried by the list endpoints, so pages pass an empty
// `phases` outside mock mode and the strip renders the inert dashed state.
type Slot = "ok" | "fail" | "run" | "pend" | "none";
const SLOTS = 5;

function toSlot(phase?: string): Slot {
  switch (phase) {
    case "Succeeded":
    case "Completed":
    case "Ready":
      return "ok";
    case "Failed":
    case "Error":
      return "fail";
    case "Running":
    case "Starting":
      return "run";
    case "Pending":
      return "pend";
    default:
      return "none";
  }
}

const COLOR: Record<Slot, string> = {
  ok: "text-success",
  fail: "text-destructive",
  run: "text-info",
  pend: "text-warning",
  none: "text-border",
};

function Glyph({ slot }: { slot: Slot }) {
  const common = { viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeLinecap: "round" as const, strokeLinejoin: "round" as const, strokeWidth: 2, className: "h-5 w-5" };
  if (slot === "run") {
    return (
      <svg {...common} className="h-5 w-5 animate-spin">
        <path d="M21 12a9 9 0 1 1-6.22-8.56" />
      </svg>
    );
  }
  return (
    <svg {...common}>
      <circle cx="12" cy="12" r="9" strokeDasharray={slot === "none" ? "3 3" : undefined} />
      {slot === "ok" && <path d="M8.5 12.5l2.5 2.5 4.5-5" />}
      {slot === "fail" && <path d="M15 9l-6 6M9 9l6 6" />}
      {slot === "pend" && <path d="M12 8v4l2 2" />}
    </svg>
  );
}

export function RunStrip({ phases = [], to }: { phases?: string[]; to?: string }) {
  // Right-align the newest runs; pad the left with empty slots up to SLOTS.
  const recent = phases.slice(-SLOTS);
  const slots: Slot[] = [
    ...Array.from({ length: Math.max(0, SLOTS - recent.length) }, () => "none" as Slot),
    ...recent.map(toSlot),
  ];
  const inner = (
    <span className="-mx-1.5 -my-1 inline-flex items-center gap-1.5 rounded-sm px-1.5 py-1 transition-colors hover:bg-muted">
      {slots.map((s, i) => (
        <span key={i} className={`inline-grid h-5 w-5 place-items-center ${COLOR[s]}`}>
          <Glyph slot={s} />
        </span>
      ))}
    </span>
  );
  return to ? <Link to={to} aria-label="最近运行">{inner}</Link> : inner;
}
