import { useState, type FormEvent } from "react";
import { useNavigate, useLocation, Navigate } from "react-router-dom";
import { useSession } from "@/app/session";
import { Icon } from "@/components/Icon";

// Real login screen — authenticates against platform-backend POST /auth/login.
// On success the JWT is persisted and identity is hydrated from /auth/me; the
// gate in router.tsx then admits the user to the console.
const LOGIN_STYLES = `
.login-wrap { min-height:100vh; display:grid; place-items:center; background:var(--bg); padding:24px; }
.login-card { width:100%; max-width:380px; background:var(--surface); border:1px solid var(--border); border-radius:var(--radius-lg); padding:32px 28px; box-shadow:var(--shadow-lg, 0 12px 40px rgba(0,0,0,.12)); }
.login-brand { display:flex; align-items:center; gap:10px; margin-bottom:6px; }
.login-brand .mark { width:34px; height:34px; border-radius:9px; background:var(--accent); color:var(--accent-on); display:grid; place-items:center; font-weight:800; font-size:18px; }
.login-brand .name { font-size:19px; font-weight:700; color:var(--fg); }
.login-sub { font-size:13px; color:var(--muted); margin:0 0 24px; }
.login-form { display:flex; flex-direction:column; gap:16px; }
.login-err { display:flex; gap:8px; align-items:center; padding:9px 12px; border-radius:var(--radius-sm); background:color-mix(in oklab, var(--danger) 10%, transparent); border:1px solid color-mix(in oklab, var(--danger) 35%, var(--border)); color:var(--danger); font-size:13px; }
.login-err svg { width:16px; height:16px; flex:none; }
.login-foot { margin-top:18px; font-size:12px; color:var(--muted); text-align:center; }
`;

export default function Login() {
  const { status, login } = useSession();
  const navigate = useNavigate();
  const location = useLocation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Already authenticated → bounce to wherever they were headed (or home).
  if (status === "authed") {
    const to = (location.state as { from?: string } | null)?.from || "/";
    return <Navigate to={to} replace />;
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      await login(username.trim(), password);
      const to = (location.state as { from?: string } | null)?.from || "/";
      navigate(to, { replace: true });
    } catch (e) {
      setErr(e instanceof Error ? e.message : "登录失败");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login-wrap">
      <style>{LOGIN_STYLES}</style>
      <div className="login-card">
        <div className="login-brand">
          <div className="mark">A</div>
          <div className="name">
            Axis<b>ML</b>
          </div>
        </div>
        <p className="login-sub">登录到 AI 训练与推理平台控制台</p>
        <form className="login-form" onSubmit={onSubmit}>
          {err && (
            <div className="login-err">
              <Icon name="alert" />
              <span>{err}</span>
            </div>
          )}
          <div className="field">
            <label htmlFor="login-user">用户名</label>
            <input
              id="login-user"
              className="input"
              autoComplete="username"
              autoFocus
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="请输入用户名"
            />
          </div>
          <div className="field">
            <label htmlFor="login-pass">密码</label>
            <input
              id="login-pass"
              className="input"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="请输入密码"
            />
          </div>
          <button
            type="submit"
            className="btn btn-primary btn-lg"
            disabled={busy || !username || !password}
          >
            {busy ? "登录中…" : "登录"}
          </button>
        </form>
        <div className="login-foot">AxisML · Kubernetes-native ML Platform</div>
      </div>
    </div>
  );
}
