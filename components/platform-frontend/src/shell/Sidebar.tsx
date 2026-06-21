import { useMemo, type ReactNode } from "react";
import { Layout, Menu, type MenuProps } from "antd";
import { Link, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  HomeOutlined,
  DesktopOutlined,
  ExperimentOutlined,
  ThunderboltOutlined,
  CloudServerOutlined,
  BranchesOutlined,
  DatabaseOutlined,
  ContainerOutlined,
  TeamOutlined,
  ClusterOutlined,
} from "@ant-design/icons";
import { NAV, useApp, type NavItem } from "@/app/store";

const ICONS: Record<string, ReactNode> = {
  dashboard: <HomeOutlined />,
  workspace: <DesktopOutlined />,
  experiment: <ExperimentOutlined />,
  job: <ThunderboltOutlined />,
  service: <CloudServerOutlined />,
  traffic: <BranchesOutlined />,
  model: <DatabaseOutlined />,
  image: <ContainerOutlined />,
  tenant: <TeamOutlined />,
  pool: <ClusterOutlined />,
};

// Pick the nav entry whose path best matches the current location, so detail
// routes (/jobs/x) keep their parent list item (/jobs) highlighted.
function useSelectedKey(): string {
  const { pathname } = useLocation();
  return useMemo(() => {
    const all = NAV.flatMap((g) => g.items);
    const match = all
      .filter((it) => (it.path === "/" ? pathname === "/" : pathname.startsWith(it.path)))
      .sort((a, b) => b.path.length - a.path.length)[0];
    return match?.path ?? "/";
  }, [pathname]);
}

export function Sidebar() {
  const { collapsed, canSee } = useApp();
  const { t } = useTranslation();
  const selected = useSelectedKey();

  const toItem = (it: NavItem) => ({
    key: it.path,
    icon: ICONS[it.icon],
    label: <Link to={it.path}>{t(it.labelKey)}</Link>,
  });
  const items: MenuProps["items"] = [];
  NAV.forEach((group) => {
    const visible = group.items.filter(canSee);
    if (!visible.length) return;
    if (!group.groupKey) items.push(...visible.map(toItem));
    else items.push({ type: "group", key: group.groupKey, label: t(group.groupKey), children: visible.map(toItem) });
  });

  return (
    <Layout.Sider
      theme="light"
      collapsible
      collapsed={collapsed}
      trigger={null}
      width={232}
      collapsedWidth={64}
      className="h-full border-r border-border-soft"
    >
      <div className="flex h-14 items-center gap-2.5 px-4">
        <div className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-accent text-base font-extrabold text-accent-on">
          A
        </div>
        {!collapsed && (
          <div className="text-lg font-bold text-fg">
            Axis<b className="text-accent">ML</b>
          </div>
        )}
      </div>
      <Menu
        mode="inline"
        theme="light"
        selectedKeys={[selected]}
        items={items}
        className="border-none"
        style={{ background: "transparent" }}
      />
    </Layout.Sider>
  );
}
