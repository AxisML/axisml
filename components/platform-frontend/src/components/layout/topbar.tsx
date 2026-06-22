import { PanelLeft, Search, CircleHelp, Bell, LogOut } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useApp, type Lang, type ThemePref } from "@/app/store";
import { useSession } from "@/app/session";
import { useUI } from "@/app/ui";
import { useTenantOptions } from "@/api/hooks";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

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
    if (!l || app.lang === l) return;
    app.setLang(l);
    toast(l === "en" ? t("topbar.langSwitchedEn") : t("topbar.langSwitchedZh"));
  };
  const onTheme = (val: ThemePref) => {
    if (!val) return;
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
    <header className="sticky top-0 z-10 flex h-14 shrink-0 items-center gap-3 border-b bg-background px-4">
      <Button
        variant="ghost"
        size="icon"
        aria-label={t("topbar.collapse")}
        onClick={app.toggleCollapsed}
      >
        <PanelLeft />
      </Button>

      <InputGroup className="max-w-sm">
        <InputGroupAddon>
          <Search />
        </InputGroupAddon>
        <InputGroupInput placeholder={t("topbar.searchPlaceholder")} />
      </InputGroup>

      <div className="flex-1" />

      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" aria-label={t("topbar.help")}>
            <CircleHelp />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t("topbar.help")}</TooltipContent>
      </Tooltip>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon" className="relative" aria-label={t("topbar.notifications")}>
            <Bell />
            <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-info" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t("topbar.notifications")}</TooltipContent>
      </Tooltip>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button title={person} className="rounded-full outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
            <Avatar className="size-8">
              <AvatarFallback className="bg-primary text-xs font-medium text-primary-foreground">
                {initials}
              </AvatarFallback>
            </Avatar>
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-64">
          <div className="flex items-center gap-3 p-2">
            <Avatar className="size-9">
              <AvatarFallback className="bg-primary text-xs font-medium text-primary-foreground">
                {initials}
              </AvatarFallback>
            </Avatar>
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold">{person}</div>
              <div className="truncate text-xs text-muted-foreground">{email}</div>
            </div>
          </div>

          <DropdownMenuSeparator />

          {/* Tenant scope switcher — a submenu, per the product prototype.
              Exactly one tenant is always active. */}
          <DropdownMenuSub>
            <DropdownMenuSubTrigger className="flex-col items-start gap-0.5">
              <span className="text-xs text-muted-foreground">{t("topbar.tenant")}</span>
              <span className="truncate text-sm">
                {currentTenant?.name || app.tenant || t("topbar.noTenants")}
              </span>
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent className="w-56">
              {tenantOptions.length ? (
                <DropdownMenuRadioGroup value={app.tenant} onValueChange={app.setTenant}>
                  {tenantOptions.map((tn) => (
                    <DropdownMenuRadioItem key={tn.id} value={tn.id}>
                      <span className="flex-1 truncate">{tn.name}</span>
                      <span className="ml-2 text-xs text-muted-foreground">{tn.note}</span>
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              ) : (
                <div className="px-2 py-1.5 text-xs text-muted-foreground">{t("topbar.noTenants")}</div>
              )}
            </DropdownMenuSubContent>
          </DropdownMenuSub>

          <DropdownMenuSeparator />

          <div className="flex items-center justify-between px-2 py-1.5">
            <span className="text-sm text-muted-foreground">{t("topbar.language")}</span>
            <ToggleGroup
              type="single"
              size="sm"
              value={app.lang}
              onValueChange={(v) => onLang(v as Lang)}
            >
              <ToggleGroupItem value="zh">中文</ToggleGroupItem>
              <ToggleGroupItem value="en">EN</ToggleGroupItem>
            </ToggleGroup>
          </div>
          <div className="flex items-center justify-between px-2 py-1.5">
            <span className="text-sm text-muted-foreground">{t("topbar.theme")}</span>
            <ToggleGroup
              type="single"
              size="sm"
              value={app.theme}
              onValueChange={(v) => onTheme(v as ThemePref)}
            >
              <ToggleGroupItem value="light">{themeNames.light}</ToggleGroupItem>
              <ToggleGroupItem value="dark">{themeNames.dark}</ToggleGroupItem>
              <ToggleGroupItem value="system">{themeNames.system}</ToggleGroupItem>
            </ToggleGroup>
          </div>

          <DropdownMenuSeparator />

          <button
            onClick={onLogout}
            className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-destructive transition-colors hover:bg-destructive/10"
          >
            <LogOut className="size-4" />
            {t("topbar.logout")}
          </button>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
