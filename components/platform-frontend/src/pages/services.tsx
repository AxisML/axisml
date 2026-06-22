import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Search, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  useServices,
  useModels,
  useImages,
  useResourcePools,
  useModelVersions,
  useImageVersions,
} from "@/api/hooks";
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
import { Switch } from "@/components/ui/switch";
import { Spinner } from "@/components/ui/spinner";
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

interface SvcRow {
  name: string;
  desc: string;
  phase?: string;
  replicas: string;
  replicaCount: number;
  poolName?: string;
  unitName?: string;
  unit: string;
  url?: string;
  running: boolean; // true → can stop; false → can start
  displayName?: string;
}

// Phases that mean the service is up enough to offer a "stop" action; the rest
// offer "start". The display label itself comes from the shared PhaseTag catalog.
const RUNNING_PHASES = new Set(["Ready", "Degraded", "Creating", "Pending"]);

type DrawerMode = "new" | "edit" | "scale";
const INVALIDATE = [["mlservices"]];
const ALL = "__all__";

export default function Services() {
  const q = useServices();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; row?: SvcRow } | null>(null);
  const [search, setSearch] = useState("");
  const [phase, setPhase] = useState<string>("");
  const [pool, setPool] = useState<string>("");

  const del = useApiMutation((name: string) => sdk.deleteMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.deleted"),
  });
  const start = useApiMutation((name: string) => sdk.startMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.starting"),
  });
  const stop = useApiMutation((name: string) => sdk.stopMlService({ path: { name } }), {
    invalidate: INVALIDATE,
    success: t("services.stopping"),
  });

  const allRows: SvcRow[] = useMemo(
    () =>
      q.data?.items?.map((s) => ({
        name: s.name,
        desc: s.description ?? s.displayName ?? "",
        phase: s.phase,
        replicas: `${s.readyReplicas ?? 0} / ${s.replicas ?? 0}`,
        replicaCount: s.replicas ?? 0,
        poolName: s.poolName,
        unitName: s.unitName,
        unit: `${s.poolName ?? "—"}/${s.unitName ?? "—"}`,
        url: s.accessUrl,
        running: RUNNING_PHASES.has(s.phase ?? ""),
        displayName: s.displayName,
      })) ?? [],
    [q.data],
  );

  const poolOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.poolName).filter((p): p is string => !!p))),
    [allRows],
  );
  const phaseOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.phase).filter((p): p is string => !!p))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) =>
      (!search || r.name.includes(search)) &&
      (!phase || r.phase === phase) &&
      (!pool || r.poolName === pool),
  );

  const onDelete = (r: SvcRow) =>
    confirm({
      title: t("services.deleteTitle", { name: r.name }),
      desc: t("services.deleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => del.mutate(r.name),
    });

  const columns: Column<SvcRow>[] = [
    {
      key: "name",
      title: t("services.colName"),
      render: (r) => (
        <div className="min-w-0">
          <Link
            to={`/services/${r.name}`}
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
      title: t("services.colStatus"),
      width: 120,
      render: (r) => <PhaseTag phase={r.phase} />,
    },
    {
      key: "replicas",
      title: t("services.colReplicas"),
      width: 100,
      align: "right",
      render: (r) => <span className="font-mono">{r.replicas}</span>,
    },
    {
      key: "unit",
      title: t("services.colUnit"),
      width: 180,
      render: (r) => <span className="font-mono text-muted-foreground">{r.unit}</span>,
    },
    {
      key: "url",
      title: t("services.colAccess"),
      render: (r) =>
        r.url ? (
          <span className="font-mono text-xs text-muted-foreground">{r.url}</span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 240,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" asChild>
            <Link to={`/services/${r.name}`}>{t("common.detail")}</Link>
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ mode: "edit", row: r })}>
            {t("common.edit")}
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ mode: "scale", row: r })}>
            {t("services.scale")}
          </Button>
          {r.running ? (
            <Button
              variant="link"
              size="sm"
              disabled={stop.isPending}
              onClick={() => stop.mutate(r.name)}
            >
              {t("services.stop")}
            </Button>
          ) : (
            <Button
              variant="link"
              size="sm"
              disabled={start.isPending}
              onClick={() => start.mutate(r.name)}
            >
              {t("services.start")}
            </Button>
          )}
          <Button
            variant="link"
            size="sm"
            className="text-destructive"
            onClick={() => onDelete(r)}
          >
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.serviceCenter"), t("nav.services")]}
      title={t("services.title")}
      subtitle={t("services.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ mode: "new" })}>
          <Plus data-icon="inline-start" />
          {t("services.newService")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <div className="relative max-w-xs flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("services.searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Select value={phase || ALL} onValueChange={(v) => setPhase(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-40">
              <SelectValue placeholder={t("services.statusAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("services.statusAll")}</SelectItem>
              {phaseOptions.map((p) => (
                <SelectItem key={p} value={p}>
                  {t(`phase.${p}`, { defaultValue: p })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={pool || ALL} onValueChange={(v) => setPool(v === ALL ? "" : v)}>
            <SelectTrigger className="min-w-40">
              <SelectValue placeholder={t("services.poolAll")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t("services.poolAll")}</SelectItem>
              {poolOptions.map((p) => (
                <SelectItem key={p} value={p}>
                  {p}
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
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(r) => r.name}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {drawer?.mode === "new" && <NewSvcDrawer onClose={() => setDrawer(null)} />}
      {drawer?.mode === "edit" && drawer.row && <EditSvcDrawer row={drawer.row} onClose={() => setDrawer(null)} />}
      {drawer?.mode === "scale" && drawer.row && <ScaleDrawer row={drawer.row} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Resource-unit spec helper (best-effort human label from requests map) ──────
function unitSpec(u: sdk.ResourceUnit): string {
  const r = (u.requests ?? {}) as Record<string, string | undefined>;
  const parts = Object.entries(r)
    .filter(([, v]) => !!v)
    .map(([k, v]) => `${k}=${v}`);
  return parts.join(" · ") || u.name;
}

// ── Controlled port rows feeding ServicePort[] ────────────────────────────────
interface PortRow {
  id: number;
  name: string;
  port: string;
}
let portSeq = 0;

function PortList({ rows, setRows }: { rows: PortRow[]; setRows: (fn: (r: PortRow[]) => PortRow[]) => void }) {
  const { t } = useTranslation();
  const add = () => setRows((r) => [...r, { id: ++portSeq, name: "", port: "" }]);
  const remove = (id: number) => setRows((r) => (r.length <= 1 ? r : r.filter((x) => x.id !== id)));
  const update = (id: number, patch: Partial<PortRow>) =>
    setRows((r) => r.map((x) => (x.id === id ? { ...x, ...patch } : x)));
  return (
    <div className="flex flex-col gap-2">
      {rows.map((row) => (
        <div className="flex items-center gap-2" key={row.id}>
          <Input
            className="font-mono"
            value={row.name}
            onChange={(e) => update(row.id, { name: e.target.value })}
            placeholder={t("services.fPortName")}
            maxLength={15}
          />
          <Input
            className="font-mono"
            value={row.port}
            onChange={(e) => update(row.id, { port: e.target.value })}
            placeholder={t("services.fPortNumber")}
            inputMode="numeric"
          />
          <Button variant="ghost" size="icon" onClick={() => remove(row.id)}>
            <X />
          </Button>
        </div>
      ))}
      <div>
        <Button variant="link" size="sm" className="px-0" onClick={add}>
          <Plus data-icon="inline-start" />
          {t("services.addPort")}
        </Button>
      </div>
    </div>
  );
}

function toServicePorts(rows: PortRow[]): sdk.ServicePort[] {
  return rows
    .filter((r) => r.name.trim() && r.port.trim())
    .map((r) => ({ name: r.name.trim(), port: Number(r.port) }));
}

// ── Create drawer ─────────────────────────────────────────────────────────────
function NewSvcDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const models = useModels();
  const images = useImages();
  const pools = useResourcePools();

  const create = useApiMutation((body: sdk.MlServiceCreateRequest) => sdk.createMlService({ body }), {
    invalidate: INVALIDATE,
    success: t("services.onlineProgress"),
  });

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [modelName, setModelName] = useState("");
  const [modelVersion, setModelVersion] = useState("");
  const [imageName, setImageName] = useState("");
  const [imageVersion, setImageVersion] = useState("");
  const [poolName, setPoolName] = useState("");
  const [unitName, setUnitName] = useState("");
  const [replicas, setReplicas] = useState<number>(1);
  const [ports, setPorts] = useState<PortRow[]>(() => [{ id: ++portSeq, name: "http", port: "8000" }]);
  const [routeEnabled, setRouteEnabled] = useState(true);
  const [routePath, setRoutePath] = useState("");

  const modelVersions = useModelVersions(modelName);
  const imageVersions = useImageVersions(imageName);

  const modelOptions = useMemo(
    () => (models.data?.items ?? []).map((m) => ({ label: m.displayName || m.name, value: m.name })),
    [models.data],
  );
  const modelVersionOptions = useMemo(
    () => (modelVersions.data?.items ?? []).map((v) => ({ label: v.version, value: v.version })),
    [modelVersions.data],
  );
  const imageOptions = useMemo(
    () => (images.data?.items ?? []).map((i) => ({ label: i.displayName || i.name, value: i.name })),
    [images.data],
  );
  const imageVersionOptions = useMemo(
    () =>
      (imageVersions.data?.items ?? []).map((v) => ({
        label: v.version,
        value: v.version,
        uri: v.uri || `${imageName}:${v.version}`,
      })),
    [imageVersions.data, imageName],
  );
  const selectedImage = imageVersionOptions.find((v) => v.value === imageVersion)?.uri ?? "";

  const poolOptions = useMemo(
    () =>
      (pools.data?.items ?? []).map((p) => ({
        label: p.description ? `${p.name} · ${p.description}` : p.name,
        value: p.name,
      })),
    [pools.data],
  );
  const unitOptions = useMemo(() => {
    const pool = (pools.data?.items ?? []).find((p) => p.name === poolName);
    return (pool?.units ?? []).map((u) => ({ value: u.name, title: u.name, desc: unitSpec(u) }));
  }, [pools.data, poolName]);

  const onPickModel = (v: string) => {
    setModelName(v);
    setModelVersion("");
  };
  const onPickImage = (v: string) => {
    setImageName(v);
    setImageVersion("");
  };
  const onPickPool = (v: string) => {
    setPoolName(v);
    setUnitName("");
  };

  const validPorts = toServicePorts(ports);
  const canSubmit =
    !!name.trim() &&
    !!modelName &&
    !!modelVersion &&
    !!selectedImage &&
    !!poolName &&
    !!unitName &&
    replicas >= 0 &&
    validPorts.length > 0 &&
    !create.isPending;

  const submit = () => {
    const route: sdk.MlServiceRoute = routeEnabled
      ? { enabled: true, ...(routePath.trim() ? { path: routePath.trim() } : {}) }
      : { enabled: false };
    const body: sdk.MlServiceCreateRequest = {
      name: name.trim(),
      modelName,
      modelVersion,
      image: selectedImage,
      poolName,
      unitName,
      replicas,
      ports: validPorts,
      ...(description.trim() ? { description: description.trim() } : {}),
      route,
    };
    create.mutate(body, { onSuccess: onClose });
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("services.drawerNew")}</SheetTitle>
          <p className="text-xs text-muted-foreground">{t("services.drawerNewSub")}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("services.fsBasic")} />
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-name">
              {t("services.fName")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="svc-name"
              className="font-mono"
              placeholder={t("services.fNamePlaceholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <FieldDescription>{t("services.fNameHelp")}</FieldDescription>
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-desc">{t("services.fDesc")}</FieldLabel>
            <Textarea
              id="svc-desc"
              rows={2}
              placeholder={t("services.fDescPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Field>

          <FieldSection n={2} title={t("services.fsModelImage")} />
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>
                {t("services.fModel")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Select value={modelName || undefined} onValueChange={onPickModel}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("services.selectModel")} />
                </SelectTrigger>
                <SelectContent>
                  {modelOptions.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>
                {t("services.fModelVersion")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Select
                value={modelVersion || undefined}
                onValueChange={setModelVersion}
                disabled={!modelName}
              >
                <SelectTrigger className="w-full">
                  <SelectValue
                    placeholder={
                      !modelName
                        ? t("services.selectModelFirst")
                        : modelVersionOptions.length === 0
                          ? t("services.noVersion")
                          : t("services.selectVersion")
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {modelVersionOptions.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>
                {t("services.fImage")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Select value={imageName || undefined} onValueChange={onPickImage}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("services.selectImage")} />
                </SelectTrigger>
                <SelectContent>
                  {imageOptions.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>
                {t("services.fImageVersion")}
                <span className="text-destructive">*</span>
              </FieldLabel>
              <Select
                value={imageVersion || undefined}
                onValueChange={setImageVersion}
                disabled={!imageName}
              >
                <SelectTrigger className="w-full">
                  <SelectValue
                    placeholder={
                      !imageName
                        ? t("services.selectImageFirst")
                        : imageVersionOptions.length === 0
                          ? t("services.noVersion")
                          : t("services.selectVersion")
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {imageVersionOptions.map((v) => (
                    <SelectItem key={v.value} value={v.value}>
                      {v.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          </div>

          <FieldSection n={3} title={t("services.fsResource")} />
          <Field className="mb-4">
            <FieldLabel>
              {t("services.fPool")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Select value={poolName || undefined} onValueChange={onPickPool}>
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("services.selectPool")} />
              </SelectTrigger>
              <SelectContent>
                {poolOptions.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field className="mb-4">
            <FieldLabel>
              {t("services.fUnit")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            {unitOptions.length === 0 ? (
              <span className="text-sm text-muted-foreground">{t("services.pickPoolFirst")}</span>
            ) : (
              <CardRadio options={unitOptions} value={unitName} onChange={setUnitName} />
            )}
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-replicas">
              {t("services.fReplicas")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="svc-replicas"
              type="number"
              min={0}
              className="w-40"
              value={replicas}
              onChange={(e) => setReplicas(Number(e.target.value))}
            />
          </Field>

          <FieldSection n={4} title={t("services.fsPortRoute")} />
          <Field className="mb-4">
            <FieldLabel>
              {t("services.fPorts")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <PortList rows={ports} setRows={setPorts} />
          </Field>
          <div className="mb-4 flex items-center justify-between">
            <div>
              <div className="text-sm">{t("services.fRouteEnabled")}</div>
              <div className="text-xs text-muted-foreground">{t("services.fRouteHelp")}</div>
            </div>
            <Switch checked={routeEnabled} onCheckedChange={setRouteEnabled} />
          </div>
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-path">{t("services.fPath")}</FieldLabel>
            <Input
              id="svc-path"
              className="font-mono"
              placeholder={t("services.fPathPlaceholder")}
              value={routePath}
              disabled={!routeEnabled}
              onChange={(e) => setRoutePath(e.target.value)}
            />
          </Field>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!canSubmit} onClick={submit}>
            {create.isPending && <Spinner data-icon="inline-start" />}
            {t("services.online")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Edit drawer (display metadata only) ───────────────────────────────────────
function EditSvcDrawer({ row, onClose }: { row: SvcRow; onClose: () => void }) {
  const { t } = useTranslation();
  const [displayName, setDisplayName] = useState(row.displayName ?? "");
  const [description, setDescription] = useState(row.desc ?? "");
  const update = useApiMutation(
    (body: sdk.MlServicePatchRequest) => sdk.updateMlService({ path: { name: row.name }, body }),
    { invalidate: INVALIDATE, success: t("services.saved") },
  );

  const submit = () =>
    update.mutate(
      {
        displayName: displayName.trim() || undefined,
        description: description.trim() || undefined,
      },
      { onSuccess: onClose },
    );

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("services.drawerEdit")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{row.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("services.fsBasic")} />
          <p className="mb-4 text-sm text-muted-foreground">{t("services.editNote")}</p>
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-display">{t("services.fDisplayName")}</FieldLabel>
            <Input
              id="svc-display"
              placeholder={t("services.fDisplayNamePlaceholder")}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-edit-desc">{t("services.fDesc")}</FieldLabel>
            <Textarea
              id="svc-edit-desc"
              rows={2}
              placeholder={t("services.fDescPlaceholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Field>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={update.isPending} onClick={submit}>
            {update.isPending && <Spinner data-icon="inline-start" />}
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Scale drawer ──────────────────────────────────────────────────────────────
function ScaleDrawer({ row, onClose }: { row: SvcRow; onClose: () => void }) {
  const { t } = useTranslation();
  const [replicas, setReplicas] = useState<number>(row.replicaCount);
  const scale = useApiMutation(
    (body: sdk.MlServiceScaleRequest) => sdk.scaleMlService({ path: { name: row.name }, body }),
    { invalidate: INVALIDATE, success: t("services.scaleSubmitted") },
  );

  const valid = Number.isInteger(replicas) && replicas >= 0;
  const submit = () => scale.mutate({ replicas }, { onSuccess: onClose });

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[420px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("services.drawerScale")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{row.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <p className="mb-5 text-sm text-muted-foreground">{t("services.scaleNote")}</p>
          <Field className="mb-4">
            <FieldLabel htmlFor="svc-scale-replicas">{t("services.fTargetReplicas")}</FieldLabel>
            <Input
              id="svc-scale-replicas"
              type="number"
              min={0}
              className="w-40"
              value={replicas}
              onChange={(e) => setReplicas(Number(e.target.value))}
            />
            <FieldDescription>
              {t("services.scaleHint", { ready: row.replicas, unit: row.unit })}
            </FieldDescription>
          </Field>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!valid || scale.isPending} onClick={submit}>
            {scale.isPending && <Spinner data-icon="inline-start" />}
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
