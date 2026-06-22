import { useEffect, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Play,
  Power,
  Trash2,
  Copy,
  Code2,
  Info,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import { useApp } from "@/app/store";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { PhaseTag } from "@/components/PhaseTag";
import { LogViewer } from "@/components/LogViewer";
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
import { Switch } from "@/components/ui/switch";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Field, FieldLabel, FieldDescription } from "@/components/ui/field";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const isRunning = (phase?: string) =>
  phase === "Running" || phase === "Degraded" || phase === "Starting" || phase === "Creating" || phase === "Pending";

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
  const del = useApiMutation(
    (vars: { name: string; deletePvc: boolean }) =>
      sdk.deleteWorkspace({ path: { name: vars.name }, body: { deletePvc: vars.deletePvc } }),
    { invalidate: [["workspaces"]], success: t("workspaces.deleted") },
  );

  const back = (
    <Link
      to="/workspaces"
      className="mb-3 inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-info"
    >
      <ArrowLeft className="size-4" />
      {t("workspaces.backToList")}
    </Link>
  );

  if (q.isLoading) {
    return (
      <PageContainer breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]} title={name}>
        {back}
        <div className="grid place-items-center py-20">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      </PageContainer>
    );
  }
  if (q.isError || !q.data) {
    return (
      <PageContainer breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]} title={name}>
        {back}
        <Card>
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("workspaces.notFound")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        </Card>
      </PageContainer>
    );
  }

  const w = q.data;
  const running = isRunning(w.phase);
  const pvc = w.volumes?.find((v) => v.size)?.size;

  const onDelete = () => {
    let deletePvc = pvc != null;
    confirm({
      title: t("workspaces.deleteTitle", { name: w.name }),
      desc: running ? t("workspaces.deleteDescRunning") : t("workspaces.deleteDescStopped"),
      info:
        pvc != null ? (
          <label className="flex items-center gap-2">
            <input type="checkbox" defaultChecked onChange={(e) => (deletePvc = e.target.checked)} />
            {t("workspaces.deletePvc", { size: pvc })}
          </label>
        ) : undefined,
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate({ name: w.name, deletePvc }),
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
      subtitle={`${w.description ?? w.displayName ?? ""}${w.owner ? ` · ${t("common.creator")} ${w.owner}` : ""}`}
      extra={
        <div className="flex items-center gap-2">
          {running ? (
            <Button variant="outline" onClick={() => stop.mutate(w.name)} disabled={stop.isPending}>
              {stop.isPending ? <Spinner data-icon="inline-start" /> : <Power data-icon="inline-start" />}
              {t("phase.Stopped")}
            </Button>
          ) : (
            <Button onClick={() => start.mutate(w.name)} disabled={start.isPending}>
              {start.isPending ? <Spinner data-icon="inline-start" /> : <Play data-icon="inline-start" />}
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
      <div className="-mt-2">{back}</div>
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

function Chip({ children }: { children: ReactNode }) {
  return (
    <span className="rounded-md border bg-muted px-2 py-0.5 font-mono text-sm text-muted-foreground">
      {children}
    </span>
  );
}

function Row({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </>
  );
}

function InfoPane({ w, running, onEdit }: { w: sdk.Workspace; running: boolean; onEdit: () => void }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const accessUrl = w.endpoint?.accessUrl;
  const vol = w.volumes?.find((v) => v.size) ?? w.volumes?.[0];

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
        <dl className="grid grid-cols-[120px_1fr] items-baseline gap-x-4 gap-y-2.5 text-sm">
          <Row label={t("common.name")}>
            <Chip>{w.name}</Chip>
          </Row>
          {(w.description || w.displayName) && (
            <Row label={t("common.description")}>{w.description ?? w.displayName}</Row>
          )}
          <Row label={t("workspaces.fPool")}>
            {w.poolName ? <Chip>{w.poolName}</Chip> : <span className="text-muted-foreground">—</span>}
          </Row>
          <Row label={t("workspaces.fUnit")}>
            {w.unitName ? <Chip>{w.unitName}</Chip> : <span className="text-muted-foreground">—</span>}
          </Row>
          <Row label={t("workspaces.fImage")}>
            <Chip>{w.image}</Chip>
          </Row>
          <Row label={t("workspaces.fPort")}>
            <span className="font-mono">{w.containerPort}</span>
          </Row>
          {accessUrl && (
            <Row label={t("workspaces.fAccessUrl")}>
              <span className="inline-flex items-center gap-1.5">
                <Chip>{accessUrl}</Chip>
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
            </Row>
          )}
          {w.endpoint?.internalDns && (
            <Row label={t("workspaces.fInternalDns")}>
              <Chip>{w.endpoint.internalDns}</Chip>
            </Row>
          )}
          <Row label={t("workspaces.fVolume")}>
            {vol ? (
              <span className="inline-flex items-center gap-2">
                <Chip>{vol.name ?? vol.mountPath}</Chip>
                <span className="text-sm text-muted-foreground">
                  {[vol.size, vol.storageClass].filter(Boolean).join(" · ") || "—"}
                </span>
              </span>
            ) : (
              <span className="text-muted-foreground">{t("workspaces.noVolume")}</span>
            )}
          </Row>
          {vol?.mountPath && (
            <Row label={t("workspaces.fMountPath")}>
              <Chip>{vol.mountPath}</Chip>
            </Row>
          )}
          <Row label={t("workspaces.fEnv")}>
            {w.env?.length ? (
              <div className="flex flex-wrap gap-1.5">
                {w.env.map((e) => (
                  <Chip key={e.name}>
                    {e.name}={e.value}
                  </Chip>
                ))}
              </div>
            ) : (
              <span className="text-muted-foreground">{t("workspaces.noEnv")}</span>
            )}
          </Row>
          <Row label={t("common.creator")}>
            {w.owner} ·{" "}
            <span className="font-mono text-muted-foreground">
              {dayjs(w.createdAt).format("YYYY-MM-DD HH:mm")}
            </span>
          </Row>
        </dl>
      </CardContent>
    </Card>
  );
}

function LogPane({ name }: { name: string }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [follow, setFollow] = useState(true);
  const [pod, setPod] = useState<string>("");
  const podsQ = useQuery<sdk.PodList>({
    queryKey: ["workspaces", tenant, name, "pods"],
    enabled: tenant !== "" && name !== "",
    queryFn: async () => {
      const { data, error } = await sdk.listWorkspacePods({ path: { name } });
      if (error) throw error;
      return data as sdk.PodList;
    },
  });

  const pods = podsQ.data?.items ?? [];
  useEffect(() => {
    if (!pod && pods.length) setPod(pods[0].name);
  }, [pod, pods]);

  const logsQ = useQuery({
    queryKey: ["workspaces", tenant, name, "logs", pod],
    enabled: tenant !== "" && name !== "" && pod !== "",
    queryFn: async () => {
      const { data, error } = await sdk.getWorkspacePodLogs({ path: { name, pod } });
      if (error) throw error;
      return data as unknown as string;
    },
  });

  return (
    <Card>
      <CardContent>
        <div className="mb-3 flex items-center gap-3">
          <Select value={pod || undefined} onValueChange={setPod} disabled={!pods.length}>
            <SelectTrigger className="min-w-56">
              <Badge variant="secondary" className="font-mono">
                {t("workspaces.podLabel")}
              </Badge>
              <SelectValue placeholder={podsQ.isError ? t("common.loadFailed") : t("workspaces.noPods")} />
            </SelectTrigger>
            <SelectContent>
              {pods.map((p) => (
                <SelectItem key={p.name} value={p.name}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="grow" />
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            {t("workspaces.follow")}
            <Switch checked={follow} onCheckedChange={setFollow} size="sm" />
          </label>
        </div>
        {!pods.length ? (
          <Alert variant="info">
            <Info />
            <AlertDescription>{t("workspaces.logHint")}</AlertDescription>
          </Alert>
        ) : (
          <LogViewer text={logsQ.data} empty={t("workspaces.logHint")} />
        )}
      </CardContent>
    </Card>
  );
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
                    {dayjs(e.lastTimestamp).format("YYYY-MM-DD HH:mm:ss")}
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
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("workspaces.drawerEdit")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{w.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <Alert variant="info" className="mb-4">
            <Info />
            <AlertDescription>{t("workspaces.editNotice")}</AlertDescription>
          </Alert>
          <Field className="mb-4">
            <FieldLabel htmlFor="ws-edit-name">{t("workspaces.fName")}</FieldLabel>
            <Input
              id="ws-edit-name"
              value={v.displayName}
              onChange={(e) => set("displayName", e.target.value)}
            />
            <FieldDescription>{t("workspaces.fNameHelp")}</FieldDescription>
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="ws-edit-desc">{t("workspaces.fDesc")}</FieldLabel>
            <Textarea
              id="ws-edit-desc"
              rows={2}
              placeholder={t("workspaces.fDescPlaceholder")}
              value={v.description}
              onChange={(e) => set("description", e.target.value)}
            />
          </Field>
          <div className="rounded-lg border bg-muted p-3 text-sm text-muted-foreground">
            <Code2 className="mr-2 inline size-4 text-foreground" />
            {t("workspaces.fImage")}: <span className="font-mono text-foreground">{w.image}</span>
            {w.unitName && (
              <>
                {" · "}
                {t("workspaces.fUnit")}: <span className="font-mono text-foreground">{w.unitName}</span>
              </>
            )}
          </div>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={update.isPending}>
            {update.isPending && <Spinner data-icon="inline-start" />}
            {t("workspaces.saveChanges")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
