import { Link } from "react-router-dom";

// Last-5-runs status strip (.runbar) from the list pages. Each cell is one of
// ok / fail / pend / run / none, oldest→newest left to right.
export type RunState = "ok" | "fail" | "pend" | "run" | "none";

const GLYPH: Record<RunState, string> = {
  ok: '<circle cx="12" cy="12" r="9"/><path d="M8.5 12.5l2.5 2.5 4.5-5"/>',
  fail: '<circle cx="12" cy="12" r="9"/><path d="M15 9l-6 6M9 9l6 6"/>',
  pend: '<circle cx="12" cy="12" r="9"/><path d="M12 8v4l3 2"/>',
  run: '<path d="M21 12a9 9 0 1 1-6.22-8.56"/>',
  none: '<circle cx="12" cy="12" r="9" stroke-dasharray="3 3"/>',
};

export function RunBar({ states, to, label }: { states: RunState[]; to: string; label?: string }) {
  return (
    <Link className="runbar" to={to} aria-label={label} title={label}>
      {states.map((s, i) => (
        <span key={i} className={"ri " + s}>
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeLinecap="round"
            strokeLinejoin="round"
            dangerouslySetInnerHTML={{ __html: GLYPH[s] }}
          />
        </span>
      ))}
    </Link>
  );
}
