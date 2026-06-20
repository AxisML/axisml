import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

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

export const ROLES: Record<
  Role,
  { name: string; short: string; note: string; person: string; email: string; initials: string }
> = {
  "system-admin": {
    name: "系统管理员",
    short: "系统管理员",
    note: "平台级超管",
    person: "张伟",
    email: "zhangwei@axisml.io",
    initials: "ZW",
  },
  "tenant-admin": {
    name: "租户管理员",
    short: "租户管理员",
    note: "本租户负责人",
    person: "李娜",
    email: "lina@axisml.io",
    initials: "LN",
  },
  user: {
    name: "普通用户",
    short: "普通用户",
    note: "算法 / 推理工程师",
    person: "王芳",
    email: "wangfang@axisml.io",
    initials: "WF",
  },
};

export interface Tenant {
  id: string;
  name: string;
  note: string;
}
export const TENANTS: Tenant[] = [
  { id: "llm-lab", name: "大模型研究院", note: "12 成员 · A100/H100" },
  { id: "rec-algo", name: "推荐算法团队", note: "8 成员 · A100" },
  { id: "av-perception", name: "智能驾驶感知", note: "15 成员 · H100/L40S" },
  { id: "risk-ai", name: "风控 AI", note: "6 成员 · 通用 CPU" },
];

export type ThemePref = "light" | "dark" | "system";
export type Lang = "zh" | "en";

const LS = {
  role: "axisml.role",
  tenant: "axisml.tenant",
  collapsed: "axisml.collapsed",
  theme: "axisml.theme",
  lang: "axisml.lang",
} as const;

interface AppState {
  role: Role;
  tenant: string; // "all" or a tenant id
  collapsed: boolean;
  theme: ThemePref;
  lang: Lang;
  setRole: (r: Role) => void;
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
  const [role, setRoleState] = useState<Role>(
    () => (localStorage.getItem(LS.role) as Role) || "system-admin",
  );
  const [tenant, setTenantState] = useState<string>(() => localStorage.getItem(LS.tenant) || "all");
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

  const setRole = useCallback((r: Role) => {
    localStorage.setItem(LS.role, r);
    setRoleState(r);
  }, []);
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
      setRole,
      setTenant,
      toggleCollapsed,
      setTheme,
      setLang,
      tenantLabel: () =>
        tenant === "all" ? "全部租户" : TENANTS.find((t) => t.id === tenant)?.name || "全部租户",
      canSee: (item: NavItem) => !item.roles || item.roles.includes(role),
    }),
    [role, tenant, collapsed, theme, lang, setRole, setTenant, toggleCollapsed, setTheme, setLang],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useApp() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useApp must be used within AppStoreProvider");
  return ctx;
}
