import { useEffect, Suspense } from "react";
import { Layout, Spin } from "antd";
import { Outlet } from "react-router-dom";
import { useApp } from "@/app/store";
import { useTenantOptions } from "@/api/hooks";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

export function AppShell() {
  const { tenant, setTenant } = useApp();
  // No "all-tenants" view: ensure a tenant is always selected by defaulting to
  // the user's first available tenant once the options load.
  const tenantOptions = useTenantOptions();
  useEffect(() => {
    if (!tenant && tenantOptions.length) setTenant(tenantOptions[0].id);
  }, [tenant, tenantOptions, setTenant]);

  return (
    <Layout className="h-screen">
      <Sidebar />
      <Layout className="min-w-0">
        <Topbar />
        <Layout.Content className="overflow-auto bg-surface">
          <Suspense fallback={<div className="grid h-full place-items-center py-24"><Spin size="large" /></div>}>
            <Outlet />
          </Suspense>
        </Layout.Content>
      </Layout>
    </Layout>
  );
}
