import { useState } from "react";
import { useNavigate, useLocation, Navigate } from "react-router-dom";
import { Form, Input, Button, Alert, Typography } from "antd";
import { UserOutlined, LockOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useSession } from "@/app/session";

// Real login screen — authenticates against platform-backend POST /auth/login.
// On success the JWT is persisted and identity is hydrated from /auth/me; the
// gate in router.tsx then admits the user to the console.
export default function Login() {
  const { status, login } = useSession();
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation();
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Already authenticated → bounce to wherever they were headed (or home).
  if (status === "authed") {
    const to = (location.state as { from?: string } | null)?.from || "/";
    return <Navigate to={to} replace />;
  }

  const onFinish = async (values: { username: string; password: string }) => {
    setErr(null);
    setBusy(true);
    try {
      await login(values.username.trim(), values.password);
      const to = (location.state as { from?: string } | null)?.from || "/";
      navigate(to, { replace: true });
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("login.error"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid min-h-screen place-items-center bg-surface p-6">
      <div className="w-full max-w-sm rounded-lg border border-border-soft bg-bg p-8 shadow-lg">
        <div className="mb-1 flex items-center gap-2.5">
          <div className="grid h-9 w-9 place-items-center rounded-lg bg-accent text-lg font-extrabold text-accent-on">
            A
          </div>
          <div className="text-xl font-bold text-fg">
            Axis<b className="text-accent">ML</b>
          </div>
        </div>
        <Typography.Paragraph type="secondary" className="!mb-6">
          {t("login.subtitle")}
        </Typography.Paragraph>

        {err && <Alert type="error" showIcon message={err} className="!mb-4" />}

        <Form layout="vertical" requiredMark={false} onFinish={onFinish} disabled={busy}>
          <Form.Item
            name="username"
            label={t("login.username")}
            rules={[{ required: true, message: t("login.usernamePlaceholder") }]}
          >
            <Input
              size="large"
              autoFocus
              autoComplete="username"
              prefix={<UserOutlined className="text-muted" />}
              placeholder={t("login.usernamePlaceholder")}
            />
          </Form.Item>
          <Form.Item
            name="password"
            label={t("login.password")}
            rules={[{ required: true, message: t("login.passwordPlaceholder") }]}
          >
            <Input.Password
              size="large"
              autoComplete="current-password"
              prefix={<LockOutlined className="text-muted" />}
              placeholder={t("login.passwordPlaceholder")}
            />
          </Form.Item>
          <Button type="primary" size="large" htmlType="submit" block loading={busy}>
            {busy ? t("login.submitting") : t("login.submit")}
          </Button>
        </Form>

        <p className="mt-5 text-center text-xs text-muted">{t("login.footer")}</p>
      </div>
    </div>
  );
}
