import { useEffect, Suspense } from "react";
import { Outlet } from "react-router-dom";
import { useApp } from "@/app/store";
import { useTenantOptions } from "@/api/hooks";
import { Spinner } from "@/components/ui/spinner";
import { Sidebar } from "./sidebar";
import { Topbar } from "./topbar";

export function AppShell() {
  const { tenant, setTenant } = useApp();
  // No "all-tenants" view: ensure a valid tenant is always selected. Default to
  // the user's first available tenant when none is set OR when the persisted
  // tenant is no longer one of the caller's memberships (revoked access, tenant
  // deleted, or a different user on the same browser) — otherwise every scoped
  // query would 403/400 against a stale id with no way back but a manual switch.
  const tenantOptions = useTenantOptions();
  useEffect(() => {
    if (!tenantOptions.length) return; // still loading or genuinely none
    const valid = tenantOptions.some((o) => o.id === tenant);
    if (!valid) setTenant(tenantOptions[0].id);
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
