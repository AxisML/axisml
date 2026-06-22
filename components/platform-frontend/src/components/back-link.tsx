import { type ReactNode } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

// Consistent "back to list" link shown in a detail page's subtitle slot. Replaces
// the four divergent treatments the detail pages had (ghost Button / outline
// Button / bare links with mismatched icon sizes).
export function BackLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
    >
      <ArrowLeft className="size-3.5" />
      {children}
    </Link>
  );
}
