import { App, Alert } from "antd";
import { useTranslation } from "react-i18next";
import type { ReactNode } from "react";

// Global UI services (toasts + confirm/blocking-delete modal) backed by Ant
// Design's `App` context (message + Modal). The interface is intentionally the
// same `{ toast, confirm }` shape the pages already consume, so call sites need
// no churn — the implementation just delegates to AntD instead of bespoke CSS.

export interface ConfirmOptions {
  title?: string;
  desc?: ReactNode;
  info?: ReactNode;
  block?: ReactNode;
  blocked?: boolean;
  okLabel?: string;
  /** Defaults to a danger (red) primary button; pass `false` for a normal one. */
  danger?: boolean;
  toast?: string;
  onConfirm?: () => void;
}

export function useUI() {
  const { message, modal } = App.useApp();
  const { t } = useTranslation();

  const toast = (msg: string) => {
    void message.success(msg);
  };

  const confirm = (opts: ConfirmOptions) => {
    modal.confirm({
      title: opts.title || t("common.confirmAction"),
      content: (
        <div className="space-y-2">
          {opts.desc && <p className="m-0 text-fg-2">{opts.desc}</p>}
          {opts.info && <Alert type="warning" showIcon message={opts.info} />}
          {opts.block && <Alert type="error" showIcon message={opts.block} />}
        </div>
      ),
      okText: opts.okLabel || t("common.confirmDelete"),
      okButtonProps: { danger: opts.danger !== false, disabled: opts.blocked },
      cancelText: t("common.cancel"),
      onOk: () => {
        opts.onConfirm?.();
        if (opts.toast) void message.success(opts.toast);
      },
    });
  };

  return { toast, confirm };
}
