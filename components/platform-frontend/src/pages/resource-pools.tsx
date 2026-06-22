import { useMemo, useState } from "react";
import { Plus, Search, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useResourcePools } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { DataTable, type Column } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";

type PoolDrawer =
  | { kind: "new" }
  | { kind: "edit"; pool: sdk.ResourcePool }
  | { kind: "detail"; pool: sdk.ResourcePool }
  | { kind: "units"; pool: sdk.ResourcePool };

function selectorPairs(sel?: sdk.StringMap): string[] {
  return Object.entries(sel ?? {}).map(([k, v]) => `${k}=${v}`);
}

export default function ResourcePools() {
  const q = useResourcePools();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<PoolDrawer | null>(null);
  const [search, setSearch] = useState("");

  const delPool = useApiMutation((pool: string) => sdk.deleteResourcePool({ path: { pool } }), {
    invalidate: [["resourcepools"]],
    success: t("pools.deleted"),
  });

  const allRows = q.data?.items ?? [];
  const rows = useMemo(
    () => allRows.filter((p) => !search || p.name.includes(search)),
    [allRows, search],
  );

  const onDelete = (p: sdk.ResourcePool) =>
    confirm({
      title: t("pools.deleteTitle", { name: p.name }),
      desc: t("pools.deleteDesc"),
      info: t("pools.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delPool.mutate(p.name),
    });

  const columns: Column<sdk.ResourcePool>[] = [
    {
      key: "name",
      title: t("pools.colName"),
      render: (p) => (
        <button
          type="button"
          className="font-mono font-medium text-foreground hover:text-info hover:underline"
          onClick={() => setDrawer({ kind: "detail", pool: p })}
        >
          {p.name}
        </button>
      ),
    },
    {
      key: "description",
      title: t("pools.colDesc"),
      render: (p) => p.description || <span className="text-muted-foreground">—</span>,
    },
    {
      key: "selector",
      title: t("pools.colSelector"),
      render: (p) => {
        const pairs = selectorPairs(p.nodeSelector);
        if (!pairs.length) return <span className="text-muted-foreground">{t("pools.noSelector")}</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {pairs.map((s) => (
              <Badge key={s} variant="outline" className="font-mono">
                {s}
              </Badge>
            ))}
          </div>
        );
      },
    },
    {
      key: "units",
      title: t("pools.colUnits"),
      width: 100,
      align: "right",
      render: (p) => (
        <button
          type="button"
          className="text-info hover:underline"
          onClick={() => setDrawer({ kind: "units", pool: p })}
        >
          {p.units?.length ?? 0}
        </button>
      ),
    },
    {
      key: "createdAt",
      title: t("pools.colCreated"),
      width: 180,
      render: (p) => (
        <span className="text-muted-foreground">
          {p.createdAt ? dayjs(p.createdAt).format("YYYY-MM-DD HH:mm") : "—"}
        </span>
      ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 200,
      align: "right",
      render: (p) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" onClick={() => setDrawer({ kind: "detail", pool: p })}>
            {t("common.detail")}
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ kind: "edit", pool: p })}>
            {t("common.edit")}
          </Button>
          <Button variant="link" size="sm" onClick={() => setDrawer({ kind: "units", pool: p })}>
            {t("pools.manageUnits")}
          </Button>
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDelete(p)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.systemMgmt"), t("nav.pools")]}
      title={t("pools.title")}
      subtitle={t("pools.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ kind: "new" })}>
          <Plus data-icon="inline-start" />
          {t("pools.newPool")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <div className="relative max-w-xs flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("pools.searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Button variant="outline" onClick={() => setSearch("")}>
            {t("common.reset")}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(p) => p.name}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {drawer?.kind === "new" && <PoolFormDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "edit" && (
        <PoolFormDrawer pool={drawer.pool} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "detail" && (
        <PoolDetailDrawer pool={drawer.pool} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "units" && (
        <ManageUnitsDrawer pool={drawer.pool} onClose={() => setDrawer(null)} />
      )}
    </PageContainer>
  );
}

// ── Detail drawer (Tabs + key/value grid, read-only) ──────────────────────────
function DescRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </>
  );
}

function PoolDetailDrawer({ pool, onClose }: { pool: sdk.ResourcePool; onClose: () => void }) {
  const { t } = useTranslation();
  const pairs = selectorPairs(pool.nodeSelector);
  const units = pool.units ?? [];

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("pools.detailTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{pool.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <Tabs defaultValue="basic">
            <TabsList>
              <TabsTrigger value="basic">{t("pools.tabBasic")}</TabsTrigger>
              <TabsTrigger value="units">{t("pools.tabUnits")}</TabsTrigger>
            </TabsList>
            <TabsContent value="basic" className="pt-4">
              <dl className="grid grid-cols-[140px_1fr] gap-x-4 gap-y-2.5 text-sm">
                <DescRow label={t("pools.dName")}>
                  <span className="font-mono">{pool.name}</span>
                </DescRow>
                <DescRow label={t("pools.dDesc")}>
                  {pool.description || <span className="text-muted-foreground">—</span>}
                </DescRow>
                <DescRow label={t("pools.dSelector")}>
                  {pairs.length ? (
                    <div className="flex flex-wrap gap-1">
                      {pairs.map((s) => (
                        <Badge key={s} variant="outline" className="font-mono">
                          {s}
                        </Badge>
                      ))}
                    </div>
                  ) : (
                    <span className="text-muted-foreground">{t("pools.noSelector")}</span>
                  )}
                </DescRow>
                <DescRow label={t("pools.dNodeCount")}>{pool.nodeCount ?? "—"}</DescRow>
                <DescRow label={t("pools.dUnitCount")}>{units.length}</DescRow>
                <DescRow label={t("pools.dCreated")}>
                  {pool.createdAt ? dayjs(pool.createdAt).format("YYYY-MM-DD HH:mm") : "—"}
                </DescRow>
                <DescRow label={t("pools.dUpdated")}>
                  {pool.updatedAt ? dayjs(pool.updatedAt).format("YYYY-MM-DD HH:mm") : "—"}
                </DescRow>
              </dl>
            </TabsContent>
            <TabsContent value="units" className="pt-4">
              {units.length ? (
                <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
                  {units.map((u) => (
                    <Card key={u.name} className="gap-0 bg-muted p-3">
                      <div className="font-mono text-sm font-medium text-foreground">{u.name}</div>
                      {u.description && (
                        <div className="mt-0.5 text-xs text-muted-foreground">{u.description}</div>
                      )}
                      <div className="mt-2 flex flex-wrap gap-1">
                        {Object.entries(u.requests ?? {}).map(([k, v]) => (
                          <Badge key={k} variant="outline" className="font-mono">
                            {k}={v}
                          </Badge>
                        ))}
                      </div>
                    </Card>
                  ))}
                </div>
              ) : (
                <Empty>
                  <EmptyHeader>
                    <EmptyTitle>{t("pools.unitsEmpty")}</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              )}
            </TabsContent>
          </Tabs>
        </div>
      </SheetContent>
    </Sheet>
  );
}

// ── Pool create / edit drawer (numbered FieldSections, mirrors workspace form) ─
interface PoolFormValues {
  name: string;
  description: string;
  selector: string;
}

function parseSelector(s?: string): sdk.StringMap | undefined {
  const out: sdk.StringMap = {};
  for (const part of (s ?? "").split(",")) {
    const [k, ...rest] = part.split("=");
    const key = k.trim();
    if (key) out[key] = rest.join("=").trim();
  }
  return Object.keys(out).length ? out : undefined;
}

function PoolFormDrawer({ pool, onClose }: { pool?: sdk.ResourcePool; onClose: () => void }) {
  const { t } = useTranslation();
  const editing = !!pool;
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<PoolFormValues>({
    name: pool?.name ?? "",
    description: pool?.description ?? "",
    selector: selectorPairs(pool?.nodeSelector).join(", "),
  });
  const set = <K extends keyof PoolFormValues>(k: K, val: PoolFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const create = useApiMutation(
    (body: sdk.ResourcePoolCreateRequest) => sdk.createResourcePool({ body }),
    { invalidate: [["resourcepools"]], success: t("pools.created2") },
  );
  const update = useApiMutation(
    (vars: { pool: string; body: sdk.ResourcePoolPatchRequest }) =>
      sdk.updateResourcePool({ path: { pool: vars.pool }, body: vars.body }),
    { invalidate: [["resourcepools"]], success: t("pools.saved") },
  );
  const pending = create.isPending || update.isPending;

  const submit = () => {
    setSubmitted(true);
    const nodeSelector = parseSelector(v.selector);
    if (editing) {
      update.mutate(
        {
          pool: pool!.name,
          body: { description: v.description.trim() || undefined, nodeSelector },
        },
        { onSuccess: onClose },
      );
    } else {
      const name = v.name.trim();
      if (!name) return;
      create.mutate(
        {
          name,
          description: v.description.trim() || undefined,
          nodeSelector,
        },
        { onSuccess: onClose },
      );
    }
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{editing ? t("pools.drawerEdit") : t("pools.drawerNew")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            {editing ? <span className="font-mono">{pool!.name}</span> : t("pools.drawerNewSub")}
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("pools.fsBasic")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="pool-name">
                {t("pools.fName")}
                {!editing && <span className="text-destructive">*</span>}
              </FieldLabel>
              <Input
                id="pool-name"
                className="font-mono"
                placeholder={t("pools.fNamePlaceholder")}
                value={v.name}
                disabled={editing}
                aria-invalid={submitted && !editing && !v.name.trim()}
                onChange={(e) => set("name", e.target.value)}
              />
              {!editing && <FieldDescription>{t("pools.fNameHelp")}</FieldDescription>}
            </Field>
            <Field>
              <FieldLabel htmlFor="pool-desc">{t("pools.fDesc")}</FieldLabel>
              <Textarea
                id="pool-desc"
                rows={2}
                placeholder={t("pools.fDescPlaceholder")}
                value={v.description}
                onChange={(e) => set("description", e.target.value)}
              />
            </Field>
          </FieldGroup>

          <FieldSection n={2} title={t("pools.fsSchedule")} />
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="pool-selector">{t("pools.fSelector")}</FieldLabel>
              <Input
                id="pool-selector"
                className="font-mono"
                placeholder={t("pools.fSelectorPlaceholder")}
                value={v.selector}
                onChange={(e) => set("selector", e.target.value)}
              />
              <FieldDescription>{t("pools.fSelectorHelp")}</FieldDescription>
            </Field>
          </FieldGroup>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={pending}>
            {pending && <Spinner data-icon="inline-start" />}
            {editing ? t("common.save") : t("pools.createPool")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Manage-units drawer: CRUD the pool's inline units[] via local rows ─────────
interface UnitFormRow {
  name: string;
  description?: string;
  cpu?: number;
  memory?: number;
  gpu?: number;
}

function unitToRow(u: sdk.ResourceUnit): UnitFormRow {
  const num = (m: sdk.ResourceMap | undefined, k: string) => {
    const v = m?.[k];
    const n = v != null ? Number(v) : NaN;
    return Number.isFinite(n) ? n : undefined;
  };
  return {
    name: u.name,
    description: u.description,
    cpu: num(u.requests, "cpu"),
    memory: num(u.requests, "memory"),
    gpu: num(u.requests, "nvidia.com/gpu"),
  };
}

function rowToRequest(r: UnitFormRow): sdk.ResourceUnitCreateRequest {
  const map: sdk.ResourceMap = {};
  if (r.cpu != null) map["cpu"] = String(r.cpu);
  if (r.memory != null) map["memory"] = `${r.memory}Gi`;
  if (r.gpu != null && r.gpu > 0) map["nvidia.com/gpu"] = String(r.gpu);
  return {
    name: r.name.trim(),
    description: r.description?.trim() || undefined,
    requests: map,
    limits: map,
  };
}

function ManageUnitsDrawer({ pool, onClose }: { pool: sdk.ResourcePool; onClose: () => void }) {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const existing = useMemo(() => new Set((pool.units ?? []).map((u) => u.name)), [pool.units]);
  const [units, setUnits] = useState<UnitFormRow[]>(() => (pool.units ?? []).map(unitToRow));
  const setUnit = (i: number, patch: Partial<UnitFormRow>) =>
    setUnits(units.map((u, j) => (j === i ? { ...u, ...patch } : u)));
  const removeUnit = (i: number) => setUnits(units.filter((_, j) => j !== i));

  const createUnit = useApiMutation(
    (body: sdk.ResourceUnitCreateRequest) =>
      sdk.createResourceUnit({ path: { pool: pool.name }, body }),
    { invalidate: [["resourcepools"]], success: t("pools.unitsSaved") },
  );
  const delUnit = useApiMutation(
    (unit: string) => sdk.deleteResourceUnit({ path: { pool: pool.name, unit } }),
    { invalidate: [["resourcepools"]], success: t("pools.unitsSaved") },
  );

  const onFinish = () => {
    // Persist only newly added units; existing ones are managed via delete.
    const toCreate = units.filter((r) => r.name?.trim() && !existing.has(r.name.trim()));
    if (!toCreate.length) {
      onClose();
      return;
    }
    let done = 0;
    toCreate.forEach((r) =>
      createUnit.mutate(rowToRequest(r), {
        onSuccess: () => {
          done += 1;
          if (done === toCreate.length) onClose();
        },
      }),
    );
  };

  const removeExisting = (name: string, i: number) =>
    confirm({
      title: t("pools.deleteTitle", { name }),
      desc: t("pools.deleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => {
        delUnit.mutate(name);
        removeUnit(i);
      },
    });

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("pools.unitsDrawerTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{pool.name}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          {delUnit.isPending && (
            <div className="mb-3 flex justify-center">
              <Spinner className="size-5 text-muted-foreground" />
            </div>
          )}
          <p className="mb-4 text-xs text-muted-foreground">{t("pools.unitsDrawerSub")}</p>
          <FieldSection n={1} title={t("pools.fsUnits")} />
          <FieldGroup>
            <div className="flex flex-col gap-3">
            {units.map((row, i) => {
              const isExisting = !!row.name && existing.has(row.name);
              return (
                <Card key={i} className="gap-0 bg-muted p-3">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-semibold text-foreground">
                      {row.name || t("pools.uName")}
                    </span>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive"
                      onClick={() => (isExisting ? removeExisting(row.name, i) : removeUnit(i))}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <Field>
                      <FieldLabel>
                        {t("pools.uName")}
                        <span className="text-destructive">*</span>
                      </FieldLabel>
                      <Input
                        className="font-mono"
                        placeholder={t("pools.uNamePlaceholder")}
                        value={row.name}
                        disabled={isExisting}
                        onChange={(e) => setUnit(i, { name: e.target.value })}
                      />
                    </Field>
                    <Field>
                      <FieldLabel>{t("pools.uDesc")}</FieldLabel>
                      <Input
                        placeholder={t("pools.uDescPlaceholder")}
                        value={row.description ?? ""}
                        disabled={isExisting}
                        onChange={(e) => setUnit(i, { description: e.target.value })}
                      />
                    </Field>
                    <Field>
                      <FieldLabel>{`${t("pools.uCpu")} (${t("pools.uCpuUnit")})`}</FieldLabel>
                      <Input
                        type="number"
                        min={0}
                        value={row.cpu ?? ""}
                        disabled={isExisting}
                        onChange={(e) =>
                          setUnit(i, { cpu: e.target.value === "" ? undefined : Number(e.target.value) })
                        }
                      />
                    </Field>
                    <Field>
                      <FieldLabel>{`${t("pools.uMem")} (${t("pools.uMemUnit")})`}</FieldLabel>
                      <Input
                        type="number"
                        min={0}
                        value={row.memory ?? ""}
                        disabled={isExisting}
                        onChange={(e) =>
                          setUnit(i, { memory: e.target.value === "" ? undefined : Number(e.target.value) })
                        }
                      />
                    </Field>
                    <Field>
                      <FieldLabel>{`${t("pools.uGpu")} (${t("pools.uGpuUnit")})`}</FieldLabel>
                      <Input
                        type="number"
                        min={0}
                        value={row.gpu ?? ""}
                        disabled={isExisting}
                        onChange={(e) =>
                          setUnit(i, { gpu: e.target.value === "" ? undefined : Number(e.target.value) })
                        }
                      />
                    </Field>
                  </div>
                </Card>
              );
            })}
            <Button
              variant="outline"
              className="w-full border-dashed"
              onClick={() => setUnits([...units, { name: "" }])}
            >
              <Plus data-icon="inline-start" />
              {t("pools.addUnit")}
            </Button>
            </div>
          </FieldGroup>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={onFinish} disabled={createUnit.isPending}>
            {createUnit.isPending && <Spinner data-icon="inline-start" />}
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
