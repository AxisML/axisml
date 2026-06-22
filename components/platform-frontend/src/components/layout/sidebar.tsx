import { useMemo, type ComponentType } from "react";
import { Link, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  Home,
  Monitor,
  FlaskConical,
  Zap,
  Server,
  Split,
  Database,
  Container,
  Users,
  Boxes,
} from "lucide-react";
import { NAV, useApp, type NavItem } from "@/app/store";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

const ICONS: Record<string, ComponentType<{ className?: string }>> = {
  dashboard: Home,
  workspace: Monitor,
  experiment: FlaskConical,
  job: Zap,
  service: Server,
  traffic: Split,
  model: Database,
  image: Container,
  tenant: Users,
  pool: Boxes,
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

  const renderItem = (it: NavItem) => {
    const Icon = ICONS[it.icon] ?? Home;
    const active = selected === it.path;
    const link = (
      <Link
        to={it.path}
        className={cn(
          "flex h-9 items-center gap-2.5 rounded-md px-2.5 text-sm font-medium transition-colors",
          collapsed && "justify-center px-0",
          active
            ? "bg-sidebar-accent text-sidebar-accent-foreground"
            : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground",
        )}
      >
        <Icon className="size-4 shrink-0" />
        {!collapsed && <span className="truncate">{t(it.labelKey)}</span>}
      </Link>
    );
    if (!collapsed) return <li key={it.path}>{link}</li>;
    return (
      <li key={it.path}>
        <Tooltip>
          <TooltipTrigger asChild>{link}</TooltipTrigger>
          <TooltipContent side="right">{t(it.labelKey)}</TooltipContent>
        </Tooltip>
      </li>
    );
  };

  return (
    <aside
      className={cn(
        "flex h-full shrink-0 flex-col border-r bg-sidebar text-sidebar-foreground transition-[width] duration-200",
        collapsed ? "w-16" : "w-58",
      )}
    >
      <div className="flex h-14 shrink-0 items-center gap-2.5 px-4">
        <div className="grid size-8 shrink-0 place-items-center rounded-md bg-primary text-base font-bold text-primary-foreground">
          A
        </div>
        {!collapsed && (
          <div className="text-lg font-semibold tracking-tight">
            Axis<span className="text-muted-foreground">ML</span>
          </div>
        )}
      </div>

      <ScrollArea className="flex-1">
        <nav className="flex flex-col gap-5 px-2 py-3">
          {NAV.map((group, gi) => {
            const visible = group.items.filter(canSee);
            if (!visible.length) return null;
            return (
              <div key={group.groupKey ?? `g${gi}`} className="flex flex-col gap-1">
                {group.groupKey && !collapsed && (
                  <div className="px-2.5 pb-1 font-mono text-[11px] font-medium tracking-wide text-muted-foreground/70 uppercase">
                    {t(group.groupKey)}
                  </div>
                )}
                <ul className="flex flex-col gap-0.5">{visible.map(renderItem)}</ul>
              </div>
            );
          })}
        </nav>
      </ScrollArea>
    </aside>
  );
}
