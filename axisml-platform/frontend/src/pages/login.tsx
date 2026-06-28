import { useState } from "react";
import { useNavigate, useLocation, Navigate } from "react-router-dom";
import { User, Lock, OctagonX } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useSession } from "@/app/session";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Field, FieldLabel } from "@/components/ui/field";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Spinner } from "@/components/ui/spinner";

// Real login screen — authenticates against platform-backend POST /auth/login.
// On success the JWT is persisted and identity is hydrated from /auth/me; the
// gate in router.tsx then admits the user to the platform.
export default function Login() {
  const { status, login } = useSession();
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Already authenticated → bounce to wherever they were headed (or home).
  if (status === "authed") {
    const to = (location.state as { from?: string } | null)?.from || "/";
    return <Navigate to={to} replace />;
  }

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password) return;
    setErr(null);
    setBusy(true);
    try {
      await login(username.trim(), password);
      const to = (location.state as { from?: string } | null)?.from || "/";
      navigate(to, { replace: true });
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("login.error"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="grid min-h-screen place-items-center bg-background p-6">
      <Card className="w-full max-w-sm">
        <CardContent className="p-8">
          <div className="mb-6 flex items-center gap-2.5">
            <div className="grid size-9 place-items-center rounded-md bg-primary text-lg font-bold text-primary-foreground">
              A
            </div>
            <div className="text-xl font-semibold tracking-tight">
              Axis<span className="text-muted-foreground">ML</span>
            </div>
          </div>

          {err && (
            <Alert variant="destructive" className="mb-4">
              <OctagonX />
              <AlertDescription>{err}</AlertDescription>
            </Alert>
          )}

          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <Field>
              <FieldLabel htmlFor="username">{t("login.username")}</FieldLabel>
              <InputGroup>
                <InputGroupAddon>
                  <User />
                </InputGroupAddon>
                <InputGroupInput
                  id="username"
                  autoFocus
                  autoComplete="username"
                  placeholder={t("login.usernamePlaceholder")}
                  value={username}
                  disabled={busy}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </InputGroup>
            </Field>
            <Field>
              <FieldLabel htmlFor="password">{t("login.password")}</FieldLabel>
              <InputGroup>
                <InputGroupAddon>
                  <Lock />
                </InputGroupAddon>
                <InputGroupInput
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder={t("login.passwordPlaceholder")}
                  value={password}
                  disabled={busy}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </InputGroup>
            </Field>
            <Button type="submit" size="lg" className="w-full" disabled={busy}>
              {busy && <Spinner data-icon="inline-start" />}
              {busy ? t("login.submitting") : t("login.submit")}
            </Button>
          </form>

          <p className="mt-5 text-center text-xs text-muted-foreground">{t("login.footer")}</p>
        </CardContent>
      </Card>
    </div>
  );
}
