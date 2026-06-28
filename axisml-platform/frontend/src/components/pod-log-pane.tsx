import { useTranslation } from "react-i18next";
import { Card, CardContent } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LogViewer } from "@/components/log-viewer";
import { PageLoading, DetailError } from "@/components/page-state";
import { errorText } from "@/lib/errors";
import type { PodLogsState } from "@/lib/use-pod-logs";

// Canonical pod-logs pane, mirroring the prototype's log tab: a `log-bar` with a
// "POD"-tagged pod picker on the left and a live-follow toggle on the right, over
// the shared dark LogViewer. Replaces the divergent log panes (which disagreed on
// header layout, refresh affordance, and empty state). Drive it with `usePodLogs`.
export function PodLogPane({
  logs,
  emptyText,
}: {
  logs: PodLogsState;
  emptyText: string;
}) {
  const { t } = useTranslation();
  const { pods, pod, setPod, follow, setFollow, podsQ, logsQ } = logs;
  return (
    <Card>
      <CardContent>
        <div className="mb-4 flex items-center gap-3">
          <div className="inline-flex items-center gap-2 rounded-md border bg-background py-1 pr-2.5 pl-1.5">
            <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-[10px] font-bold tracking-wider text-muted-foreground">
              {t("common.podLabel")}
            </span>
            <Select value={pod || undefined} onValueChange={setPod} disabled={!pods.length}>
              <SelectTrigger
                size="sm"
                className="h-auto min-w-44 border-0 bg-transparent px-0 shadow-none focus-visible:ring-0"
              >
                <SelectValue placeholder={emptyText} />
              </SelectTrigger>
              <SelectContent>
                {pods.map((p) => (
                  <SelectItem key={p.name} value={p.name}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grow" />
          <label className="inline-flex cursor-pointer items-center gap-2 text-xs text-muted-foreground">
            {t("common.follow")}
            <Switch
              checked={follow}
              onCheckedChange={setFollow}
              disabled={!pods.length}
              aria-label={t("common.follow")}
            />
          </label>
        </div>
        {podsQ.isLoading || logsQ.isLoading ? (
          <PageLoading className="py-16" />
        ) : podsQ.isError ? (
          // Surface the fetch failure rather than masking it as "no logs".
          <DetailError message={errorText(podsQ.error)} />
        ) : !pods.length ? (
          <DetailError message={emptyText} />
        ) : logsQ.isError ? (
          <DetailError message={errorText(logsQ.error)} />
        ) : (
          <LogViewer text={logsQ.data} empty={emptyText} />
        )}
      </CardContent>
    </Card>
  );
}
