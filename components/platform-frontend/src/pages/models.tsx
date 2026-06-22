import { useMemo, useState } from "react";
import {
  Plus,
  Search,
  Trash2,
  CloudDownload,
  Copy,
  Database,
  Inbox,
  Tag,
  X,
  LayoutGrid,
  List as ListIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useModels, useModelVersions } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { DataTable, type Column } from "@/components/data-table";
import { USE_MOCK } from "@/api/mock";
import { modelVersions } from "@/api/mock/data";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Skeleton } from "@/components/ui/skeleton";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field";

interface ModelRow {
  name: string;
  desc: string;
  framework: string;
  latest: string;
  versions: number;
  updated: string;
}

type DrawerState =
  | { kind: "versions"; model: string; desc: string; framework: string }
  | { kind: "pull"; model: string; version: string }
  | { kind: "new" }
  | { kind: "upload"; model: string };

export default function Models() {
  const q = useModels();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [search, setSearch] = useState("");
  const [drawer, setDrawer] = useState<DrawerState | null>(null);

  const delModel = useApiMutation(
    (name: string) => sdk.deleteModelDefinition({ path: { tenant, name } }),
    { invalidate: [["models"]], success: t("models.modelDeleted") },
  );

  const allRows: ModelRow[] = useMemo(
    () =>
      q.data?.items?.map((m) => {
        // Version roll-ups aren't carried by the list endpoint; under mock we
        // derive latest/count from the version fixtures so cards read like the
        // prototype, otherwise stay honest ("—" / 0).
        const vs = USE_MOCK ? modelVersions(m.name) : [];
        return {
          name: m.name,
          desc: m.description ?? m.displayName ?? "",
          framework: (m.labels?.framework as string) ?? "—",
          latest: vs[0]?.version ?? "—",
          versions: vs.length,
          updated: m.updatedAt ?? m.createdAt ?? "",
        };
      }) ?? [],
    [q.data],
  );

  const rows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) return allRows;
    return allRows.filter(
      (r) => r.name.toLowerCase().includes(needle) || r.desc.toLowerCase().includes(needle),
    );
  }, [allRows, search]);

  const openVersions = (r: ModelRow) =>
    setDrawer({ kind: "versions", model: r.name, desc: r.desc, framework: r.framework });

  const onDeleteModel = (r: ModelRow) =>
    confirm({
      title: t("models.deleteModelTitle", { name: r.name }),
      desc: t("models.deleteModelDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delModel.mutate(r.name),
    });

  const columns: Column<ModelRow>[] = [
    {
      key: "name",
      title: t("models.colName"),
      render: (r) => (
        <button type="button" className="min-w-0 text-left" onClick={() => openVersions(r)}>
          <div className="font-mono font-medium text-info hover:underline">{r.name}</div>
          {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
        </button>
      ),
    },
    {
      key: "framework",
      title: t("models.colFramework"),
      width: 140,
      render: (r) =>
        r.framework && r.framework !== "—" ? (
          <Badge variant="secondary">{r.framework}</Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      key: "latest",
      title: t("models.colLatest"),
      width: 130,
      render: (r) => <span className="font-mono text-muted-foreground">{r.latest}</span>,
    },
    {
      key: "versions",
      title: t("models.colVersions"),
      width: 90,
      align: "right",
      render: (r) => <span className="font-mono">{r.versions}</span>,
    },
    {
      key: "updated",
      title: t("models.colUpdated"),
      width: 150,
      render: (r) => (
        <span className="text-muted-foreground">{r.updated ? dayjs(r.updated).fromNow() : "—"}</span>
      ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 160,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" onClick={() => setDrawer({ kind: "upload", model: r.name })}>
            {t("models.addVersion")}
          </Button>
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDeleteModel(r)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.assetCenter"), t("nav.models")]}
      title={t("models.title")}
      subtitle={t("models.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ kind: "new" })}>
          <Plus data-icon="inline-start" />
          {t("models.newModel")}
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative max-w-xs flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-8"
            placeholder={t("models.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="grow" />
        <ToggleGroup
          type="single"
          variant="outline"
          spacing={0}
          value={view}
          onValueChange={(v) => v && setView(v as "cards" | "list")}
        >
          <ToggleGroupItem value="cards" aria-label={t("models.viewCards")}>
            <LayoutGrid data-icon="inline-start" />
            {t("models.viewCards")}
          </ToggleGroupItem>
          <ToggleGroupItem value="list" aria-label={t("models.viewList")}>
            <ListIcon data-icon="inline-start" />
            {t("models.viewList")}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      {view === "cards" ? (
        q.isLoading ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Card key={i} className="p-4">
                <div className="flex items-center gap-3">
                  <Skeleton className="size-9 rounded-md" />
                  <Skeleton className="h-4 w-32" />
                </div>
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-3 w-40" />
              </Card>
            ))}
          </div>
        ) : rows.length === 0 ? (
          <Empty className="border">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Database />
              </EmptyMedia>
              <EmptyTitle>{q.isError ? t("common.loadFailed") : t("common.noData")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
              {rows.map((r) => (
                <Card
                  key={r.name}
                  className="cursor-pointer gap-0 p-4 transition-shadow hover:border-primary/30 hover:shadow-md"
                  onClick={() => openVersions(r)}
                >
                  <div className="flex items-center gap-2.5">
                    <div className="grid size-[38px] shrink-0 place-items-center rounded-[9px] border bg-muted">
                      <Database className="size-[20px] text-muted-foreground" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-mono text-sm font-semibold text-foreground">{r.name}</div>
                      {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
                    </div>
                    <Badge variant="secondary">{r.framework}</Badge>
                  </div>
                  <div className="mt-2.5 flex items-center gap-2 text-xs">
                    <span className="inline-flex items-center gap-1.5 rounded-full border bg-muted px-2 py-0.5 font-mono text-[11.5px] text-foreground/80">
                      <Tag className="size-3.5 text-muted-foreground" />
                      {r.latest}
                    </span>
                    <span className="ml-auto text-muted-foreground">
                      {r.updated ? dayjs(r.updated).fromNow() : "—"}
                    </span>
                  </div>
                  <Separator className="mt-3.5 mb-2.5" />
                  <div className="flex items-center text-xs text-muted-foreground">
                    <span>{r.versions} {t("models.versionsSuffix")}</span>
                    <div className="grow" />
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-destructive"
                          onClick={(e) => {
                            e.stopPropagation();
                            onDeleteModel(r);
                          }}
                          aria-label={t("common.delete")}
                        >
                          <Trash2 />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("common.delete")}</TooltipContent>
                    </Tooltip>
                  </div>
                </Card>
              ))}
            </div>
            <div className="mt-4 text-sm text-muted-foreground">
              {t("models.total", { count: rows.length })}
            </div>
          </>
        )
      ) : (
        <Card className="overflow-hidden p-0">
          <DataTable
            columns={columns}
            data={rows}
            rowKey={(r) => r.name}
            loading={q.isLoading}
            error={q.isError}
          />
        </Card>
      )}

      {drawer?.kind === "versions" && (
        <VersionsDrawer
          model={drawer.model}
          desc={drawer.desc}
          framework={drawer.framework}
          onClose={() => setDrawer(null)}
          onPull={(version) => setDrawer({ kind: "pull", model: drawer.model, version })}
          onUpload={() => setDrawer({ kind: "upload", model: drawer.model })}
        />
      )}
      {drawer?.kind === "pull" && (
        <PullDrawer model={drawer.model} version={drawer.version} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "new" && <NewModelDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "upload" && <UploadDrawer model={drawer.model} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Version list drawer ───────────────────────────────────────────────────────
type StatusVariant = "success" | "info" | "destructive" | "secondary";

function statusMeta(
  status: sdk.ModelStatus,
  t: (k: string) => string,
): { variant: StatusVariant; label: string; pending: boolean } {
  switch (status) {
    case "Ready":
      return { variant: "success", label: t("models.statusReady"), pending: false };
    case "Uploading":
      return { variant: "info", label: t("models.statusUploading"), pending: true };
    case "Failed":
      return { variant: "destructive", label: t("models.statusFailed"), pending: false };
    default:
      return { variant: "secondary", label: status, pending: false };
  }
}

function VersionsDrawer({
  model,
  desc,
  framework,
  onClose,
  onPull,
  onUpload,
}: {
  model: string;
  desc: string;
  framework: string;
  onClose: () => void;
  onPull: (version: string) => void;
  onUpload: () => void;
}) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast, confirm } = useUI();
  const [search, setSearch] = useState("");
  const versQ = useModelVersions(model);

  const delVer = useApiMutation(
    (version: string) => sdk.deleteModel({ path: { tenant, name: model, version } }),
    { invalidate: [["models"]], success: t("models.verDeleted") },
  );

  const items = versQ.data?.items ?? [];
  const filtered = items.filter((v) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return v.version.toLowerCase().includes(needle) || (v.description ?? "").toLowerCase().includes(needle);
  });

  const onDeleteVer = (version: string) =>
    confirm({
      title: t("models.deleteVerTitle", { version }),
      desc: t("models.deleteVerDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delVer.mutate(version),
    });

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle className="font-mono">{model}</SheetTitle>
          <p className="text-xs text-muted-foreground">{`${desc || t("models.verWeights")} · ${framework}`}</p>
        </SheetHeader>

        <div className="flex flex-1 flex-col overflow-auto px-6 py-4">
          <div className="mb-4 flex items-center gap-3">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-8"
                placeholder={t("models.verSearchPlaceholder")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <Button onClick={onUpload}>
              <Plus data-icon="inline-start" />
              {t("models.addVersion")}
            </Button>
          </div>

          {versQ.isLoading ? (
            <div className="grid place-items-center py-12">
              <Spinner className="size-7 text-muted-foreground" />
            </div>
          ) : filtered.length === 0 ? (
            <Empty className="border">
              <EmptyHeader>
                <EmptyTitle>{versQ.isError ? t("common.loadFailed") : t("common.noData")}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            <ul className="flex flex-col">
              {filtered.map((v, i) => {
                const meta = statusMeta(v.status, t);
                return (
                  <li key={v.version}>
                    {i > 0 && <Separator />}
                    <div className="flex items-start gap-3 py-3">
                      <div className="min-w-0 flex-1">
                        <div className="mb-1 flex flex-wrap items-center gap-2">
                          <span className="font-mono font-medium text-foreground">{v.version}</span>
                          <Badge variant={meta.variant}>{meta.label}</Badge>
                          {v.source && <Badge variant="outline">{v.source}</Badge>}
                        </div>
                        {v.description && (
                          <div className="mb-1 text-sm text-muted-foreground">{v.description}</div>
                        )}
                        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                          <span className={"font-mono " + (meta.pending ? "" : "text-foreground")}>
                            {v.uri ?? t("models.addrPending")}
                          </span>
                          {v.uri && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon-xs"
                                  onClick={() => {
                                    void navigator.clipboard?.writeText(v.uri ?? "");
                                    toast(t("models.addrCopied"));
                                  }}
                                >
                                  <Copy />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>{t("common.actions")}</TooltipContent>
                            </Tooltip>
                          )}
                          {v.owner && <span>· {v.owner}</span>}
                        </div>
                      </div>
                      {!meta.pending && (
                        <div className="flex items-center gap-0.5">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button variant="ghost" size="icon-sm" onClick={() => onPull(v.version)}>
                                <CloudDownload />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t("models.pullTitle")}</TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                className="text-destructive"
                                onClick={() => onDeleteVer(v.version)}
                              >
                                <Trash2 />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t("common.delete")}</TooltipContent>
                          </Tooltip>
                        </div>
                      )}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

// ── Pull-command drawer ───────────────────────────────────────────────────────
function PullDrawer({ model, version, onClose }: { model: string; version: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast } = useUI();
  const resolveQ = useModelResolve(model, version, tenant);
  const cmd = resolveQ.uri ? `docker pull ${resolveQ.uri}` : t("models.pullResolving");

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[520px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("models.pullTitle")}</SheetTitle>
          <p className="font-mono text-xs text-muted-foreground">{`${model}@${version}`}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <p className="mb-3 text-sm text-muted-foreground">{t("models.pullHint")}</p>
          <pre className="overflow-x-auto rounded-md border bg-muted p-3 font-mono text-xs text-foreground">
            {cmd}
          </pre>
          <Button
            variant="outline"
            className="mt-3"
            disabled={!resolveQ.uri}
            onClick={() => {
              void navigator.clipboard?.writeText(cmd);
              toast(t("models.commandCopied"));
            }}
          >
            <Copy data-icon="inline-start" />
            {t("models.copyCommand")}
          </Button>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button onClick={onClose}>{t("models.done")}</Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// Resolve a version's pull URI on demand (thin local query; not a shared hook).
function useModelResolve(model: string, version: string, tenant: string): { uri?: string } {
  const q = useModelVersions(model);
  // Versions list already carries the URI; resolve endpoint mints temp creds but
  // the displayed pull target is the version URI. Fall back to a live resolve only
  // if the version row lacks one.
  const fromList = q.data?.items?.find((v) => v.version === version)?.uri;
  void tenant;
  return { uri: fromList };
}

// ── New-model drawer ──────────────────────────────────────────────────────────
const TASK_OPTIONS = [
  "Text Generation",
  "Text Classification",
  "Question Answering",
  "Summarization",
  "Translation",
  "Feature Extraction",
  "Image Classification",
  "Object Detection",
  "Automatic Speech Recognition",
  "Text-to-Image",
];
const FRAMEWORK_OPTIONS = ["PyTorch", "Safetensors", "Transformers", "TensorFlow", "JAX", "ONNX", "GGUF"];
const NONE = "__none__";

interface NewModelValues {
  name: string;
  description: string;
  tasks: string[];
  framework: string;
  params: string;
}

function NewModelDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<NewModelValues>({
    name: "",
    description: "",
    tasks: [],
    framework: "",
    params: "",
  });
  const set = <K extends keyof NewModelValues>(k: K, val: NewModelValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));
  const [taskInput, setTaskInput] = useState("");
  const [customTags, setCustomTags] = useState<Record<string, string>>({});
  const [ctKey, setCtKey] = useState("");
  const [ctVal, setCtVal] = useState("");

  const create = useApiMutation(
    (body: sdk.ArtifactDefinitionCreateRequest) => sdk.createModelDefinition({ path: { tenant, name: body.name }, body }),
    { invalidate: [["models"]], success: t("models.modelCreated") },
  );

  const addTask = (task: string) => {
    const tk = task.trim();
    if (!tk || v.tasks.includes(tk)) return;
    set("tasks", [...v.tasks, tk]);
    setTaskInput("");
  };
  const removeTask = (task: string) => set("tasks", v.tasks.filter((x) => x !== task));

  const addTag = () => {
    const k = ctKey.trim();
    if (!k) return;
    setCustomTags((m) => ({ ...m, [k]: ctVal.trim() }));
    setCtKey("");
    setCtVal("");
  };
  const removeTag = (k: string) =>
    setCustomTags((m) => {
      const next = { ...m };
      delete next[k];
      return next;
    });

  const submit = () => {
    setSubmitted(true);
    if (!v.name.trim()) return;
    const labels: Record<string, string> = {};
    if (v.framework) labels.framework = v.framework;
    if (v.tasks.length) labels.tasks = v.tasks.join(",");
    if (v.params.trim()) labels.params = v.params.trim();
    create.mutate(
      {
        name: v.name.trim(),
        description: v.description.trim() || undefined,
        labels: Object.keys(labels).length ? labels : undefined,
        annotations: Object.keys(customTags).length ? customTags : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("models.newModelTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">{t("models.newModelSub")}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("models.fsBasic")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="model-name">
                {t("models.fName")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="model-name"
                className="font-mono"
                placeholder={t("models.fNamePlaceholder")}
                value={v.name}
                aria-invalid={submitted && !v.name.trim()}
                onChange={(e) => set("name", e.target.value)}
              />
              <FieldDescription>{t("models.fNameHelp")}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="model-desc">{t("models.fDesc")}</FieldLabel>
              <Textarea
                id="model-desc"
                rows={2}
                placeholder={t("models.fDescPlaceholder")}
                value={v.description}
                onChange={(e) => set("description", e.target.value)}
              />
            </Field>
          </FieldGroup>

          <FieldSection n={2} title={t("models.fsLabels")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="model-tasks">{t("models.lTasks")}</FieldLabel>
              {v.tasks.length > 0 && (
                <div className="mb-2 flex flex-wrap gap-2">
                  {v.tasks.map((task) => (
                    <Badge key={task} variant="outline" className="gap-1 pr-1">
                      {task}
                      <button
                        type="button"
                        className="grid size-3.5 place-items-center rounded-sm hover:bg-muted"
                        onClick={() => removeTask(task)}
                      >
                        <X className="size-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}
              <Input
                id="model-tasks"
                placeholder={t("models.lTasks")}
                value={taskInput}
                list="model-task-options"
                onChange={(e) => setTaskInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addTask(taskInput);
                  }
                }}
              />
              <datalist id="model-task-options">
                {TASK_OPTIONS.map((o) => (
                  <option key={o} value={o} />
                ))}
              </datalist>
            </Field>
            <Field>
              <FieldLabel htmlFor="model-params">{t("models.lParameters")}</FieldLabel>
              <Input
                id="model-params"
                className="w-40 font-mono"
                placeholder={t("models.paramsPlaceholder")}
                value={v.params}
                onChange={(e) => set("params", e.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel>{t("models.lFramework")}</FieldLabel>
              <Select
                value={v.framework || NONE}
                onValueChange={(val) => set("framework", val === NONE ? "" : val)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("models.lFramework")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>{t("models.lFramework")}</SelectItem>
                  {FRAMEWORK_OPTIONS.map((o) => (
                    <SelectItem key={o} value={o}>
                      {o}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field>
              <FieldLabel>{t("models.lCustom")}</FieldLabel>
              {Object.keys(customTags).length > 0 && (
                <div className="mb-2 flex flex-wrap gap-2">
                  {Object.entries(customTags).map(([k, val]) => (
                    <Badge key={k} variant="outline" className="gap-1 pr-1 font-mono">
                      {k}:{val}
                      <button
                        type="button"
                        className="grid size-3.5 place-items-center rounded-sm hover:bg-muted"
                        onClick={() => removeTag(k)}
                      >
                        <X className="size-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}
              <div className="flex items-center gap-2">
                <Input
                  className="font-mono"
                  placeholder={t("models.customKeyPlaceholder")}
                  value={ctKey}
                  onChange={(e) => setCtKey(e.target.value)}
                />
                <Input
                  className="font-mono"
                  placeholder={t("models.customValPlaceholder")}
                  value={ctVal}
                  onChange={(e) => setCtVal(e.target.value)}
                />
                <Button variant="outline" onClick={addTag}>
                  {t("models.add")}
                </Button>
              </div>
            </Field>
          </FieldGroup>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={create.isPending}>
            {create.isPending && <Spinner data-icon="inline-start" />}
            {t("models.createModel")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Upload-version drawer (two add methods: web upload vs external register) ───
interface UploadValues {
  version: string;
  description: string;
  remoteSourceKind: sdk.RemoteSourceKind;
  remoteUri: string;
}

function UploadDrawer({ model, onClose }: { model: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [submitted, setSubmitted] = useState(false);
  const [method, setMethod] = useState<sdk.ArtifactSource>("webUpload");
  const [v, setV] = useState<UploadValues>({
    version: "",
    description: "",
    remoteSourceKind: "s3",
    remoteUri: "",
  });
  const set = <K extends keyof UploadValues>(k: K, val: UploadValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const initiate = useApiMutation(
    (body: sdk.ModelInitiateRequest) => sdk.initiateModel({ path: { tenant, name: model }, body }),
    { invalidate: [["models"]], success: t("models.versionSubmitted") },
  );

  const submit = () => {
    setSubmitted(true);
    const isExternal = method === "external";
    if (!v.version.trim()) return;
    if (isExternal && !v.remoteUri.trim()) return;
    initiate.mutate(
      {
        version: v.version.trim(),
        description: v.description.trim() || undefined,
        source: method,
        remoteSourceKind: isExternal ? v.remoteSourceKind : undefined,
        remoteUri: isExternal && v.remoteUri.trim() ? v.remoteUri.trim() : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("models.uploadTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">{t("models.uploadSub")}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("models.fsBasic")} />
          <FieldGroup>
            <Field>
              <FieldLabel>{t("models.fModel")}</FieldLabel>
              <Input className="font-mono" value={model} disabled />
            </Field>
            <Field>
              <FieldLabel htmlFor="upload-version">
                {t("models.fVersion")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Input
                id="upload-version"
                className="font-mono"
                placeholder={t("models.fVersionPlaceholder")}
                value={v.version}
                aria-invalid={submitted && !v.version.trim()}
                onChange={(e) => set("version", e.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="upload-desc">{t("models.fDesc")}</FieldLabel>
              <Textarea
                id="upload-desc"
                rows={2}
                placeholder={t("models.fUploadDescPlaceholder")}
                value={v.description}
                onChange={(e) => set("description", e.target.value)}
              />
            </Field>
          </FieldGroup>

          <FieldSection n={2} title={t("models.fsMethod")} />
          <Tabs
            value={method === "external" ? "remote" : method}
            onValueChange={(k) => setMethod(k === "remote" ? "external" : (k as sdk.ArtifactSource))}
          >
            <TabsList>
              <TabsTrigger value="webUpload">{t("models.methodWeb")}</TabsTrigger>
              <TabsTrigger value="remote">{t("models.methodRemote")}</TabsTrigger>
              <TabsTrigger value="oras">{t("models.methodOras")}</TabsTrigger>
            </TabsList>
            <TabsContent value="webUpload" className="pt-4">
              <div className="grid place-items-center gap-2 rounded-lg border border-dashed bg-card p-8 text-center">
                <Inbox className="size-8 text-muted-foreground" />
                <div className="text-sm font-medium text-foreground">{t("models.dzTitle")}</div>
                <div className="text-xs text-muted-foreground">{t("models.dzHint")}</div>
              </div>
            </TabsContent>
            <TabsContent value="remote" className="flex flex-col gap-4 pt-4">
              <FieldGroup>
                <Field>
                  <FieldLabel>{t("models.fStorageKind")}</FieldLabel>
                  <Select
                    value={v.remoteSourceKind}
                    onValueChange={(val) => set("remoteSourceKind", val as sdk.RemoteSourceKind)}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="s3">{t("models.storageS3")}</SelectItem>
                      <SelectItem value="oci">{t("models.storageOci")}</SelectItem>
                      <SelectItem value="http">{t("models.storageHttp")}</SelectItem>
                      <SelectItem value="hf">{t("models.storageHf")}</SelectItem>
                      <SelectItem value="custom">{t("models.storageCustom")}</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
                <Field>
                  <FieldLabel htmlFor="remote-uri">{t("models.fRemoteUri")}</FieldLabel>
                  <Input
                    id="remote-uri"
                    className="font-mono"
                    placeholder={t("models.remoteUriPlaceholder")}
                    value={v.remoteUri}
                    aria-invalid={submitted && method === "external" && !v.remoteUri.trim()}
                    onChange={(e) => set("remoteUri", e.target.value)}
                  />
                </Field>
              </FieldGroup>
            </TabsContent>
            <TabsContent value="oras" className="pt-4">
              <OrasGuide model={model} tenant={tenant} />
            </TabsContent>
          </Tabs>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={initiate.isPending}>
            {initiate.isPending && <Spinner data-icon="inline-start" />}
            {t("models.submit")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function OrasGuide({ model, tenant }: { model: string; tenant: string }) {
  const { t } = useTranslation();
  const dl = `# Linux x86_64
curl -LO https://github.com/oras-project/oras/releases/download/v1.2.0/oras_1.2.0_linux_amd64.tar.gz
tar -xzf oras_1.2.0_linux_amd64.tar.gz oras
sudo mv oras /usr/local/bin/ && oras version`;
  const push = `oras login zot.axisml.internal -u <user> -p <token>
cd ./${model}
oras push zot.axisml.internal/${tenant}/${model}:v5 \\
  --artifact-type application/vnd.axisml.model.v1 \\
  ./*:application/octet-stream`;
  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-muted-foreground">{t("models.orasHelp")}</p>
      <div>
        <div className="mb-1 text-sm font-semibold text-foreground">{t("models.orasStep1")}</div>
        <pre className="overflow-x-auto rounded-md border bg-muted p-3 font-mono text-xs text-foreground">{dl}</pre>
        <a className="text-xs text-info hover:underline" href="https://oras.land/docs/installation" target="_blank" rel="noopener noreferrer">
          {t("models.orasDocsLink")}
        </a>
      </div>
      <div>
        <div className="mb-1 text-sm font-semibold text-foreground">{t("models.orasStep2")}</div>
        <pre className="overflow-x-auto rounded-md border bg-muted p-3 font-mono text-xs text-foreground">{push}</pre>
      </div>
    </div>
  );
}
