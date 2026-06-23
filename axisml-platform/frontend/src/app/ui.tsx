import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { toast as sonnerToast } from "sonner";
import { TriangleAlert, OctagonX } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useApp } from "./store";
import { applyLang } from "@/i18n";

// Global UI services: toasts (sonner) + a single shared confirm/blocking-delete
// dialog (Radix AlertDialog). The interface mirrors the prior `{ toast, confirm }`
// shape so call sites need no churn. Also the place we keep i18next/dayjs locale
// in lock-step with the app store's `lang` (the data-theme toggle lives in store).

export interface ConfirmOptions {
  title?: string;
  desc?: ReactNode;
  info?: ReactNode;
  block?: ReactNode;
  blocked?: boolean;
  okLabel?: string;
  /** Defaults to a danger (destructive) confirm button; pass `false` for a normal one. */
  danger?: boolean;
  toast?: string;
  onConfirm?: () => void;
}

interface UIContext {
  toast: (msg: string) => void;
  confirm: (opts: ConfirmOptions) => void;
}

const Ctx = createContext<UIContext | null>(null);

export function UIProvider({ children }: { children: ReactNode }) {
  const { lang } = useApp();
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [opts, setOpts] = useState<ConfirmOptions>({});

  // Keep i18next + dayjs locale in lock-step with the chosen language.
  useEffect(() => {
    applyLang(lang);
  }, [lang]);

  const toast = useCallback((msg: string) => {
    sonnerToast.success(msg);
  }, []);

  const confirm = useCallback((next: ConfirmOptions) => {
    setOpts(next);
    setOpen(true);
  }, []);

  const onConfirm = () => {
    opts.onConfirm?.();
    if (opts.toast) sonnerToast.success(opts.toast);
    setOpen(false);
  };

  const value = useMemo<UIContext>(() => ({ toast, confirm }), [toast, confirm]);

  return (
    <Ctx.Provider value={value}>
      <TooltipProvider delayDuration={200}>{children}</TooltipProvider>
      <Toaster position="top-center" />
      <AlertDialog open={open} onOpenChange={setOpen}>
        <AlertDialogContent size="sm" className="text-left sm:max-w-md">
          <AlertDialogHeader className="sm:place-items-start sm:text-left">
            <AlertDialogTitle>{opts.title || t("common.confirmAction")}</AlertDialogTitle>
            {opts.desc && <AlertDialogDescription>{opts.desc}</AlertDialogDescription>}
          </AlertDialogHeader>
          {(opts.info || opts.block) && (
            <div className="flex flex-col gap-2">
              {opts.info && (
                <Alert variant="warning">
                  <TriangleAlert />
                  <AlertDescription>{opts.info}</AlertDescription>
                </Alert>
              )}
              {opts.block && (
                <Alert variant="destructive">
                  <OctagonX />
                  <AlertTitle>{opts.block}</AlertTitle>
                </Alert>
              )}
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant={opts.danger === false ? "default" : "destructive"}
              disabled={opts.blocked}
              onClick={onConfirm}
            >
              {opts.okLabel || t("common.confirmDelete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Ctx.Provider>
  );
}

export function useUI() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useUI must be used within UIProvider");
  return ctx;
}
