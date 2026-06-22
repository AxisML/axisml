import { type ReactNode } from "react";
import { Copy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useUI } from "@/app/ui";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

// Dark terminal-style code / command surface shared by command panes, pull-/push-
// guides, and resolve commands. Uses the same near-black palette as LogViewer so
// every monospace block reads identically (it stays dark in both themes, like a
// real terminal). Pass `copy` to overlay a copy-to-clipboard button.
export function CodeBlock({
  children,
  copy,
  className,
}: {
  children: ReactNode;
  copy?: string;
  className?: string;
}) {
  const { t } = useTranslation();
  const { toast } = useUI();
  return (
    <div className={cn("relative", className)}>
      <pre className="m-0 overflow-auto rounded-md bg-zinc-950 p-4 font-mono text-xs leading-relaxed text-zinc-100">
        {children}
      </pre>
      {copy != null && (
        <Button
          variant="ghost"
          size="icon-sm"
          className="absolute top-2 right-2 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
          aria-label={t("common.copy", { defaultValue: "Copy" })}
          onClick={() => {
            void navigator.clipboard?.writeText(copy);
            toast(t("common.copied", { defaultValue: "已复制" }));
          }}
        >
          <Copy />
        </Button>
      )}
    </div>
  );
}
