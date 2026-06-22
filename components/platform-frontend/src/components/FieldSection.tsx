import type { ReactNode } from "react";

// Numbered form section header (the prototype's `.fieldset-title`): a solid ink
// circular badge with the step number, followed by the bold section title.
export function FieldSection({ n, title, children }: { n: number; title: ReactNode; children?: ReactNode }) {
  return (
    <div className="mt-7 mb-4 flex items-center gap-2.5 first:mt-0">
      <span className="grid size-[22px] shrink-0 place-items-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
        {n}
      </span>
      <span className="text-[15px] font-semibold">{title}</span>
      {children}
    </div>
  );
}
