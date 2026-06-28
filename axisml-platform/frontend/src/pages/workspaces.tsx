import { useMemo, useState, type ReactNode } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  Plus,
  Trash2,
  Play,
  Power,
  LayoutGrid,
  List as ListIcon,
  Cpu,
  Terminal,
} from "lucide-react";
import {
  JupyterMark,
  VscodeMark,
  PytorchMark,
  WorkspaceMark,
  wsBrand,
} from "@/components/workspace-brand";
import { useTranslation } from "react-i18next";
import { useWorkspaces, useVolumeOptions } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { FieldSection } from "@/components/field-section";
import { CardRadio } from "@/components/card-radio";
import { SearchInput } from "@/components/search-input";
import { FilterSelect } from "@/components/filter-select";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { FormDrawer } from "@/components/form-drawer";
import { parseEnv } from "@/lib/run-spec";
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface WsRow {
  name: string;
  desc: string;
  phase?: string;
  unit: string;
  image: string;
  owner: string;
  pvc?: string;
}

const isRunning = (phase?: string) =>
  phase === "Running" || phase === "Degraded" || phase === "Starting" || phase === "Creating" || phase === "Pending";
const isStopped = (phase?: string) => !isRunning(phase);

export default function Workspaces() {
  const q = useWorkspaces();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [drawer, setDrawer] = useState(false);
  const [search, setSearch] = useState("");
  const [phase, setPhase] = useState<string>("");
  const [pool, setPool] = useState<string>("");
  const [creator, setCreator] = useState<string>("");

  const start = useApiMutation((name: string) => sdk.startWorkspace({ path: { name } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.starting"),
  });
  const stop = useApiMutation((name: string) => sdk.stopWorkspace({ path: { name } }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.stopped"),
  });
  const del = useApiMutation(
    (vars: { name: string; deletePvc: boolean }) =>
      sdk.deleteWorkspace({ path: { name: vars.name }, body: { deletePvc: vars.deletePvc } }),
    { invalidate: [["workspaces"]], success: t("workspaces.deleted") },
  );

  const allRows: WsRow[] = useMemo(
    () =>
      q.data?.items?.map((w) => ({
        name: w.name,
        desc: w.description ?? w.displayName ?? "",
        phase: w.phase,
        unit: w.unitName ?? "—",
        image: w.image ?? "—",
        owner: w.owner ?? "—",
        pvc: w.volumes?.find((v) => v.size)?.size,
      })) ?? [],
    [q.data],
  );

  const poolOptions = useMemo(
    () => Array.from(new Set((q.data?.items ?? []).map((w) => w.poolName).filter(Boolean) as string[])),
    [q.data],
  );
  const creatorOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.owner).filter((o) => o && o !== "—"))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) =>
      (!search || r.name.includes(search)) &&
      (!phase || r.phase === phase) &&
      (!creator || r.owner === creator) &&
      (!pool ||
        (q.data?.items ?? []).some((w) => w.name === r.name && w.poolName === pool)),
  );

  const onDelete = (r: WsRow, descKey: "running" | "stopped" | "default") => {
    let deletePvc = r.pvc != null;
    confirm({
      title: t("workspaces.deleteTitle", { name: r.name }),
      desc:
        descKey === "running"
          ? t("workspaces.deleteDescRunning")
          : descKey === "stopped"
            ? t("workspaces.deleteDescStopped")
            : t("workspaces.deleteDescDefault"),
      info:
        r.pvc != null ? (
          <label className="flex items-center gap-2">
            <Checkbox defaultChecked onCheckedChange={(c) => (deletePvc = c === true)} />
            {t("workspaces.deletePvc", { size: r.pvc })}
          </label>
        ) : undefined,
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate({ name: r.name, deletePvc }),
    });
  };

  const columns: Column<WsRow>[] = [
    {
      key: "name",
      title: t("workspaces.colName"),
      render: (r) => (
        <div className="min-w-0">
          <Link
            to={`/workspaces/${r.name}`}
            className="font-mono font-medium text-foreground hover:text-info hover:underline"
          >
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
        </div>
      ),
    },
    {
      key: "phase",
      title: t("workspaces.colStatus"),
      width: 120,
      render: (r) => <PhaseTag phase={r.phase} />,
    },
    {
      key: "unit",
      title: t("workspaces.colUnit"),
      width: 160,
      render: (r) => <span className="font-mono text-sm">{r.unit}</span>,
    },
    {
      key: "image",
      title: t("workspaces.colImage"),
      width: 200,
      render: (r) => <span className="font-mono text-sm text-muted-foreground">{r.image}</span>,
    },
    { key: "owner", title: t("workspaces.colCreator"), width: 140, dataIndex: "owner" },
    {
      key: "actions",
      title: t("common.actions"),
      width: 180,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" asChild>
            <Link to={`/workspaces/${r.name}`}>{t("common.detail")}</Link>
          </Button>
          {isStopped(r.phase) ? (
            <Button variant="link" size="sm" onClick={() => start.mutate(r.name)}>
              {t("workspaces.start")}
            </Button>
          ) : (
            <Button variant="link" size="sm" onClick={() => stop.mutate(r.name)}>
              {t("workspaces.stop")}
            </Button>
          )}
          <Button
            variant="link"
            size="sm"
            className="text-destructive"
            onClick={() => onDelete(r, isStopped(r.phase) ? "stopped" : "running")}
          >
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.workspace")]}
      title={t("workspaces.title")}
      subtitle={t("workspaces.subtitle")}
      extra={
        <Button onClick={() => setDrawer(true)}>
          <Plus data-icon="inline-start" />
          {t("workspaces.newWorkspace")}
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <SearchInput
          className="max-w-xs flex-1"
          placeholder={t("workspaces.searchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
        <FilterSelect
          value={phase}
          onChange={setPhase}
          options={["Running", "Starting", "Stopped"].map((p) => ({ value: p, label: t(`phase.${p}`) }))}
          allLabel={t("workspaces.statusAll")}
        />
        <FilterSelect
          value={pool}
          onChange={setPool}
          options={poolOptions}
          allLabel={t("workspaces.poolAll")}
        />
        <FilterSelect
          value={creator}
          onChange={setCreator}
          options={creatorOptions}
          allLabel={t("workspaces.creatorAll")}
        />
        <Button
          variant="outline"
          onClick={() => {
            setSearch("");
            setPhase("");
            setPool("");
            setCreator("");
          }}
        >
          {t("common.reset")}
        </Button>
        <div className="grow" />
        <ToggleGroup
          type="single"
          variant="outline"
          spacing={0}
          value={view}
          onValueChange={(v) => v && setView(v as "cards" | "list")}
        >
          <ToggleGroupItem value="cards" aria-label={t("workspaces.viewCards")}>
            <LayoutGrid data-icon="inline-start" />
            {t("workspaces.viewCards")}
          </ToggleGroupItem>
          <ToggleGroupItem value="list" aria-label={t("workspaces.viewList")}>
            <ListIcon data-icon="inline-start" />
            {t("workspaces.viewList")}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      {view === "cards" ? (
        <CardsView
          q={q}
          rows={rows}
          onStart={(name) => start.mutate(name)}
          onStop={(name) => stop.mutate(name)}
          onDelete={onDelete}
        />
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

      {drawer && <WsDrawer onClose={() => setDrawer(false)} />}
    </PageContainer>
  );
}

function CardsView({
  q,
  rows,
  onStart,
  onStop,
  onDelete,
}: {
  q: ReturnType<typeof useWorkspaces>;
  rows: WsRow[];
  onStart: (name: string) => void;
  onStop: (name: string) => void;
  onDelete: (r: WsRow, descKey: "running" | "stopped" | "default") => void;
}) {
  const { t } = useTranslation();
  if (q.isLoading) {
    return (
      <div className="grid place-items-center py-20">
        <Spinner className="size-7 text-muted-foreground" />
      </div>
    );
  }
  if (q.isError) {
    return (
      <Card>
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{t("common.loadFailed")}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      </Card>
    );
  }
  if (rows.length === 0) {
    return (
      <Card>
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{t("common.noData")}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      </Card>
    );
  }
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      {rows.map((r) => (
        <WsCard key={r.name} row={r} onStart={onStart} onStop={onStop} onDelete={onDelete} />
      ))}
    </div>
  );
}

function WsCard({
  row,
  onStart,
  onStop,
  onDelete,
}: {
  row: WsRow;
  onStart: (name: string) => void;
  onStop: (name: string) => void;
  onDelete: (r: WsRow, descKey: "running" | "stopped" | "default") => void;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const running = row.phase === "Running" || row.phase === "Degraded";
  const stopped = isStopped(row.phase);
  const brand = wsBrand(row.image);

  // One launch-tool button (brand mark on the bottom-left action row).
  const launch = (key: string, title: string, mark: ReactNode) => (
    <Tooltip key={key}>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon-sm"
          disabled={!running}
          asChild={running}
          aria-label={title}
        >
          {running ? <Link to={`/workspaces/${row.name}`}>{mark}</Link> : mark}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{running ? title : t("workspaces.availableAfterStart")}</TooltipContent>
    </Tooltip>
  );

  return (
    <Card
      role="link"
      tabIndex={0}
      onClick={() => navigate(`/workspaces/${row.name}`)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          navigate(`/workspaces/${row.name}`);
        }
      }}
      className="cursor-pointer gap-0 p-4 transition-shadow hover:border-primary/30 hover:shadow-md focus-visible:border-primary/30 focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none"
    >
      <div className="flex items-center gap-2.5">
        <div className="grid size-[38px] shrink-0 place-items-center rounded-[9px] border bg-muted">
          <WorkspaceMark brand={brand} className="size-[22px]" />
        </div>
        <div className="min-w-0 flex-1">
          <Link
            to={`/workspaces/${row.name}`}
            className="font-mono text-sm font-semibold text-foreground hover:text-info hover:underline"
          >
            {row.name}
          </Link>
          {row.desc && <div className="truncate text-xs text-muted-foreground">{row.desc}</div>}
        </div>
        <PhaseTag phase={row.phase} />
      </div>

      <div className="mt-2.5 flex items-center gap-2">
        <span className="inline-flex items-center gap-1.5 rounded-full border bg-muted px-2 py-0.5 font-mono text-[11.5px] text-foreground/80">
          <Cpu className="size-3.5 text-muted-foreground" />
          {row.unit}
        </span>
        <span className="ml-auto inline-flex items-center gap-1.5 text-xs text-foreground/80">
          <span className="grid size-5 place-items-center rounded-full bg-muted text-[10px] font-semibold text-foreground">
            {row.owner.slice(0, 1)}
          </span>
          {row.owner}
        </span>
      </div>

      <Separator className="mt-3.5 mb-2.5" />

      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
        {stopped ? (
          <>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" onClick={() => onStart(row.name)} aria-label={t("workspaces.start")}>
                  <Play className="fill-current" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("workspaces.start")}</TooltipContent>
            </Tooltip>
            <div className="grow" />
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" className="text-destructive" onClick={() => onDelete(row, "stopped")} aria-label={t("workspaces.remove")}>
                  <Trash2 />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("workspaces.remove")}</TooltipContent>
            </Tooltip>
          </>
        ) : (
          <>
            {launch("jupyter", t("workspaces.openJupyter"), <JupyterMark className="size-[18px]" />)}
            {launch("vscode", t("workspaces.openVscode"), <VscodeMark className="size-[18px]" />)}
            {launch("terminal", t("workspaces.openTerminal"), <Terminal />)}
            <div className="grow" />
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" onClick={() => onStop(row.name)} aria-label={t("workspaces.stop")}>
                  <Power />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("workspaces.stop")}</TooltipContent>
            </Tooltip>
            {running && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-sm" className="text-destructive" onClick={() => onDelete(row, "running")} aria-label={t("workspaces.remove")}>
                    <Trash2 />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("workspaces.remove")}</TooltipContent>
              </Tooltip>
            )}
          </>
        )}
      </div>
    </Card>
  );
}

