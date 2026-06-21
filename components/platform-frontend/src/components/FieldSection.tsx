import type { ReactNode } from "react";

// Numbered form section header (the prototype's `.fieldset-title`): a solid red
// circular badge with the step number, followed by the bold section title.
export function FieldSection({ n, title, children }: { n: number; title: ReactNode; children?: ReactNode }) {
  return (
    <div className="mb-4 mt-7 flex items-center gap-2.5 first:mt-0">
      <span className="grid h-[22px] w-[22px] shrink-0 place-items-center rounded-full bg-accent text-xs font-semibold leading-none text-accent-on">
        {n}
      </span>
      <span className="text-[15px] font-semibold text-fg">{title}</span>
      {children}
    </div>
  );
}
