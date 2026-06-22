import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type * as sdk from "@/api/generated";

// Shared pod-logs data hook behind every detail page's log tab (run / service /
// workspace). Owns the logic those panes each duplicated: list the pods, auto-
// select the first, then fetch the selected pod's log snapshot on demand. Callers
// supply the (already error-unwrapped) `listPods`/`getLogs` calls and a query-key
// base; the hook adds the "pods"/"logs" suffixes.
export function usePodLogs({
  queryKey,
  listPods,
  getLogs,
  enabled = true,
}: {
  queryKey: readonly unknown[];
  listPods: () => Promise<{ items?: sdk.Pod[] }>;
  getLogs: (pod: string) => Promise<string>;
  enabled?: boolean;
}) {
  const [pod, setPod] = useState("");
  const [follow, setFollow] = useState(false);

  const podsQ = useQuery({
    queryKey: [...queryKey, "pods"],
    enabled,
    queryFn: listPods,
  });
  const pods = podsQ.data?.items ?? [];

  useEffect(() => {
    if (!pod && pods.length) setPod(pods[0].name);
  }, [pod, pods]);

  const logsQ = useQuery({
    queryKey: [...queryKey, "logs", pod],
    enabled: enabled && pod !== "",
    queryFn: () => getLogs(pod),
    refetchInterval: follow ? 3000 : false,
  });

  return { pods, pod, setPod, follow, setFollow, podsQ, logsQ };
}

export type PodLogsState = ReturnType<typeof usePodLogs>;
