import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useSession } from "./session";

// ── Navigation config (single source of truth, ported from js/app.js NAV) ──────
export interface NavItem {
  key: string;
  label: string;
  icon: string;
  path: string;
  roles?: Role[];
}
export interface NavGroup {
  group?: string;
  items: NavItem[];
}

export const NAV: NavGroup[] = [
  { items: [{ key: "dashboard", label: "首页", icon: "dashboard", path: "/" }] },
  {
    group: "训练中心",
    items: [
      { key: "workspace", label: "工作区", icon: "workspace", path: "/workspaces" },
      { key: "experiments", label: "实验管理", icon: "experiment", path: "/experiments" },
      { key: "jobs", label: "自定义任务", icon: "job", path: "/jobs" },
    ],
  },
  {
    group: "服务中心",
    items: [
      { key: "services", label: "在线服务", icon: "service", path: "/services" },
      { key: "traffic", label: "流量配置", icon: "traffic", path: "/traffic" },
    ],
  },
  {
    group: "资产中心",
    items: [
      { key: "models", label: "模型仓", icon: "model", path: "/models" },
      { key: "images", label: "镜像仓", icon: "image", path: "/images" },
    ],
  },
  {
    group: "系统管理",
    items: [
      {
        key: "tenants",
        label: "租户管理",
        icon: "tenant",
        path: "/tenants",
        roles: ["system-admin", "tenant-admin"],
      },
      { key: "pools", label: "资源池管理", icon: "pool", path: "/resource-pools", roles: ["system-admin"] },
    ],
  },
];

export type Role = "system-admin" | "tenant-admin" | "user";

export const ROLE_LABELS: Record<Role, string> = {
  "system-admin": "系统管理员",
  "tenant-admin": "租户管理员",
  user: "普通用户",
};

export type ThemePref = "light" | "dark" | "system";
export type Lang = "zh" | "en";

const LS = {
  tenant: "axisml.tenant",
  collapsed: "axisml.collapsed",
  theme: "axisml.theme",
  lang: "axisml.lang",
} as const;

interface AppState {
  role: Role; // sourced from the real auth session, not a demo picker
  tenant: string; // "all" or a tenant id
  collapsed: boolean;
  theme: ThemePref;
  lang: Lang;
  setTenant: (t: string) => void;
  toggleCollapsed: () => void;
  setTheme: (t: ThemePref) => void;
  setLang: (l: Lang) => void;
  tenantLabel: () => string;
  canSee: (item: NavItem) => boolean;
}

const Ctx = createContext<AppState | null>(null);

function prefersDark() {
  return !!(window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches);
}
function resolveTheme(p: ThemePref) {
  return p === "system" ? (prefersDark() ? "dark" : "light") : p;
}

export function AppStoreProvider({ children }: { children: ReactNode }) {
  const { role } = useSession();
  // Active tenant scope. There is no "all-tenants" view — the switcher always
  // selects exactly one tenant (a default is chosen on bootstrap). Legacy "all"
  // values are migrated to unset.
  const [tenant, setTenantState] = useState<string>(() => {
    const stored = localStorage.getItem(LS.tenant);
    return stored && stored !== "all" ? stored : "";
  });
  const [collapsed, setCollapsed] = useState<boolean>(() => localStorage.getItem(LS.collapsed) === "1");
  const [theme, setThemeState] = useState<ThemePref>(
    () => (localStorage.getItem(LS.theme) as ThemePref) || "light",
  );
  const [lang, setLangState] = useState<Lang>(() => (localStorage.getItem(LS.lang) as Lang) || "zh");

  // Apply theme token override on <html> (mirrors applyTheme() in app.js).
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", resolveTheme(theme));
    if (theme !== "system" || !window.matchMedia) return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => document.documentElement.setAttribute("data-theme", resolveTheme("system"));
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [theme]);

  useEffect(() => {
    document.documentElement.setAttribute("lang", lang === "en" ? "en" : "zh-CN");
  }, [lang]);

  const setTenant = useCallback((t: string) => {
    localStorage.setItem(LS.tenant, t);
    setTenantState(t);
  }, []);
  const toggleCollapsed = useCallback(() => {
    setCollapsed((c) => {
      const next = !c;
      localStorage.setItem(LS.collapsed, next ? "1" : "0");
      return next;
    });
  }, []);
  const setTheme = useCallback((t: ThemePref) => {
    localStorage.setItem(LS.theme, t);
    setThemeState(t);
  }, []);
  const setLang = useCallback((l: Lang) => {
    localStorage.setItem(LS.lang, l);
    setLangState(l);
  }, []);

  const value = useMemo<AppState>(
    () => ({
      role,
      tenant,
      collapsed,
      theme,
      lang,
      setTenant,
      toggleCollapsed,
      setTheme,
      setLang,
      tenantLabel: () => tenant,
      canSee: (item: NavItem) => !item.roles || item.roles.includes(role),
    }),
    [role, tenant, collapsed, theme, lang, setTenant, toggleCollapsed, setTheme, setLang],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useApp() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useApp must be used within AppStoreProvider");
  return ctx;
}
