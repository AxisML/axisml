import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import { useApp } from "@/app/store";
import { useTenantOptions } from "@/api/hooks";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

export function AppShell() {
  const { collapsed, tenant, setTenant } = useApp();
  // No "all-tenants" view: ensure a tenant is always selected by defaulting to
  // the user's first available tenant once the options load.
  const tenantOptions = useTenantOptions();
  useEffect(() => {
    if (!tenant && tenantOptions.length) setTenant(tenantOptions[0].id);
  }, [tenant, tenantOptions, setTenant]);
  return (
    <div className={"app-shell" + (collapsed ? " collapsed" : "")}>
      <Sidebar />
      <div className="app-main">
        <Topbar />
        <Outlet />
      </div>
    </div>
  );
}
