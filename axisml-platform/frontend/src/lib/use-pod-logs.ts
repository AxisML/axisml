import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type * as sdk from "@/api/generated";
import { USE_MOCK } from "@/api/mock";

// Stream a pod's logs via the endpoint's SSE `follow=true` mode. The generated
// SDK returns parsed bodies (not streams), so we read the ReadableStream
// directly with fetch. Auth rides the bearer token (mirroring api/setup.ts) plus
// the tenant cookie already set by earlier SDK requests (credentials:"include").
// Server-Sent-Event frames are `data:`-prefixed lines separated by a blank line.
async function streamLogs(
  path: string,
  { signal, onChunk }: { signal: AbortSignal; onChunk: (text: string) => void },
): Promise<void> {
  const token = localStorage.getItem("axisml.token");
  const res = await fetch(path, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    credentials: "include",
    signal,
  });
  if (!res.ok || !res.body) throw new Error(`log stream failed (${res.status})`);
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const frames = buf.split("\n\n");
    buf = frames.pop() ?? ""; // keep the trailing partial frame
    for (const frame of frames) {
      const text = frame
        .split("\n")
        .filter((l) => l.startsWith("data:"))
        .map((l) => l.slice(5).replace(/^ /, ""))
        .join("\n");
      if (text) onChunk(text + "\n");
    }
  }
}

// Shared pod-logs data hook behind every detail page's log tab (run / service /
// workspace). Owns the logic those panes each duplicated: list the pods, auto-
// select the first, then fetch the selected pod's logs. Callers supply the
// (already error-unwrapped) `listPods`/`getLogs` calls, a query-key base, and an
// optional `streamPath` builder for real SSE follow.
export function usePodLogs({
  queryKey,
  listPods,
  getLogs,
  streamPath,
  enabled = true,
}: {
  queryKey: readonly unknown[];
  listPods: () => Promise<{ items?: sdk.Pod[] }>;
  getLogs: (pod: string) => Promise<string>;
  streamPath?: (pod: string) => string;
  enabled?: boolean;
}) {
  const [pod, setPod] = useState("");
  const [follow, setFollow] = useState(false);
  const [streamText, setStreamText] = useState("");
  const [streamError, setStreamError] = useState<Error | null>(null);

  const podsQ = useQuery({
    queryKey: [...queryKey, "pods"],
    enabled,
    queryFn: listPods,
  });
  const pods = podsQ.data?.items ?? [];

  useEffect(() => {
    if (!pod && pods.length) setPod(pods[0].name);
  }, [pod, pods]);

  // Real SSE follow needs a live backend; under VITE_USE_MOCK_API the mock can't
  // stream, so follow falls back to snapshot polling there.
  const sse = follow && !USE_MOCK && !!streamPath;

  const logsQ = useQuery({
    queryKey: [...queryKey, "logs", pod],
    enabled: enabled && pod !== "" && !sse,
    queryFn: () => getLogs(pod),
    refetchInterval: follow && !sse ? 3000 : false,
  });

  // Keep the builder in a ref so an inline `streamPath` arrow doesn't retrigger
  // the stream on every render; it only (re)connects on follow/pod changes.
  const streamPathRef = useRef(streamPath);
  streamPathRef.current = streamPath;
  useEffect(() => {
    if (!sse || !pod) return;
    const build = streamPathRef.current;
    if (!build) return;
    const ctrl = new AbortController();
    setStreamText("");
    setStreamError(null);
    streamLogs(build(pod), {
      signal: ctrl.signal,
      onChunk: (c) => setStreamText((prev) => prev + c),
    }).catch((e) => {
      if (!ctrl.signal.aborted) setStreamError(e instanceof Error ? e : new Error(String(e)));
    });
    return () => ctrl.abort();
  }, [sse, pod]);

  const text = sse ? streamText : logsQ.data;
  return { pods, pod, setPod, follow, setFollow, podsQ, logsQ, text, streaming: sse, streamError };
}

export type PodLogsState = ReturnType<typeof usePodLogs>;
