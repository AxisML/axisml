import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useUI } from "@/app/ui";
import { errorText } from "@/components/states";

type SdkResult<T> = Promise<{ data?: T; error?: unknown }>;

// Shared write-path wrapper over the generated SDK. Every create/update/delete/
// action flows through here so it: surfaces backend errors as a toast (no silent
// failures), shows a success toast, and invalidates the affected list queries so
// the UI reflects the new server state. Invalidation uses key PREFIXES — e.g.
// ["resourcepools"] matches the tenant-scoped ["resourcepools", <tenant>] keys.
export function useApiMutation<TArgs, TData>(
  fn: (args: TArgs) => SdkResult<TData>,
  opts: { invalidate?: unknown[][]; success?: string } = {},
) {
  const qc = useQueryClient();
  const { toast } = useUI();
  return useMutation<TData, Error, TArgs>({
    mutationFn: async (args: TArgs) => {
      const { data, error } = await fn(args);
      if (error) throw new Error(errorText(error));
      return data as TData;
    },
    onSuccess: () => {
      opts.invalidate?.forEach((key) => qc.invalidateQueries({ queryKey: key }));
      if (opts.success) toast(opts.success);
    },
    onError: (e) => toast(e instanceof Error ? e.message : "操作失败"),
  });
}
