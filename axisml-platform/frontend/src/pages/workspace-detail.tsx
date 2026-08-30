import { useState, type ReactNode } from "react";
import { useParams } from "react-router-dom";
import { Play, Power, Trash2, Copy, Code2, Info } from "lucide-react";
import { JupyterMark, VscodeMark } from "@/components/workspace-brand";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { PodLogPane } from "@/components/pod-log-pane";
import { usePodLogs } from "@/lib/use-pod-logs";
import { BackLink } from "@/components/back-link";
import { MonoChip } from "@/components/mono-chip";
import { Descriptions, Desc } from "@/components/descriptions";
import { PageLoading, DetailError } from "@/components/page-state";
import { fmtDateTime, fmtDateTimeSec } from "@/lib/format";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardAction,
} from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { FormDrawer } from "@/components/form-drawer";
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const isRunning = (phase?: string) =>
  phase === "Queued" || phase === "Running" || phase === "Degraded" || phase === "Starting" || phase === "Creating" || phase === "Pending";

function useWorkspace(name: string) {
  const { tenant } = useApp();
  return useQuery<sdk.Workspace>({
    queryKey: ["workspaces", tenant, name],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getWorkspace({ path: { name } });
      if (error) throw error;
      return data as sdk.Workspace;
    },
  });
}

// A per-tool launch button: opens `url` in a new tab, disabled when absent.
function LaunchButton({ url, mark, label }: { url?: string; mark: ReactNode; label: string }) {
  return (
    <Button variant="outline" asChild={!!url} disabled={!url}>
      {url ? (
        <a href={url} target="_blank" rel="noreferrer">
          {mark}
          {label}
        </a>
      ) : (
        <>
          {mark}
          {label}
        </>
      )}
    </Button>
  );
}

export default function WorkspaceDetail() {
  const { name = "" } = useParams();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const q = useWorkspace(name);
  const [edit, setEdit] = useState(false);

  const start = useApiMutation((n: string) => sdk.startWorkspace({ path: { name: n } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.starting"),
  });
  const stop = useApiMutation((n: string) => sdk.stopWorkspace({ path: { name: n } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.stopped"),
  });
  const del = useApiMutation((name: string) => sdk.deleteWorkspace({ path: { name } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.deleted"),
  });

  const backLink = <BackLink to="/workspaces">{t("workspaces.backToList")}</BackLink>;

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]} title={name} subtitle={backLink}>
        <PageLoading />
      </PageContainer>
    );
  }
  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]} title={name} subtitle={backLink}>
        <DetailError message={t("workspaces.notFound")} />
      </PageContainer>
    );
  }

  const w = q.data;
  const running = isRunning(w.phase);

  const onDelete = () => {
    confirm({
      title: t("workspaces.deleteTitle", { name: w.name }),
      desc: running ? t("workspaces.deleteDescRunning") : t("workspaces.deleteDescStopped"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(w.name),
    });
  };

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]}
      title={
        <span className="inline-flex items-center gap-3">
          <span className="font-mono">{w.name}</span>
          <PhaseTag phase={w.phase} />
        </span>
      }
      subtitle={backLink}
      extra={
        <div className="flex items-center gap-2">
          {running && (
            <>
              <LaunchButton
                url={w.endpoint?.tools?.find((x) => x.name === "jupyter")?.url ?? w.endpoint?.accessUrl}
                mark={<JupyterMark data-icon="inline-start" className="size-[15px]" />}
                label={t("workspaces.openJupyter")}
              />
              <LaunchButton
                url={w.endpoint?.tools?.find((x) => x.name === "vscode")?.url ?? w.endpoint?.accessUrl}
                mark={<VscodeMark data-icon="inline-start" className="size-[15px]" />}
                label={t("workspaces.openVscode")}
              />
            </>
          )}
          {running ? (
            <Button variant="outline" onClick={() => stop.mutate(w.name)} disabled={stop.isPending}>
              {stop.isPending ? <Spinner data-icon="inline-start" /> : <Power data-icon="inline-start" />}
              {t("phase.Stopped")}
            </Button>
          ) : (
            <Button onClick={() => start.mutate(w.name)} disabled={start.isPending}>
              {start.isPending ? <Spinner data-icon="inline-start" /> : <Play data-icon="inline-start" className="fill-current" />}
              {t("workspaces.start")}
            </Button>
          )}
          <Button variant="outline" className="text-destructive" onClick={onDelete}>
            <Trash2 data-icon="inline-start" />
            {t("common.delete")}
          </Button>
        </div>
      }
    >
      <Tabs defaultValue="info">
        <TabsList>
          <TabsTrigger value="info">{t("workspaces.tabInfo")}</TabsTrigger>
          <TabsTrigger value="log">{t("workspaces.tabLog")}</TabsTrigger>
          <TabsTrigger value="ev">{t("workspaces.tabEvents")}</TabsTrigger>
        </TabsList>
        <TabsContent value="info" className="mt-4">
          <InfoPane w={w} running={running} onEdit={() => setEdit(true)} />
        </TabsContent>
        <TabsContent value="log" className="mt-4">
          <LogPane name={w.name} />
        </TabsContent>
        <TabsContent value="ev" className="mt-4">
          <EventsPane name={w.name} />
        </TabsContent>
      </Tabs>
      {edit && <EditDrawer w={w} onClose={() => setEdit(false)} />}
    </PageContainer>
  );
}

