import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import * as sdk from "../generated";
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
} from "../generated";

// Thin react-query wrappers over the generated SDK. Every list page calls one of
// these so backend communication always flows through the api-spec-generated
// client. Pages fall back to faithful demo content when the backend (currently a
// 501 contract-only shell) returns nothing — see each page's FALLBACK constants.

type SdkResult<T> = Promise<{ data?: T; error?: unknown }>;

function useApi<T>(key: unknown[], fn: () => SdkResult<T>): UseQueryResult<T> {
  return useQuery<T>({
    queryKey: key,
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
export const useTenants = () => useApi<ListTenantsResponse>(["tenants"], () => sdk.listTenants());
export const useResourcePools = () =>
  useApi<ListResourcePoolsResponse>(["resourcepools"], () => sdk.listResourcePools());
