import { type ReactNode } from "react";
import { cn } from "@/lib/utils";

// Key/value "descriptions" grid shared by the detail panes. Replaces the six
// near-identical local DescGrid/DescRow/Row helpers (which drifted on label-column
// width and alignment). A fixed 120px label column keeps every detail pane aligned.
//
// `columns="double"` packs two label/value pairs per row on md+ (the job/run/
// service info panes); `columns="single"` is one pair per row.
export function Descriptions({
  columns = "double",
  className,
  children,
}: {
  columns?: "single" | "double";
  className?: string;
  children: ReactNode;
}) {
  return (
    <dl
      className={cn(
        "grid grid-cols-[120px_1fr] gap-x-4 gap-y-3 text-sm",
        columns === "double" && "md:grid-cols-[120px_1fr_120px_1fr]",
        className,
      )}
    >
      {children}
    </dl>
  );
}

// One label/value pair. `span` makes it occupy a full row in a double grid.
export function Desc({ label, span, children }: { label: ReactNode; span?: boolean; children: ReactNode }) {
  return (
    <div className={span ? "contents md:col-span-4 md:grid md:grid-cols-[120px_1fr]" : "contents"}>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  );
}
