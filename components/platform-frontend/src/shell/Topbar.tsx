import { Layout, Input, Dropdown, Popover, Avatar, Badge, Button, Segmented, Divider, Tooltip } from "antd";
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SearchOutlined,
  QuestionCircleOutlined,
  BellOutlined,
  RightOutlined,
  LogoutOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useApp, type Lang, type ThemePref } from "@/app/store";
import { useSession } from "@/app/session";
import { useUI } from "@/app/ui";
import { useTenantOptions } from "@/api/hooks";

export function Topbar() {
  const app = useApp();
  const session = useSession();
  const { toast, confirm } = useUI();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const initials = session.initials;
  const person = session.displayName || session.me?.user.username || "";
  const email = session.email || session.me?.user.username || "";
  const tenantOptions = useTenantOptions();
  const currentTenant = tenantOptions.find((x) => x.id === app.tenant);

  const themeNames: Record<ThemePref, string> = {
    light: t("topbar.themeLight"),
    dark: t("topbar.themeDark"),
    system: t("topbar.themeSystem"),
  };

  const onLang = (l: Lang) => {
    if (app.lang === l) return;
    app.setLang(l);
    toast(l === "en" ? t("topbar.langSwitchedEn") : t("topbar.langSwitchedZh"));
  };
  const onTheme = (val: ThemePref) => {
    app.setTheme(val);
    toast(t("topbar.themeSwitched", { name: themeNames[val] }));
  };
  const onLogout = () =>
    confirm({
      title: t("topbar.logoutConfirmTitle"),
      desc: t("topbar.logoutConfirmDesc"),
      okLabel: t("topbar.logout"),
      danger: false,
      onConfirm: () => void session.logout().then(() => navigate("/login", { replace: true })),
    });

  return (
    <Layout.Header className="sticky top-0 z-10 flex h-14 items-center gap-3 border-b border-border-soft bg-bg px-4">
      <Button
        type="text"
        aria-label={t("topbar.collapse")}
        icon={app.collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
        onClick={app.toggleCollapsed}
      />

      <Input
        allowClear
        prefix={<SearchOutlined className="text-muted" />}
        placeholder={t("topbar.searchPlaceholder")}
        className="max-w-sm"
      />

      <div className="flex-1" />

      <Tooltip title={t("topbar.help")}>
        <Button type="text" aria-label={t("topbar.help")} icon={<QuestionCircleOutlined />} />
      </Tooltip>
      <Tooltip title={t("topbar.notifications")}>
        <Badge dot offset={[-2, 2]}>
          <Button type="text" aria-label={t("topbar.notifications")} icon={<BellOutlined />} />
        </Badge>
      </Tooltip>

      <Dropdown
        trigger={["click"]}
        placement="bottomRight"
        dropdownRender={() => (
          <div className="w-64 rounded-lg border border-border-soft bg-bg p-1 shadow-lg">
            <div className="flex items-center gap-3 p-3">
              <Avatar className="!bg-accent">{initials}</Avatar>
              <div className="min-w-0">
                <div className="truncate font-semibold text-fg">{person}</div>
                <div className="truncate text-xs text-muted">{email}</div>
              </div>
            </div>
            <Divider className="my-1" />
            {/* Tenant scope switcher — a left flyout, per the product prototype.
                Exactly one tenant is always active. */}
            <Popover
              trigger="hover"
              placement="leftTop"
              arrow={false}
              overlayInnerStyle={{ padding: 4 }}
              content={
                <div className="w-52">
                  <div className="px-3 py-1.5 text-xs text-muted">{t("topbar.switchTenant")}</div>
                  {tenantOptions.length ? (
                    tenantOptions.map((tn) => (
                      <div
                        key={tn.id}
                        onClick={() => app.setTenant(tn.id)}
                        className={
                          "flex cursor-pointer items-center justify-between gap-6 rounded px-3 py-1.5 hover:bg-surface " +
                          (app.tenant === tn.id ? "text-accent" : "text-fg")
                        }
                      >
                        <span>{tn.name}</span>
                        <span className="text-xs text-muted">{tn.note}</span>
                      </div>
                    ))
                  ) : (
                    <div className="px-3 py-1.5 text-xs text-muted">{t("topbar.noTenants")}</div>
                  )}
                </div>
              }
            >
              <div className="flex cursor-pointer items-center justify-between rounded px-3 py-2 hover:bg-surface">
                <div className="flex min-w-0 flex-col">
                  <span className="text-xs text-muted">{t("topbar.tenant")}</span>
                  <span className="truncate text-sm text-fg">
                    {currentTenant?.name || app.tenant || t("topbar.noTenants")}
                  </span>
                </div>
                <RightOutlined className="text-xs text-muted" />
              </div>
            </Popover>
            <Divider className="my-1" />
            <div className="flex items-center justify-between px-3 py-2">
              <span className="text-sm text-fg-2">{t("topbar.language")}</span>
              <Segmented<Lang>
                size="small"
                value={app.lang}
                onChange={onLang}
                options={[
                  { label: "中文", value: "zh" },
                  { label: "EN", value: "en" },
                ]}
              />
            </div>
            <div className="flex items-center justify-between px-3 py-2">
              <span className="text-sm text-fg-2">{t("topbar.theme")}</span>
              <Segmented<ThemePref>
                size="small"
                value={app.theme}
                onChange={onTheme}
                options={[
                  { label: themeNames.light, value: "light" },
                  { label: themeNames.dark, value: "dark" },
                  { label: themeNames.system, value: "system" },
                ]}
              />
            </div>
            <Divider className="my-1" />
            <Button type="text" danger block className="!justify-start" icon={<LogoutOutlined />} onClick={onLogout}>
              {t("topbar.logout")}
            </Button>
          </div>
        )}
      >
        <span title={person} className="cursor-pointer">
          <Avatar className="!bg-accent">{initials}</Avatar>
        </span>
      </Dropdown>
    </Layout.Header>
  );
}
