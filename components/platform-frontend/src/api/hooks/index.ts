import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import * as sdk from "../generated";
import { useApp } from "@/app/store";
import { useSession } from "@/app/session";
import type {
  ListJobsResponse,
  ListExperimentsResponse,
  ListWorkspacesResponse,
  ListMlServicesResponse,
  ListTrafficPoliciesResponse,
  ListModelDefinitionsResponse,
  ListImageDefinitionsResponse,
  ListTenantsResponse,
  ListResourcePoolsResponse,
  ListModelVersionsResponse,
  ListImageVersionsResponse,
} from "../generated";

// Thin react-query wrappers over the generated SDK. Every list page calls one of
// these so backend communication always flows through the api-spec-generated
// client, scoped to the active tenant. There is no demo fallback — pages render
// real loading/error/empty states (see src/components/states.tsx).

type SdkResult<T> = Promise<{ data?: T; error?: unknown }>;

export interface TenantOption {
  id: string;
  name: string;
  note: string;
}

// Tenant scope options for the switcher / default selection. A system-admin can
// scope to any tenant (list them all); everyone else is limited to their real
// memberships from the session. Shared queryKey ⇒ cached across consumers.
export function useTenantOptions(): TenantOption[] {
  const { isSystemAdmin, me } = useSession();
  const adminQ = useQuery({
    queryKey: ["topbar", "tenants"],
    queryFn: async () => (await sdk.listTenants()).data,
    enabled: isSystemAdmin,
  });
  if (isSystemAdmin) {
    return (adminQ.data?.items ?? []).map((t) => ({
      id: t.identifier,
      name: t.displayName || t.identifier,
      note: t.identifier,
    }));
  }
  return (me?.tenantRoles ?? []).map((t) => ({
    id: t.tenantName,
    name: t.tenantName,
    note: t.roleName,
  }));
}

function useApi<T>(
  key: unknown[],
  fn: () => SdkResult<T>,
  opts: { scoped?: boolean } = {},
): UseQueryResult<T> {
  const { scoped = true } = opts;
  // Scope every tenant-partitioned query by the active tenant so switching tenant
  // in the topbar refetches with the new X-Axisml-Tenant header (api/setup.ts).
  // Scoped queries wait until a tenant is selected (avoids a pre-bootstrap 400);
  // global resources (tenants, resource pools) always run.
  const { tenant } = useApp();
  return useQuery<T>({
    queryKey: [...key, tenant],
    enabled: scoped ? tenant !== "" : true,
    queryFn: async () => {
      const { data, error } = await fn();
      if (error) throw error;
      return data as T;
    },
  });
}

export const useJobs = () => useApi<ListJobsResponse>(["jobs"], () => sdk.listJobs());
export const useExperiments = () => useApi<ListExperimentsResponse>(["experiments"], () => sdk.listExperiments());
export const useWorkspaces = () => useApi<ListWorkspacesResponse>(["workspaces"], () => sdk.listWorkspaces());
export const useServices = () => useApi<ListMlServicesResponse>(["mlservices"], () => sdk.listMlServices());
export const useTrafficPolicies = () =>
  useApi<ListTrafficPoliciesResponse>(["trafficpolicies"], () => sdk.listTrafficPolicies());
export const useModels = () => useApi<ListModelDefinitionsResponse>(["models"], () => sdk.listModelDefinitions());
export const useImages = () => useApi<ListImageDefinitionsResponse>(["images"], () => sdk.listImageDefinitions());
// The tenant-management table needs live workload roll-ups (member / active-run
// / online-service counts), so it opts into ?stats=true. The topbar switcher
// uses useTenantOptions (a separate, count-free listTenants) to stay cheap.
export const useTenants = () =>
  useApi<ListTenantsResponse>(["tenants", "stats"], () => sdk.listTenants({ query: { stats: true } }), {
    scoped: false,
  });
export const useResourcePools = () =>
  useApi<ListResourcePoolsResponse>(["resourcepools"], () => sdk.listResourcePools(), { scoped: false });

// Versions for a single model, fetched on demand (e.g. when a model is selected
// in the create-service form). Disabled until a model name is chosen.
export function useModelVersions(name: string): UseQueryResult<ListModelVersionsResponse> {
  const { tenant } = useApp();
  return useQuery<ListModelVersionsResponse>({
    queryKey: ["models", tenant, name, "versions"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listModelVersions({ path: { tenant, name } });
      if (error) throw error;
      return data as ListModelVersionsResponse;
    },
  });
}

// Versions for a single image, fetched on demand. Disabled until a name is chosen.
export function useImageVersions(name: string): UseQueryResult<ListImageVersionsResponse> {
  const { tenant } = useApp();
  return useQuery<ListImageVersionsResponse>({
    queryKey: ["images", tenant, name, "versions"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listImageVersions({ path: { tenant, name } });
      if (error) throw error;
      return data as ListImageVersionsResponse;
    },
  });
}
