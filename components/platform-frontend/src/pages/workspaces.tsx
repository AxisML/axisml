import { useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  Plus,
  Search,
  Trash2,
  Play,
  Power,
  LayoutGrid,
  List as ListIcon,
  Cpu,
  SquareTerminal,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaces } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { PhaseTag } from "@/components/phase-tag";
import { FieldSection } from "@/components/field-section";
import { CardRadio } from "@/components/card-radio";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
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
import { Field, FieldGroup, FieldLabel, FieldDescription } from "@/components/ui/field";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
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

const ALL = "__all__";

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
            <input type="checkbox" defaultChecked onChange={(e) => (deletePvc = e.target.checked)} />
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
      <Card className="mb-4 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative max-w-xs flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("workspaces.searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Select value={phase || ALL} onValueChange={(v) => setPhase(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-36">
              <SelectValue placeholder={t("workspaces.statusAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("workspaces.statusAll")}</SelectItem>
              {["Running", "Starting", "Stopped"].map((p) => (
                <SelectItem key={p} value={p}>
                  {t(`phase.${p}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={pool || ALL} onValueChange={(v) => setPool(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-36">
              <SelectValue placeholder={t("workspaces.poolAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("workspaces.poolAll")}</SelectItem>
              {poolOptions.map((o) => (
                <SelectItem key={o} value={o}>
                  {o}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={creator || ALL} onValueChange={(v) => setCreator(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-36">
              <SelectValue placeholder={t("workspaces.creatorAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("workspaces.creatorAll")}</SelectItem>
              {creatorOptions.map((o) => (
                <SelectItem key={o} value={o}>
                  {o}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
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
      </Card>

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

// Tool brand marks (official colours), matching the prototype's `svg.brand`.
function JupyterMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden focusable="false">
      <path
        fill="#F37726"
        d="M7.157 22.201A1.784 1.784 0 0 1 5.374 24a1.784 1.784 0 0 1-1.784-1.799 1.784 1.784 0 0 1 1.784-1.799 1.784 1.784 0 0 1 1.783 1.799zM20.582 1.427a1.415 1.415 0 0 1-1.415 1.428 1.415 1.415 0 0 1-1.416-1.428A1.415 1.415 0 0 1 19.167 0a1.415 1.415 0 0 1 1.415 1.427zM4.992 3.336A1.781 1.781 0 0 1 3.21 5.135 1.781 1.781 0 0 1 1.427 3.336 1.781 1.781 0 0 1 3.21 1.537a1.781 1.781 0 0 1 1.782 1.799zM12 18.694c-3.945 0-7.394-1.417-9.191-3.506a9.799 9.799 0 0 0 18.382 0c-1.797 2.089-5.246 3.506-9.191 3.506zM12 5.306c3.945 0 7.394 1.417 9.191 3.506a9.799 9.799 0 0 0-18.382 0C4.606 6.723 8.055 5.306 12 5.306z"
      />
    </svg>
  );
}
function VscodeMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden focusable="false">
      <path
        fill="#007ACC"
        d="M23.15 2.587L18.21.21a1.494 1.494 0 0 0-1.705.29l-9.46 8.63-4.12-3.128a.999.999 0 0 0-1.276.057L.327 7.261A1 1 0 0 0 .326 8.74L3.899 12 .326 15.26a1 1 0 0 0 .001 1.479L1.65 17.94a.999.999 0 0 0 1.276.057l4.12-3.128 9.46 8.63a1.492 1.492 0 0 0 1.704.29l4.942-2.377A1.5 1.5 0 0 0 24 20.06V3.939a1.5 1.5 0 0 0-.85-1.352zm-5.146 14.861L10.826 12l7.178-5.448v10.896z"
      />
    </svg>
  );
}
function PytorchMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden focusable="false">
      <path
        fill="#EE4C2C"
        d="M12.005 0L4.952 7.053a9.865 9.865 0 0 0 0 13.945 9.866 9.866 0 0 0 13.946 0c3.515-3.515 3.515-9.21 0-12.724l-1.508 1.508c2.682 2.682 2.682 7.026 0 9.708a6.865 6.865 0 0 1-9.71 0 6.865 6.865 0 0 1 0-9.708l4.317-4.34.008.008V0zm3.291 4.388a1.184 1.184 0 1 0 0 2.368 1.184 1.184 0 0 0 0-2.368z"
      />
    </svg>
  );
}

function wsBrand(image: string): "jupyter" | "vscode" | "pytorch" {
  const s = (image || "").toLowerCase();
  if (s.includes("vscode") || s.includes("code-server")) return "vscode";
  if (s.includes("pytorch") || s.includes("torch")) return "pytorch";
  return "jupyter";
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
  const running = row.phase === "Running" || row.phase === "Degraded";
  const stopped = isStopped(row.phase);
  const brand = wsBrand(row.image);
  const Logo = brand === "vscode" ? VscodeMark : brand === "pytorch" ? PytorchMark : JupyterMark;

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
    <Card className="gap-0 p-4 transition-shadow hover:border-primary/30 hover:shadow-md">
      <div className="flex items-center gap-2.5">
        <div className="grid size-[38px] shrink-0 place-items-center rounded-[9px] border bg-muted">
          <Logo className="size-[22px]" />
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

      <div className="flex items-center gap-1">
        {stopped ? (
          <>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" onClick={() => onStart(row.name)} aria-label={t("workspaces.start")}>
                  <Play />
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
            {launch("terminal", t("workspaces.openTerminal"), <SquareTerminal />)}
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
const WS_IMAGES: { value: string; title: string; desc: string }[] = [
  { value: "jupyter-ds:2024.3", title: "jupyter-ds:2024.3", desc: "Jupyter 开发环境 · 公共" },
  { value: "pytorch:2.3-cu121", title: "pytorch:2.3-cu121", desc: "PyTorch 训练镜像" },
  { value: "vscode-server:1.90", title: "vscode-server:1.90", desc: "VS Code 开发环境 · 公共" },
];
const WS_UNITS: { value: string; pool: string; title: string; desc: string }[] = [
  { value: "cpu-medium", pool: "cpu-medium", title: "cpu-medium", desc: "8 vCPU · 32 GiB" },
  { value: "cpu-large", pool: "cpu-medium", title: "cpu-large", desc: "16 vCPU · 64 GiB" },
  { value: "a100-1x", pool: "gpu-a100", title: "a100-1x", desc: "1×A100 · 8 vCPU · 64 GiB" },
];

interface WsFormValues {
  name: string;
  description: string;
  image: string;
  unitName: string;
  containerPort: number;
  env: string;
  volSize: string;
  mountPath: string;
}

function parseEnv(text: string): sdk.EnvVar[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .map((l) => {
      const eq = l.indexOf("=");
      const name = (eq === -1 ? l : l.slice(0, eq)).trim();
      return { name, value: eq === -1 ? "" : l.slice(eq + 1).trim() };
    })
    .filter((e) => e.name);
}

function WsDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<WsFormValues>({
    name: "",
    description: "",
    image: WS_IMAGES[0].value,
    unitName: WS_UNITS[0].value,
    containerPort: 8888,
    env: "",
    volSize: "50Gi",
    mountPath: "/workspace",
  });
  const set = <K extends keyof WsFormValues>(k: K, val: WsFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const create = useApiMutation((body: sdk.WorkspaceCreateRequest) => sdk.createWorkspace({ body }), {
    invalidate: [["workspaces"]],
    success: t("workspaces.created"),
  });

  const submit = () => {
    setSubmitted(true);
    if (!v.name.trim()) return;
    const unit = WS_UNITS.find((u) => u.value === v.unitName) ?? WS_UNITS[0];
    const envVars = parseEnv(v.env || "");
    const mountPath = v.mountPath?.trim();
    const body: sdk.WorkspaceCreateRequest = {
      name: v.name.trim(),
      image: v.image,
      poolName: unit.pool,
      unitName: unit.value,
      description: v.description?.trim() || undefined,
      containerPort: v.containerPort && v.containerPort > 0 ? v.containerPort : undefined,
      env: envVars.length ? envVars : undefined,
      volumes: mountPath ? [{ mountPath, size: v.volSize?.trim() || undefined }] : undefined,
    };
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("workspaces.drawerNew")}</SheetTitle>
          <p className="text-xs text-muted-foreground">{t("workspaces.drawerNewSub")}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
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
              <FieldLabel>{t("workspaces.fImage")}</FieldLabel>
              <CardRadio options={WS_IMAGES} value={v.image} onChange={(val) => set("image", val)} />
            </Field>
          </FieldGroup>

          <FieldSection n={3} title={t("workspaces.fsResource")} />
          <FieldGroup>
            <Field>
              <FieldLabel>{t("workspaces.fUnit")}</FieldLabel>
              <CardRadio options={WS_UNITS} value={v.unitName} onChange={(val) => set("unitName", val)} />
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
              <Field className="w-40">
                <FieldLabel htmlFor="ws-volsize">{t("workspaces.fVolSize")}</FieldLabel>
                <Input
                  id="ws-volsize"
                  className="font-mono"
                  placeholder="50Gi"
                  value={v.volSize}
                  onChange={(e) => set("volSize", e.target.value)}
                />
              </Field>
              <Field className="flex-1">
                <FieldLabel htmlFor="ws-mount">{t("workspaces.fMountPath")}</FieldLabel>
                <Input
                  id="ws-mount"
                  className="font-mono"
                  placeholder="/workspace"
                  value={v.mountPath}
                  onChange={(e) => set("mountPath", e.target.value)}
                />
                <FieldDescription>{t("workspaces.fVolSizeHelp")}</FieldDescription>
              </Field>
            </div>
          </FieldGroup>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={create.isPending}>
            {create.isPending && <Spinner data-icon="inline-start" />}
            {t("workspaces.createWorkspace")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
