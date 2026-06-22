import { type ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

// A monospace secondary badge — the recurring "mono value chip" used for images,
// resource units, env vars, and artifact refs across the detail panes. Replaces
// the several ad-hoc `Badge variant="secondary" font-mono` / raw-span variants
// with one consistent token.
export function MonoChip({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <Badge variant="secondary" className={cn("font-mono", className)}>
      {children}
    </Badge>
  );
}
