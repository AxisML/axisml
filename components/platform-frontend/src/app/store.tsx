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

// ── Navigation config (single source of truth) ────────────────────────────────
// `labelKey` / `groupKey` are i18next keys resolved at render time by the
// Sidebar; `icon` is a stable name mapped to an AntD icon there.
export interface NavItem {
  key: string;
  labelKey: string;
  icon: string;
  path: string;
  roles?: Role[];
}
export interface NavGroup {
  groupKey?: string;
  items: NavItem[];
}

export const NAV: NavGroup[] = [
  { items: [{ key: "dashboard", labelKey: "nav.dashboard", icon: "dashboard", path: "/" }] },
  {
    groupKey: "nav.trainingCenter",
    items: [
      { key: "workspace", labelKey: "nav.workspace", icon: "workspace", path: "/workspaces" },
      { key: "experiments", labelKey: "nav.experiments", icon: "experiment", path: "/experiments" },
      { key: "jobs", labelKey: "nav.jobs", icon: "job", path: "/jobs" },
    ],
  },
  {
    groupKey: "nav.serviceCenter",
    items: [
      { key: "services", labelKey: "nav.services", icon: "service", path: "/services" },
      { key: "traffic", labelKey: "nav.traffic", icon: "traffic", path: "/traffic" },
    ],
  },
  {
    groupKey: "nav.assetCenter",
    items: [
      { key: "models", labelKey: "nav.models", icon: "model", path: "/models" },
      { key: "images", labelKey: "nav.images", icon: "image", path: "/images" },
    ],
  },
  {
    groupKey: "nav.systemMgmt",
    items: [
      {
        key: "tenants",
        labelKey: "nav.tenants",
        icon: "tenant",
        path: "/tenants",
        roles: ["system-admin", "tenant-admin"],
      },
      { key: "pools", labelKey: "nav.pools", icon: "pool", path: "/resource-pools", roles: ["system-admin"] },
    ],
  },
];

export type Role = "system-admin" | "tenant-admin" | "user";

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
