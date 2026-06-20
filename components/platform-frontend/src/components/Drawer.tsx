import { useEffect, type ReactNode } from "react";
import { Icon } from "./Icon";

// Right-side slide-over drawer matching the prototype's .drawer / .overlay.
export function Drawer({
  open,
  onClose,
  title,
  sub,
  wide,
  footer,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  sub?: ReactNode;
  wide?: boolean;
  footer?: ReactNode;
  children: ReactNode;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  return (
    <>
      <div className={"overlay" + (open ? " open" : "")} onClick={onClose} />
      <div className={"drawer" + (wide ? " wide" : "") + (open ? " open" : "")} role="dialog" aria-modal="true">
        <div className="drawer-head">
          <div>
            <h3>{title}</h3>
            {sub && <div className="sub">{sub}</div>}
          </div>
          <button className="icon-btn" aria-label="关闭" onClick={onClose}>
            <Icon name="x" />
          </button>
        </div>
        <div className="drawer-body">{children}</div>
        {footer && <div className="drawer-foot">{footer}</div>}
      </div>
    </>
  );
}