// ── Create drawer ─────────────────────────────────────────────────────────────
const WS_IMAGES: { value: string; title: string; desc: string; icon: ReactNode }[] = [
  {
    value: "jupyter-ds:2024.3",
    title: "jupyter-ds:2024.3",
    desc: "Jupyter 开发环境 · 公共",
    icon: <JupyterMark className="size-[22px]" />,
  },
  {
    value: "pytorch:2.3-cu121",
    title: "pytorch:2.3-cu121",
    desc: "PyTorch 训练镜像",
    icon: <PytorchMark className="size-[22px]" />,
  },
  {
    value: "vscode-server:1.90",
    title: "vscode-server:1.90",
    desc: "VS Code 开发环境 · 公共",
    icon: <VscodeMark className="size-[22px]" />,
  },
];
const WS_POOLS: { value: string; label: string }[] = [
  { value: "gpu-a100", label: "gpu-a100 · A100 训练池" },
  { value: "gpu-l40s", label: "gpu-l40s · L40S 推理池" },
  { value: "cpu-medium", label: "cpu-medium · 通用 CPU 池" },
];
const WS_UNITS: { value: string; pool: string; title: string; desc: string }[] = [
  { value: "cpu-medium", pool: "cpu-medium", title: "cpu-medium", desc: "8 vCPU · 32 GiB" },
  { value: "cpu-large", pool: "cpu-medium", title: "cpu-large", desc: "16 vCPU · 64 GiB" },
  { value: "a100-1x", pool: "gpu-a100", title: "a100-1x", desc: "1×A100 · 8 vCPU · 64 GiB" },
  { value: "l40s-1x", pool: "gpu-l40s", title: "l40s-1x", desc: "1×L40S · 8 vCPU · 48 GiB" },
];

