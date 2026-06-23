import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";

// Standard right-side drawer shared by every create / edit / detail panel.
// Owns the chrome that was hand-rolled in ~15 places: a bordered header with a
// title + optional subtitle, a scrollable body, and an optional bordered footer
// with a Cancel button and a primary action that shows a Spinner while pending.
// Header / body / footer share the same px-6 inset so their left edges align.
//
// Sizes: "default" (560px) for forms, "compact" (420px) for small confirm-style
// panels (scale, split). Pass `footer` to fully override the action row (e.g. a
// single "Done" button); otherwise provide `submitLabel` + `onSubmit`.
export function FormDrawer({
  title,
  titleClassName,
  subtitle,
  size = "default",
  children,
  bodyClassName,
  onClose,
  onSubmit,
  submitLabel,
  submitDisabled,
  submitting,
  cancelLabel,
  footer,
}: {
  title: ReactNode;
  titleClassName?: string;
  subtitle?: ReactNode;
  size?: "default" | "compact";
  children: ReactNode;
  bodyClassName?: string;
  onClose: () => void;
  onSubmit?: () => void;
  submitLabel?: ReactNode;
  submitDisabled?: boolean;
  submitting?: boolean;
  cancelLabel?: ReactNode;
  footer?: ReactNode;
}) {
  const { t } = useTranslation();
  const width = size === "compact" ? "sm:max-w-[420px]" : "sm:max-w-[560px]";
  const hasActionFooter = submitLabel != null && onSubmit;

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className={cn("flex w-full flex-col gap-0 p-0", width)}>
        <SheetHeader className="border-b px-6 py-4">
          <SheetTitle className={titleClassName}>{title}</SheetTitle>
          {subtitle != null && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </SheetHeader>

        <div className={cn("flex-1 overflow-auto px-6 py-4", bodyClassName)}>{children}</div>

        {footer ? (
          <SheetFooter className="flex-row justify-end gap-2 border-t px-6 py-4">{footer}</SheetFooter>
        ) : hasActionFooter ? (
          <SheetFooter className="flex-row justify-end gap-2 border-t px-6 py-4">
            <Button variant="outline" onClick={onClose}>
              {cancelLabel ?? t("common.cancel")}
            </Button>
            <Button onClick={onSubmit} disabled={submitDisabled || submitting}>
              {submitting && <Spinner data-icon="inline-start" />}
              {submitLabel}
            </Button>
          </SheetFooter>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
