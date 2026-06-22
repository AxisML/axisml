import { useEffect, Suspense } from "react";
import { Outlet } from "react-router-dom";
import { useApp } from "@/app/store";
import { useTenantOptions } from "@/api/hooks";
import { Spinner } from "@/components/ui/spinner";
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
    <div className="flex h-screen overflow-hidden bg-background text-foreground">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="flex-1 overflow-auto">
          <Suspense
            fallback={
              <div className="grid h-full place-items-center py-24">
                <Spinner className="size-7 text-muted-foreground" />
              </div>
            }
          >
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