interface WsFormValues {
  name: string;
  description: string;
  image: string;
  poolName: string;
  unitName: string;
  containerPort: number;
  env: string;
  volume: string;
  mountPath: string;
}

function WsDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [submitted, setSubmitted] = useState(false);
  const { options: volOptions, isError: volError } = useVolumeOptions();
  const [v, setV] = useState<WsFormValues>({
    name: "",
    description: "",
    image: WS_IMAGES[0].value,
    poolName: WS_UNITS[0].pool,
    unitName: WS_UNITS[0].value,
    containerPort: 8888,
    env: "",
    volume: "",
    mountPath: "/workspace",
  });
  const set = <K extends keyof WsFormValues>(k: K, val: WsFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  // Pool drives which units are offered (the prototype scopes the unit grid to
  // the selected pool); picking a pool resets the unit to the first match.
  const unitsForPool = WS_UNITS.filter((u) => u.pool === v.poolName);
  const onPoolChange = (poolName: string) => {
    const first = WS_UNITS.find((u) => u.pool === poolName);
    setV((prev) => ({ ...prev, poolName, unitName: first?.value ?? prev.unitName }));
  };

  const create = useApiMutation((body: sdk.WorkspaceCreateRequest) => sdk.createWorkspace({ body }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.created"),
  });

  const submit = () => {
    setSubmitted(true);
    if (!v.name.trim()) return;
    const unit = unitsForPool.find((u) => u.value === v.unitName) ?? unitsForPool[0] ?? WS_UNITS[0];
    const envVars = parseEnv(v.env || "");
    const mountPath = v.mountPath?.trim();
    const body: sdk.WorkspaceCreateRequest = {
      name: v.name.trim(),
      image: v.image,
      poolName: v.poolName,
      unitName: unit.value,
      description: v.description?.trim() || undefined,
      containerPort: v.containerPort && v.containerPort > 0 ? v.containerPort : undefined,
      env: envVars.length ? envVars : undefined,
      volumes: v.volume && mountPath ? [{ name: v.volume, mountPath }] : undefined,
    };
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <FormDrawer
      title={t("workspaces.drawerNew")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("workspaces.createWorkspace")}
      submitting={create.isPending}
    >
      <FieldSection n={1} title={t("workspaces.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="ws-name">
            {t("workspaces.fName")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="ws-name"
            className="font-mono"
            placeholder={t("workspaces.fNamePlaceholder")}
            value={v.name}
            aria-invalid={submitted && !v.name.trim()}
            onChange={(e) => set("name", e.target.value)}
          />
          <FieldDescription>{t("workspaces.fNameHelp")}</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="ws-desc">{t("workspaces.fDesc")}</FieldLabel>
          <Textarea
            id="ws-desc"
            rows={2}
            placeholder={t("workspaces.fDescPlaceholder")}
            value={v.description}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("workspaces.fsImage")} />
      <FieldGroup>
        <Field>
          <FieldLabel>
            {t("workspaces.fImage")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <CardRadio columns={1} options={WS_IMAGES} value={v.image} onChange={(val) => set("image", val)} />
        </Field>
      </FieldGroup>

      <FieldSection n={3} title={t("workspaces.fsResource")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="ws-pool">
            {t("workspaces.fPool")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Select value={v.poolName} onValueChange={onPoolChange}>
            <SelectTrigger id="ws-pool" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {WS_POOLS.map((p) => (
                <SelectItem key={p.value} value={p.value}>
                  {p.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field>
          <FieldLabel>
            {t("workspaces.fUnit")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <CardRadio options={unitsForPool} value={v.unitName} onChange={(val) => set("unitName", val)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="ws-port">{t("workspaces.fContainerPort")}</FieldLabel>
          <Input
            id="ws-port"
            type="number"
            min={1}
            max={65535}
            className="w-40 font-mono"
            value={v.containerPort}
            onChange={(e) => set("containerPort", Number(e.target.value))}
          />
          <FieldDescription>{t("workspaces.fPortHelp")}</FieldDescription>
        </Field>
      </FieldGroup>

      <FieldSection n={4} title={t("workspaces.fsEnv")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="ws-env">{t("workspaces.fEnv")}</FieldLabel>
          <Textarea
            id="ws-env"
            rows={2}
            className="font-mono"
            placeholder={"HF_HOME=/data/hf\nCUDA_VISIBLE_DEVICES=0"}
            value={v.env}
            onChange={(e) => set("env", e.target.value)}
          />
          <FieldDescription>{t("workspaces.fEnvHelp")}</FieldDescription>
        </Field>
      </FieldGroup>

      <FieldSection n={5} title={t("workspaces.fsVolume")} />
      <FieldGroup>
        <div className="flex gap-3">
          <Field className="flex-1">
            <FieldLabel htmlFor="ws-volume">{t("workspaces.fVolume")}</FieldLabel>
            <Select value={v.volume} onValueChange={(val) => set("volume", val)}>
              <SelectTrigger id="ws-volume" className="w-full">
                <SelectValue placeholder={t("workspaces.fVolumePlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                {volOptions.map((vol) => (
                  <SelectItem key={vol.value} value={vol.value}>
                    {vol.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field className="w-56">
            <FieldLabel htmlFor="ws-mount">{t("workspaces.fMountPath")}</FieldLabel>
            <Input
              id="ws-mount"
              className="font-mono"
              placeholder="/workspace"
              value={v.mountPath}
              onChange={(e) => set("mountPath", e.target.value)}
            />
          </Field>
        </div>
        <FieldDescription>
          {volError
            ? t("common.loadFailed")
            : volOptions.length === 0
              ? t("common.noDataVolumes")
              : t("workspaces.fVolumeHelp")}
        </FieldDescription>
      </FieldGroup>
    </FormDrawer>
  );
}