function InfoPane({ w, running, onEdit }: { w: sdk.Workspace; running: boolean; onEdit: () => void }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const accessUrl = w.endpoint?.accessUrl;
  const vol = w.volumes?.[0];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("workspaces.configTitle")}</CardTitle>
        {running && (
          <CardAction>
            <Button variant="outline" size="sm" onClick={onEdit}>
              {t("common.edit")}
            </Button>
          </CardAction>
        )}
      </CardHeader>
      <CardContent>
        <Descriptions columns="single">
          <Desc label={t("common.name")}>
            <MonoChip>{w.name}</MonoChip>
          </Desc>
          {(w.description || w.displayName) && (
            <Desc label={t("common.description")}>{w.description ?? w.displayName}</Desc>
          )}
          <Desc label={t("workspaces.fPool")}>
            {w.poolName ? <MonoChip>{w.poolName}</MonoChip> : <span className="text-muted-foreground">—</span>}
          </Desc>
          <Desc label={t("workspaces.fUnit")}>
            {w.unitName ? <MonoChip>{w.unitName}</MonoChip> : <span className="text-muted-foreground">—</span>}
          </Desc>
          <Desc label={t("workspaces.fImage")}>
            <MonoChip>{w.image}</MonoChip>
          </Desc>
          <Desc label={t("workspaces.fPort")}>
            <span className="font-mono">{w.containerPort}</span>
          </Desc>
          {accessUrl && (
            <Desc label={t("workspaces.fAccessUrl")}>
              <span className="inline-flex items-center gap-1.5">
                <MonoChip>{accessUrl}</MonoChip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => {
                        void navigator.clipboard?.writeText(accessUrl);
                        toast(t("workspaces.addrCopied"));
                      }}
                    >
                      <Copy />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("workspaces.copyAddr")}</TooltipContent>
                </Tooltip>
              </span>
            </Desc>
          )}
          {w.endpoint?.internalDns && (
            <Desc label={t("workspaces.fInternalDns")}>
              <MonoChip>{w.endpoint.internalDns}</MonoChip>
            </Desc>
          )}
          <Desc label={t("workspaces.fVolume")}>
            {vol ? (
              <span className="inline-flex items-center gap-2">
                <MonoChip>{vol.name ?? vol.mountPath}</MonoChip>
                {vol.used && <span className="text-sm text-muted-foreground">{vol.used}</span>}
              </span>
            ) : (
              <span className="text-muted-foreground">{t("workspaces.noVolume")}</span>
            )}
          </Desc>
          {vol?.mountPath && (
            <Desc label={t("workspaces.fMountPath")}>
              <MonoChip>{vol.mountPath}</MonoChip>
            </Desc>
          )}
          <Desc label={t("workspaces.fEnv")}>
            {w.env?.length ? (
              <div className="flex flex-wrap gap-1.5">
                {w.env.map((e) => (
                  <MonoChip key={e.name}>
                    {e.name}={e.value}
                  </MonoChip>
                ))}
              </div>
            ) : (
              <span className="text-muted-foreground">{t("workspaces.noEnv")}</span>
            )}
          </Desc>
          <Desc label={t("common.creator")}>
            {w.owner} ·{" "}
            <span className="font-mono text-muted-foreground">{fmtDateTime(w.createdAt)}</span>
          </Desc>
        </Descriptions>
      </CardContent>
    </Card>
  );
}

function LogPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const logs = usePodLogs({
    queryKey: ["workspaces", tenant, name],
    enabled: tenant !== "" && name !== "",
    listPods: async () => {
      const { data, error } = await sdk.listWorkspacePods({ path: { name } });
      if (error) throw error;
      return data;
    },
    getLogs: async (pod) => {
      const { data, error } = await sdk.getWorkspacePodLogs({ path: { name, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
    streamPath: (pod) => `/api/v1/workspaces/${name}/pods/${encodeURIComponent(pod)}/logs?follow=true`,
  });
  return <PodLogPane logs={logs} emptyText={t("workspaces.logHint")} />;
}

function EventsPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const q = useQuery<sdk.EventList>({
    queryKey: ["workspaces", tenant, name, "events"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getWorkspaceEvents({ path: { name } });
      if (error) throw error;
      return data as sdk.EventList;
    },
  });

  if (q.isLoading) {
    return (
      <Card>
        <CardContent>
          <div className="grid place-items-center py-10">
            <Spinner className="size-7 text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    );
  }
  const items = q.data?.items ?? [];
  if (q.isError || items.length === 0) {
    return (
      <Card>
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{q.isError ? t("common.loadFailed") : t("workspaces.noEvents")}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      </Card>
    );
  }
  return (
    <Card>
      <CardContent>
        <div className="flex flex-col gap-4">
          {items.map((e, i) => (
            <div key={i} className="flex gap-3">
              <div className="flex flex-col items-center pt-1.5">
                <span
                  className={
                    "size-2 shrink-0 rounded-full " +
                    (e.type === "Warning" ? "bg-warning" : "bg-info")
                  }
                />
                {i < items.length - 1 && <span className="mt-1 w-px flex-1 bg-border" />}
              </div>
              <div className="min-w-0 pb-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium text-foreground">{e.reason}</span>
                  <Badge variant={e.type === "Warning" ? "warning" : "secondary"}>{e.type}</Badge>
                  <span className="font-mono text-xs text-muted-foreground">
                    {fmtDateTimeSec(e.lastTimestamp)}
                  </span>
                </div>
                <div className="mt-0.5 text-sm text-muted-foreground">{e.message}</div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

interface EditFormValues {
  displayName: string;
  description: string;
}

function EditDrawer({ w, onClose }: { w: sdk.Workspace; onClose: () => void }) {
  const { t } = useTranslation();
  const [v, setV] = useState<EditFormValues>({
    displayName: w.displayName ?? w.name,
    description: w.description ?? "",
  });
  const set = <K extends keyof EditFormValues>(k: K, val: EditFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const update = useApiMutation(
    (vars: { name: string; body: sdk.WorkspacePatchRequest }) =>
      sdk.updateWorkspace({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["workspaces"]], success: t("workspaces.updated") },
  );

  const submit = () => {
    update.mutate(
      {
        name: w.name,
        body: { displayName: v.displayName?.trim() || undefined, description: v.description?.trim() || undefined },
      },
      { onSuccess: onClose },
    );
  };

  return (
    <FormDrawer
      title={t("workspaces.drawerEdit")}
      subtitle={<span className="font-mono">{w.name}</span>}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("workspaces.saveChanges")}
      submitting={update.isPending}
    >
      <Alert variant="info" className="mb-4">
        <Info />
        <AlertDescription>{t("workspaces.editNotice")}</AlertDescription>
      </Alert>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="ws-edit-name">{t("workspaces.fName")}</FieldLabel>
          <Input
            id="ws-edit-name"
            value={v.displayName}
            onChange={(e) => set("displayName", e.target.value)}
          />
          <FieldDescription>{t("workspaces.fNameHelp")}</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="ws-edit-desc">{t("workspaces.fDesc")}</FieldLabel>
          <Textarea
            id="ws-edit-desc"
            rows={2}
            placeholder={t("workspaces.fDescPlaceholder")}
            value={v.description}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>
      </FieldGroup>
      <div className="mt-5 rounded-lg border bg-muted p-3 text-sm text-muted-foreground">
        <Code2 className="mr-2 inline size-4 text-foreground" />
        {t("workspaces.fImage")}: <span className="font-mono text-foreground">{w.image}</span>
        {w.unitName && (
          <>
            {" · "}
            {t("workspaces.fUnit")}: <span className="font-mono text-foreground">{w.unitName}</span>
          </>
        )}
      </div>
    </FormDrawer>
  );
}
