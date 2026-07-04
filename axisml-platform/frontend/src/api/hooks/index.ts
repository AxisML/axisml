import { useQuery, useInfiniteQuery, type UseQueryResult } from "@tanstack/react-query";
import * as sdk from "../generated";
import { unitSpecLine } from "@/lib/units";
import { useApp } from "@/app/store";
import { useSession } from "@/app/session";
import type {
  ListJobsResponse,
  ListExperimentsResponse,
  ListWorkspacesResponse,
  ListMlServicesResponse,
  ListModelDefinitionsResponse,
  ListImageDefinitionsResponse,
  ListResourcePoolsResponse,
  ListDataVolumesResponse,
  ListStorageClassesResponse,
  ListModelVersionsResponse,
  ListImageVersionsResponse,
  ListActivityResponse,
  ListWorkspaceImagesResponse,
  ClusterUsage,
  MetricSeries,
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

// Some endpoints are declared in the OpenAPI contract but not yet implemented by
// the backend — they answer 501 (not-implemented). For auxiliary roll-up /
// aggregate data we treat a 501 as "no data yet" and return an empty payload so
// the page renders its honest empty/pending state — never fake data. Any other
// error still surfaces. Under VITE_USE_MOCK_API the mock router answers these, so
// the data is populated. See api/setup.ts (the 401 interceptor is separate).
function isNotImplemented(err: unknown): boolean {
  const e = err as { status?: number; code?: string } | null | undefined;
  return e?.status === 501 || e?.code === "not-implemented";
}

// useAux mirrors useApi but degrades a 501 to `empty` instead of an error state.
function useAux<T>(
  key: unknown[],
  fn: () => SdkResult<T>,
  empty: T,
  opts: { scoped?: boolean } = {},
): UseQueryResult<T> {
  const { scoped = true } = opts;
  const { tenant } = useApp();
  return useQuery<T>({
    queryKey: [...key, tenant],
    enabled: scoped ? tenant !== "" : true,
    queryFn: async () => {
      const { data, error } = await fn();
      if (error) {
        if (isNotImplemented(error)) return empty;
        throw error;
      }
      return data as T;
    },
  });
}

// Dashboard aggregates. Cluster usage/metrics are cluster-global (unscoped); the
// activity feed is tenant-scoped.
export const useClusterUsage = (pool: string) =>
  useAux<ClusterUsage>(
    ["cluster-usage", pool],
    () => sdk.getClusterUsage(pool && pool !== "all" ? { query: { pool } } : {}),
    { aggregate: [], pools: [], updatedAt: "" },
    { scoped: false },
  );

export const useClusterMetric = (metric: "gpu_util" | "gpu_quota", pool: string, range = "24h") =>
  useAux<MetricSeries>(
    ["cluster-metric", metric, pool, range],
    () => sdk.getClusterMetrics({ query: { metric, range, ...(pool && pool !== "all" ? { pool } : {}) } }),
    { metric, range, series: [] },
    { scoped: false },
  );

export const useActivity = () =>
  useAux<ListActivityResponse>(["activity"], () => sdk.listActivity(), { count: 0, items: [] });

export const useWorkspaceImages = () =>
  useAux<ListWorkspaceImagesResponse>(["workspace-images"], () => sdk.listWorkspaceImages(), { count: 0, items: [] }, {
    scoped: false,
  });

// A page of any list endpoint: items + the opaque continue token for the next.
interface ListPage<T> {
  items?: T[];
  count?: number;
  continueToken?: string;
  partial?: boolean;
}

// Server-side paginated + filtered list. `key` must include every filter value
// so changing a filter refetches from page 1; `fetchPage` receives {limit,
// continue} and folds in the caller's filter query. Pages are flattened and
// accumulated; `loadMore` fetches the next continue token.
export function usePagedList<T>(
  key: unknown[],
  fetchPage: (page: { limit: number; continue?: string }) => SdkResult<ListPage<T>>,
  opts: { scoped?: boolean; pageSize?: number } = {},
) {
  const { scoped = true, pageSize = 50 } = opts;
  const { tenant } = useApp();
  const q = useInfiniteQuery({
    queryKey: [...key, tenant],
    enabled: scoped ? tenant !== "" : true,
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const { data, error } = await fetchPage({ limit: pageSize, continue: pageParam });
      if (error) throw error;
      return data as ListPage<T>;
    },
    getNextPageParam: (last) => last.continueToken || undefined,
  });
  const items = q.data?.pages.flatMap((p) => p.items ?? []) ?? [];
  return {
    items,
    isLoading: q.isLoading,
    isError: q.isError,
    error: q.error,
    hasMore: !!q.hasNextPage,
    loadMore: () => void q.fetchNextPage(),
    isFetchingMore: q.isFetchingNextPage,
    refetch: () => void q.refetch(),
  };
}

export type PagedListState<T> = ReturnType<typeof usePagedList<T>>;

export const useJobs = () => useApi<ListJobsResponse>(["jobs"], () => sdk.listJobs());
export const useExperiments = () => useApi<ListExperimentsResponse>(["experiments"], () => sdk.listExperiments());
export const useWorkspaces = () => useApi<ListWorkspacesResponse>(["workspaces"], () => sdk.listWorkspaces());
export const useServices = () => useApi<ListMlServicesResponse>(["mlservices"], () => sdk.listMlServices());
export const useModels = () => useApi<ListModelDefinitionsResponse>(["models"], () => sdk.listModelDefinitions());
export const useImages = () => useApi<ListImageDefinitionsResponse>(["images"], () => sdk.listImageDefinitions());
export const useResourcePools = () =>
  useApi<ListResourcePoolsResponse>(["resourcepools"], () => sdk.listResourcePools(), { scoped: false });

// Pool + per-pool unit option lists for the create-form pickers (jobs /
// experiments / workspaces). Units are embedded in the pool list response, so
// this needs no extra call — it replaces the drawers' hardcoded sample catalogs.
export function usePoolUnitOptions() {
  const q = useResourcePools();
  const items = q.data?.items ?? [];
  const pools = items.map((p) => ({
    value: p.name,
    label: p.description ? `${p.name} · ${p.description}` : p.name,
  }));
  const unitsFor = (pool: string) => {
    const p = items.find((x) => x.name === pool);
    return (p?.units ?? []).map((u) => ({ value: u.name, title: u.name, desc: unitSpecLine(u) }));
  };
  return { pools, unitsFor, isLoading: q.isLoading, isError: q.isError };
}
// Data volumes are tenant-scoped (system-admin managed): the query carries the
// active tenant and refetches on switch, like the other tenant-partitioned lists.
export const useDataVolumes = () =>
  useApi<ListDataVolumesResponse>(["datavolumes"], () => sdk.listDataVolumes());

// Shared volume → Select-option mapping for the mount pickers in the workspace /
// job / experiment create drawers, plus the query's loading/error flags so each
// drawer can surface the real state instead of masking a failed list as empty.
export function useVolumeOptions() {
  const q = useDataVolumes();
  const options = (q.data?.items ?? []).map((vol) => ({
    value: vol.name,
    label: vol.size ? `${vol.name} · ${vol.size}` : vol.name,
  }));
  return { options, isLoading: q.isLoading, isError: q.isError };
}
// Storage classes are cluster-global (not tenant-partitioned); used by the
// data-volume create form.
export const useStorageClasses = () =>
  useApi<ListStorageClassesResponse>(["storageclasses"], () => sdk.listStorageClasses(), { scoped: false });

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
