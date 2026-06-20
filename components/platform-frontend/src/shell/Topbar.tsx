import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ROLES, TENANTS, useApp, type Lang, type Role, type ThemePref } from "@/app/store";
import { useUI } from "@/app/ui";
import { Icon } from "@/components/Icon";

const THEME_NAMES: Record<ThemePref, string> = { light: "浅色", dark: "深色", system: "跟随系统" };

export function Topbar() {
  const app = useApp();
  const { toast, confirm } = useUI();
  const navigate = useNavigate();
  const [openMenu, setOpenMenu] = useState<"role" | "user" | null>(null);
  const wrapRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (!(e.target as HTMLElement).closest("[data-menu-anchor]")) setOpenMenu(null);
    };
    document.addEventListener("click", onDoc);
    return () => document.removeEventListener("click", onDoc);
  }, []);

  const r = ROLES[app.role];
  const isAdmin = app.role === "system-admin";

  const pickRole = (k: Role) => {
    app.setRole(k);
    setOpenMenu(null);
    const restricted: Record<string, Role[]> = {
      "/tenants": ["system-admin", "tenant-admin"],
      "/resource-pools": ["system-admin"],
    };
    const path = window.location.pathname;
    if (restricted[path] && !restricted[path].includes(k)) navigate("/");
  };

  return (
    <header className="topbar" id="topbar" ref={wrapRef}>
      <button className="icon-btn menu-trigger" aria-label="菜单">
        <Icon name="menu" />
      </button>
      <button className="icon-btn" aria-label="折叠" title="折叠侧栏" onClick={app.toggleCollapsed}>
        <Icon name="layers" />
      </button>
      <div className="search">
        <span>
          <Icon name="search" />
        </span>
        <input placeholder="搜索任务 / 服务 / 模型 / 镜像…" />
        <kbd>⌘K</kbd>
      </div>
      <div className="spacer" />

      {/* role switcher */}
      <div style={{ position: "relative" }} data-menu-anchor>
        <button
          className="switch"
          onClick={(e) => {
            e.stopPropagation();
            setOpenMenu(openMenu === "role" ? null : "role");
          }}
        >
          <span className="cap">角色</span>
          <span>{r.short}</span>
          <span className="chev">
            <Icon name="chevron" />
          </span>
        </button>
        <div className={"menu" + (openMenu === "role" ? " open" : "")}>
          <div className="menu-label">切换演示角色</div>
          {(Object.keys(ROLES) as Role[]).map((k) => (
            <a
              key={k}
              className={"menu-item" + (app.role === k ? " sel" : "")}
              onClick={() => pickRole(k)}
            >
              <div>
                {ROLES[k].name}
                <small>{ROLES[k].note}</small>
              </div>
              {app.role === k && (
                <span className="ck">
                  <Icon name="check" />
                </span>
              )}
            </a>
          ))}
        </div>
      </div>

      <button className="icon-btn" title="帮助">
        <Icon name="help" />
      </button>
      <button className="icon-btn" title="通知">
        <Icon name="bell" />
        <span className="ping" />
      </button>

      {/* user menu */}
      <div className="user-wrap" data-menu-anchor>
        <button
          className="avatar"
          aria-label="用户菜单"
          title={r.person}
          onClick={(e) => {
            e.stopPropagation();
            setOpenMenu(openMenu === "user" ? null : "user");
          }}
        >
          {r.initials}
        </button>
        <div className={"menu user-menu" + (openMenu === "user" ? " open" : "")}>
          <div className="user-card">
            <div className="avatar">{r.initials}</div>
            <div className="u-meta">
              <div className="u-name">{r.person}</div>
              <div className="u-sub">{r.email}</div>
            </div>
          </div>
          <hr />
          <div className="menu-sub">
            <div className="menu-item tenant-trigger has-flyout">
              <div className="ti-text">
                <span className="ti-label">所属租户</span>
                <span className="ti-val">{app.tenantLabel()}</span>
              </div>
              <Icon name="chevronR" className="caret" />
            </div>
            <div className="flyout">
              <div className="menu-label">切换租户作用域</div>
              {isAdmin && (
                <>
                  <a
                    className={"menu-item" + (app.tenant === "all" ? " sel" : "")}
                    onClick={() => app.setTenant("all")}
                  >
                    <div>
                      全部租户<small>平台全局视图</small>
                    </div>
                    {app.tenant === "all" && (
                      <span className="ck">
                        <Icon name="check" />
                      </span>
                    )}
                  </a>
                  <hr />
                </>
              )}
              {TENANTS.map((t) => (
                <a
                  key={t.id}
                  className={"menu-item" + (app.tenant === t.id ? " sel" : "")}
                  onClick={() => app.setTenant(t.id)}
                >
                  <div>
                    {t.name}
                    <small>{t.note}</small>
                  </div>
                  {app.tenant === t.id && (
                    <span className="ck">
                      <Icon name="check" />
                    </span>
                  )}
                </a>
              ))}
            </div>
          </div>
          <hr />
          <div className="opt-row">
            <span className="opt-label">语言</span>
            <div className="segmented lang-seg">
              {(["zh", "en"] as Lang[]).map((l) => (
                <button
                  key={l}
                  className={app.lang === l ? "on" : ""}
                  onClick={(e) => {
                    e.stopPropagation();
                    if (app.lang === l) return;
                    app.setLang(l);
                    toast(l === "en" ? "界面语言已切换为 English（演示）" : "界面语言已切换为简体中文");
                  }}
                >
                  {l === "zh" ? "中文" : "English"}
                </button>
              ))}
            </div>
          </div>
          <div className="opt-row">
            <span className="opt-label">主题</span>
            <div className="segmented theme-seg">
              {(
                [
                  ["light", "sun"],
                  ["dark", "moon"],
                  ["system", "monitor"],
                ] as [ThemePref, string][]
              ).map(([val, icon]) => (
                <button
                  key={val}
                  title={THEME_NAMES[val]}
                  aria-label={THEME_NAMES[val]}
                  className={app.theme === val ? "on" : ""}
                  onClick={(e) => {
                    e.stopPropagation();
                    app.setTheme(val);
                    toast("主题已切换为「" + THEME_NAMES[val] + "」");
                  }}
                >
                  <Icon name={icon} className="seg-ic" />
                </button>
              ))}
            </div>
          </div>
          <hr />
          <a
            className="menu-item danger logout-row"
            onClick={(e) => {
              e.preventDefault();
              setOpenMenu(null);
              confirm({
                title: "退出登录",
                desc: "确定要退出当前登录吗？退出后需要重新登录才能继续访问控制台。",
                okLabel: "退出登录",
                toast: "已退出登录（演示）",
              });
            }}
          >
            <span>退出登录</span>
            <Icon name="logout" className="mi" />
          </a>
        </div>
      </div>
    </header>
  );
}
