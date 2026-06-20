import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { Icon } from "@/components/Icon";

// Global UI services ported from js/app.js: bottom toasts and the centered
// confirm / blocking-delete modal. Exposed via useUI() so any page can fire them.

export interface ConfirmOptions {
  title?: string;
  desc?: ReactNode;
  info?: ReactNode;
  block?: ReactNode;
  blocked?: boolean;
  okLabel?: string;
  danger?: boolean;
  toast?: string;
  onConfirm?: () => void;
}

interface UIState {
  toast: (msg: string) => void;
  confirm: (opts: ConfirmOptions) => void;
}

const Ctx = createContext<UIState | null>(null);

interface ToastItem {
  id: number;
  msg: string;
  leaving?: boolean;
}

let toastSeq = 0;

export function UIProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [confirmOpts, setConfirmOpts] = useState<ConfirmOptions | null>(null);

  const toast = useCallback((msg: string) => {
    const id = ++toastSeq;
    setToasts((t) => [...t, { id, msg }]);
    setTimeout(() => setToasts((t) => t.map((x) => (x.id === id ? { ...x, leaving: true } : x))), 2400);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 2800);
  }, []);

  const confirm = useCallback((opts: ConfirmOptions) => setConfirmOpts(opts), []);
  const closeConfirm = useCallback(() => setConfirmOpts(null), []);

  useEffect(() => {
    if (!confirmOpts) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && closeConfirm();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [confirmOpts, closeConfirm]);

  return (
    <Ctx.Provider value={{ toast, confirm }}>
      {children}

      <div className="toast-wrap">
        {toasts.map((t) => (
          <div key={t.id} className="toast" style={t.leaving ? { opacity: 0, transition: "opacity .3s" } : undefined}>
            <Icon name="check" />
            <span>{t.msg}</span>
          </div>
        ))}
      </div>

      {confirmOpts && (
        <>
          <div className="overlay open" onClick={closeConfirm} />
          <div className="modal open" role="dialog" aria-modal="true">
            <div className="modal-head">
              <span className="warn-ico">
                <Icon name={confirmOpts.danger ? "alert" : "help"} />
              </span>
              <span>{confirmOpts.title || "确认操作"}</span>
            </div>
            <div className="modal-body">
              {confirmOpts.desc && <p>{confirmOpts.desc}</p>}
              {confirmOpts.info && (
                <div className="block-info warn-soft">
                  <div className="bi-title">
                    <Icon name="help" />
                    提示
                  </div>
                  {confirmOpts.info}
                </div>
              )}
              {confirmOpts.block && (
                <div className="block-info danger-soft">
                  <div className="bi-title">× 阻断</div>
                  {confirmOpts.block}
                </div>
              )}
            </div>
            <div className="modal-foot">
              <button className="btn" onClick={closeConfirm}>
                取消
              </button>
              <button
                className={`btn ${confirmOpts.danger === false ? "btn-primary" : "btn-danger"}`}
                disabled={confirmOpts.blocked}
                onClick={() => {
                  const { onConfirm, toast: t } = confirmOpts;
                  closeConfirm();
                  onConfirm?.();
                  if (t) toast(t);
                }}
              >
                {confirmOpts.okLabel || "确认删除"}
              </button>
            </div>
          </div>
        </>
      )}
    </Ctx.Provider>
  );
}

export function useUI() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useUI must be used within UIProvider");
  return ctx;
}
