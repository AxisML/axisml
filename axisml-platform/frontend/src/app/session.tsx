import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import * as sdk from "@/api/generated";
import type { MeResponse } from "@/api/generated";
import type { Role } from "./store";

// ── Real auth session ─────────────────────────────────────────────────────────
// The platform-backend issues a JWT at POST /auth/login. We persist it in
// localStorage (key consumed by the generated client's `auth` resolver — see
// api/client.ts) and hydrate the current identity from GET /auth/me. Every other
// page reads identity (role, tenant scope, profile) from here — there is no more
// "demo role" picker; the role is whatever the backend says it is.

const TOKEN_KEY = "axisml.token";

export type SessionStatus = "loading" | "authed" | "anon";

export interface Session {
  status: SessionStatus;
  me: MeResponse | null;
  role: Role; // derived top-level role for nav gating
  displayName: string;
  email: string;
  initials: string;
  permissions: string[];
  isSystemAdmin: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}
function setToken(t: string | null) {
  if (t) localStorage.setItem(TOKEN_KEY, t);
  else localStorage.removeItem(TOKEN_KEY);
}

function deriveRole(me: MeResponse | null): Role {
  if (!me) return "user";
  if (me.isSystemAdmin) return "system-admin";
  if (me.tenantRoles?.some((r) => r.roleName === "tenant-admin")) return "tenant-admin";
  return "user";
}

function initialsOf(me: MeResponse | null): string {
  const base = me?.user.displayName || me?.user.username || "";
  if (!base) return "?";
  const parts = base.trim().split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return base.slice(0, 2).toUpperCase();
}

const Ctx = createContext<Session | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>(() => (getToken() ? "loading" : "anon"));
  const [me, setMe] = useState<MeResponse | null>(null);

  const refresh = useCallback(async () => {
    if (!getToken()) {
      setMe(null);
      setStatus("anon");
      return;
    }
    const { data, error } = await sdk.getCurrentUser();
    if (error || !data) {
      setToken(null);
      setMe(null);
      setStatus("anon");
      return;
    }
    setMe(data);
    setStatus("authed");
  }, []);

  // Hydrate identity on first mount when a token is already present.
  useEffect(() => {
    void refresh();
  }, [refresh]);

  const login = useCallback(
    async (username: string, password: string) => {
      const { data, error } = await sdk.login({ body: { username, password } });
      if (error || !data) {
        const msg =
          (error as { detail?: string; title?: string } | undefined)?.detail ||
          (error as { title?: string } | undefined)?.title ||
          "用户名或密码错误";
        throw new Error(msg);
      }
      setToken(data.jwt);
      await refresh();
    },
    [refresh],
  );

  const logout = useCallback(async () => {
    try {
      await sdk.logout();
    } catch {
      // best-effort server-side revocation; clear locally regardless
    }
    setToken(null);
    setMe(null);
    setStatus("anon");
  }, []);

  const value = useMemo<Session>(
    () => ({
      status,
      me,
      role: deriveRole(me),
      displayName: me?.user.displayName || me?.user.username || "",
      email: me?.user.email || "",
      initials: initialsOf(me),
      permissions: me?.permissions ?? [],
      isSystemAdmin: !!me?.isSystemAdmin,
      login,
      logout,
    }),
    [status, me, login, logout],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useSession() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useSession must be used within SessionProvider");
  return ctx;
}
