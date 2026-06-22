import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus, Search, Ban, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useTenants } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { errorText } from "@/lib/errors";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";
import { PhaseTag } from "@/components/PhaseTag";
import { DataTable, type Column } from "@/components/DataTable";
import { USE_MOCK } from "@/api/mock";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
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
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// Placeholder pool / unit catalogs for the quota editor's Select options. The
// authoritative source is the ResourcePool CRD; until the create-tenant flow is
// wired to live pools, these mirror the prototype's sample pools/units.
const POOL_OPTIONS = ["gpu-h100", "gpu-a100", "cpu-large"];
const UNIT_OPTIONS = ["h100-4x-xlarge", "h100-8x-xlarge-ib", "a100-1x-large", "a100-4x-xlarge", "cpu-large-1"];
const MEMBER_ROLES: ("user" | "tenant-admin")[] = ["user", "tenant-admin"];

interface TenantRow {
  ident: string;
  display: string;
  active: boolean;
  pools: { pool: string; allocated: number; used?: number }[];
  members: number;
  activeTasks: number;
  services: number;
  created: string;
}

// Demo-only quota utilisation ratio (deterministic per pool name). Real quota
// usage has no metrics source, so the meter only renders under mock; otherwise
// the column shows the honest allocated-quantity text.
function mockUsedRatio(pool: string): number {
  const h = [...pool].reduce((a, c) => a + c.charCodeAt(0), 0);
  return 0.5 + ((h % 45) / 100); // 0.50 – 0.94
}

// Per-pool quota row: pool name + (under mock) a used/allocated meter, else the
// honest allocated-quantity text. Mirrors the prototype's quota usage bars.
function QuotaBar({ pool, allocated, used }: { pool: string; allocated: number; used?: number }) {
  if (used == null) {
    return (
      <div className="flex items-center gap-2">
        <span className="w-[88px] shrink-0 truncate font-mono text-xs text-muted-foreground">{pool}</span>
        <span className="font-mono text-xs text-muted-foreground">{allocated} 单元</span>
      </div>
    );
  }
  const pct = allocated === 0 ? 0 : Math.min(100, Math.round((used / allocated) * 100));
  const fill = pct >= 80 ? "bg-destructive" : pct >= 60 ? "bg-warning" : "bg-success";
  return (
    <div className="flex items-center gap-2">
      <span className="w-[88px] shrink-0 truncate font-mono text-xs text-muted-foreground">{pool}</span>
      <div className="h-[7px] flex-1 overflow-hidden rounded-full bg-muted">
        <div className={`h-full rounded-full ${fill}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="w-11 shrink-0 text-right font-mono text-xs text-muted-foreground">{used}/{allocated}</span>
    </div>
  );
}

type DrawerKind =
  | { kind: "tenant" }
  | { kind: "quota"; ident: string; display: string }
  | { kind: "members"; ident: string; display: string }
  | { kind: "member"; ident: string };

export default function Tenants() {
  const q = useTenants();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<"all" | "active" | "suspended">("all");

  const delTenant = useApiMutation((name: string) => sdk.deleteTenant({ path: { name } }), {
    invalidate: [["tenants"]],
    success: t("tenants.deleted"),
  });
  const suspend = useApiMutation((name: string) => sdk.suspendTenant({ path: { name } }), {
    invalidate: [["tenants"]],
    success: t("tenants.suspended"),
  });
  const resume = useApiMutation((name: string) => sdk.resumeTenant({ path: { name } }), {
    invalidate: [["tenants"]],
    success: t("tenants.resumed"),
  });

  const allRows: TenantRow[] = useMemo(
    () =>
      q.data?.items?.map((tenant) => ({
        ident: tenant.identifier,
        display: tenant.displayName,
        active: !tenant.suspended,
        pools: (tenant.quotas ?? []).map((quota) => {
          const allocated = (quota.units ?? []).reduce((sum, u) => sum + (u.quantity ?? 0), 0);
          return {
            pool: quota.pool,
            allocated,
            used: USE_MOCK ? Math.round(allocated * mockUsedRatio(quota.pool)) : undefined,
          };
        }),
        members: tenant.memberCount ?? 0,
        activeTasks: (tenant.activeJobRuns ?? 0) + (tenant.activeExperimentRuns ?? 0),
        services: tenant.onlineServices ?? 0,
        created: tenant.createdAt,
      })) ?? [],
    [q.data],
  );

  const rows = useMemo(
    () =>
      allRows.filter(
        (r) =>
          (!search || r.ident.includes(search) || r.display.includes(search)) &&
          (status === "all" || (status === "active" ? r.active : !r.active)),
      ),
    [allRows, search, status],
  );

  const onSuspend = (r: TenantRow) =>
    confirm({
      title: t("tenants.suspendTitle", { name: r.ident }),
      desc: t("tenants.suspendDesc"),
      info: t("tenants.suspendInfo"),
      okLabel: t("tenants.confirmSuspend"),
      danger: false,
      onConfirm: () => suspend.mutate(r.ident),
    });

  const onDelete = (r: TenantRow) =>
    confirm({
      title: t("tenants.deleteTitle", { name: r.ident }),
      desc: t("tenants.deleteDesc"),
      info: t("tenants.deleteInfo"),
      okLabel: t("tenants.confirmDelete"),
      onConfirm: () => delTenant.mutate(r.ident),
    });

  const columns: Column<TenantRow>[] = [
    {
      key: "ident",
      title: t("tenants.colTenant"),
      render: (r) => (
        <div className="min-w-0">
          <button
            type="button"
            className="font-mono font-medium text-foreground hover:text-info hover:underline"
            onClick={() => setDrawer({ kind: "quota", ident: r.ident, display: r.display })}
          >
            {r.ident}
          </button>
          <div className="truncate text-xs text-muted-foreground">{r.display}</div>
        </div>
      ),
    },
    {
      key: "status",
      title: t("tenants.colStatus"),
      width: 110,
      render: (r) => <PhaseTag phase={r.active ? "Active" : "Suspended"} />,
    },
    {
      key: "quota",
      title: t("tenants.colQuota"),
      width: 240,
      render: (r) =>
        r.pools.length === 0 ? (
          <span className="text-muted-foreground">{t("tenants.noQuota")}</span>
        ) : (
          <div className="flex flex-col gap-1.5">
            {r.pools.map((p) => (
              <QuotaBar key={p.pool} pool={p.pool} allocated={p.allocated} used={p.used} />
            ))}
          </div>
        ),
    },
    { key: "members", title: t("tenants.colMembers"), dataIndex: "members", width: 80, align: "right" },
    { key: "activeTasks", title: t("tenants.colActiveTasks"), dataIndex: "activeTasks", width: 90, align: "right" },
    { key: "services", title: t("tenants.colServices"), dataIndex: "services", width: 90, align: "right" },
    {
      key: "created",
      title: t("tenants.colCreated"),
      width: 160,
      render: (r) => (
        <span className="text-muted-foreground">{r.created ? dayjs(r.created).format("YYYY-MM-DD") : "—"}</span>
      ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 240,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button
            variant="link"
            size="sm"
            onClick={() => setDrawer({ kind: "quota", ident: r.ident, display: r.display })}
          >
            {t("tenants.editQuota")}
          </Button>
          <Button
            variant="link"
            size="sm"
            onClick={() => setDrawer({ kind: "members", ident: r.ident, display: r.display })}
          >
            {t("tenants.manageMembers")}
          </Button>
          <Button
            variant="link"
            size="sm"
            onClick={() => setDrawer({ kind: "member", ident: r.ident })}
          >
            {t("tenants.addMember")}
          </Button>
          {r.active ? (
            <Button variant="link" size="sm" onClick={() => onSuspend(r)}>
              {t("tenants.suspend")}
            </Button>
          ) : (
            <Button variant="link" size="sm" onClick={() => resume.mutate(r.ident)}>
              {t("tenants.resume")}
            </Button>
          )}
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.systemMgmt"), t("nav.tenants")]}
      title={t("tenants.title")}
      subtitle={t("tenants.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ kind: "tenant" })}>
          <Plus data-icon="inline-start" />
          {t("tenants.newTenant")}
        </Button>
      }
    >
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-center gap-3 border-b p-4">
          <div className="relative max-w-xs flex-1">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("tenants.searchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <Select value={status} onValueChange={(v) => setStatus(v as typeof status)}>
            <SelectTrigger className="min-w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("tenants.statusAll")}</SelectItem>
              <SelectItem value="active">{t("tenants.statusActive")}</SelectItem>
              <SelectItem value="suspended">{t("tenants.statusSuspended")}</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            onClick={() => {
              setSearch("");
              setStatus("all");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          rowKey={(r) => r.ident}
          loading={q.isLoading}
          error={q.isError}
        />
      </Card>

      {drawer?.kind === "tenant" && <TenantDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "quota" && (
        <QuotaDrawer ident={drawer.ident} display={drawer.display} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "members" && (
        <MembersDrawer
          ident={drawer.ident}
          display={drawer.display}
          onAddMember={() => setDrawer({ kind: "member", ident: drawer.ident })}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === "member" && <MemberDrawer ident={drawer.ident} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Create-tenant drawer (numbered FieldSections → createTenant) ───────────────
interface TenantFormValues {
  displayName: string;
  identifier: string;
  initialAdmin: string;
}

function TenantDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<TenantFormValues>({
    displayName: "",
    identifier: "",
    initialAdmin: "",
  });
  const set = <K extends keyof TenantFormValues>(k: K, val: TenantFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const create = useApiMutation((body: sdk.TenantCreateRequest) => sdk.createTenant({ body }), {
    invalidate: [["tenants"]],
    success: t("tenants.created"),
  });

  const submit = () => {
    setSubmitted(true);
    const display = v.displayName.trim();
    const ident = v.identifier.trim();
    const admin = v.initialAdmin.trim();
    if (!display || !ident || !admin) return;
    create.mutate(
      {
        displayName: display,
        identifier: ident,
        initialAdmin: admin,
        // The identifier is a dns1123 slug; reuse it as the physical namespace
        // (tenant name = namespace convention).
        kubernetesNamespace: ident,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[560px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("tenants.drawerNew")}</SheetTitle>
          <p className="text-xs text-muted-foreground">{t("tenants.drawerNewSub")}</p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("tenants.fsBasic")} />
          <Field className="mb-4">
            <FieldLabel htmlFor="tenant-display">
              {t("tenants.fDisplayName")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="tenant-display"
              placeholder={t("tenants.fDisplayNamePlaceholder")}
              value={v.displayName}
              aria-invalid={submitted && !v.displayName.trim()}
              onChange={(e) => set("displayName", e.target.value)}
            />
            <FieldDescription>{t("tenants.fDisplayNameHelp")}</FieldDescription>
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="tenant-ident">
              {t("tenants.fIdentifier")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="tenant-ident"
              className="font-mono"
              placeholder={t("tenants.fIdentifierPlaceholder")}
              value={v.identifier}
              aria-invalid={submitted && !v.identifier.trim()}
              onChange={(e) => set("identifier", e.target.value)}
            />
            <FieldDescription>{t("tenants.fIdentifierHelp")}</FieldDescription>
          </Field>
          <Field className="mb-4">
            <FieldLabel htmlFor="tenant-admin">
              {t("tenants.fAdmin")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="tenant-admin"
              placeholder={t("tenants.fAdminPlaceholder")}
              value={v.initialAdmin}
              aria-invalid={submitted && !v.initialAdmin.trim()}
              onChange={(e) => set("initialAdmin", e.target.value)}
            />
          </Field>

          <FieldSection n={2} title={t("tenants.fsQuota")} />
          <p className="text-xs text-muted-foreground">{t("tenants.quotaHint")}</p>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={create.isPending}>
            {create.isPending && <Spinner data-icon="inline-start" />}
            {t("tenants.createTenant")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Quota editor drawer (live quotas → rows grouped by pool) ───────────────────
interface QuotaRow {
  pool?: string;
  unitName?: string;
  quantity?: number;
}

function QuotaDrawer({ ident, display, onClose }: { ident: string; display: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { confirm } = useUI();

  const quotasQ = useQuery({
    queryKey: ["tenant-quotas", ident],
    queryFn: async () => {
      const { data, error } = await sdk.listTenantQuotas({ path: { name: ident } });
      if (error) throw new Error(errorText(error));
      return data;
    },
  });

  const updateQuota = useApiMutation(
    (arg: { pool: string; units: sdk.QuotaUnit[] }) =>
      sdk.updateTenantQuota({ path: { name: ident, pool: arg.pool }, body: { units: arg.units } }),
    { invalidate: [["tenant-quotas", ident], ["tenants"]], success: t("tenants.quotaSaved") },
  );
  const createQuota = useApiMutation(
    (arg: { pool: string; units: sdk.QuotaUnit[] }) =>
      sdk.createTenantQuota({ path: { name: ident }, body: { pool: arg.pool, units: arg.units } }),
    { invalidate: [["tenant-quotas", ident], ["tenants"]], success: t("tenants.quotaSaved") },
  );
  const delQuota = useApiMutation((pool: string) => sdk.deleteTenantQuota({ path: { name: ident, pool } }), {
    invalidate: [["tenant-quotas", ident], ["tenants"]],
    success: t("tenants.quotaRemoved"),
  });

  const quotas = quotasQ.data?.items ?? [];
  const existingPools = useMemo(() => new Set(quotas.map((quota) => quota.pool)), [quotas]);

  const initialRows: QuotaRow[] = useMemo(
    () =>
      quotas.flatMap((quota) =>
        (quota.units ?? []).map((u) => ({
          pool: quota.pool,
          unitName: u.unitName,
          quantity: u.quantity,
        })),
      ),
    [quotas],
  );

  // Editable rows hydrate once from live quotas (mirrors antd Form.List initialValues).
  const [rows, setRows] = useState<QuotaRow[] | null>(null);
  const editRows = rows ?? initialRows;
  const setRow = (i: number, patch: Partial<QuotaRow>) =>
    setRows(editRows.map((r, j) => (j === i ? { ...r, ...patch } : r)));
  const addRow = () => setRows([...editRows, { quantity: 0 }]);
  const removeRow = (i: number) => setRows(editRows.filter((_, j) => j !== i));

  const onFinish = () => {
    // Group rows by pool, then upsert each pool (PATCH if it already exists,
    // POST otherwise). Backend owns the (pool/unit/quantity) vocabulary.
    const byPool = new Map<string, sdk.QuotaUnit[]>();
    for (const r of editRows) {
      const pool = r.pool?.trim();
      const unitName = r.unitName?.trim();
      if (!pool || !unitName) continue;
      const units = byPool.get(pool) ?? [];
      units.push({ unitName, quantity: Math.max(0, Number(r.quantity) || 0) });
      byPool.set(pool, units);
    }
    if (byPool.size === 0) {
      onClose();
      return;
    }
    let done = 0;
    const total = byPool.size;
    const tick = () => {
      done += 1;
      if (done === total) onClose();
    };
    for (const [pool, units] of byPool) {
      const mutation = existingPools.has(pool) ? updateQuota : createQuota;
      mutation.mutate({ pool, units }, { onSuccess: tick });
    }
  };

  const onRemovePool = (pool: string) =>
    confirm({
      title: t("tenants.removePoolTitle", { pool }),
      desc: t("tenants.removePoolDesc"),
      okLabel: t("tenants.confirmRemovePool"),
      onConfirm: () => delQuota.mutate(pool),
    });

  const saving = updateQuota.isPending || createQuota.isPending;

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[680px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("tenants.quotaDrawerTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{ident}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <p className="mb-4 text-xs text-muted-foreground">{t("tenants.quotaDrawerSub", { name: display })}</p>
          {quotasQ.isLoading ? (
            <div className="flex justify-center py-8">
              <Spinner className="size-7 text-muted-foreground" />
            </div>
          ) : quotasQ.isError ? (
            <Empty>
              <EmptyHeader>
                <EmptyTitle>{t("common.loadFailed")}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          ) : (
            <>
              {delQuota.isPending && (
                <div className="mb-3 flex justify-center">
                  <Spinner className="size-5 text-muted-foreground" />
                </div>
              )}
              <FieldSection n={1} title={t("tenants.fsQuotaRows")} />
              <p className="mb-3 text-xs text-muted-foreground">{t("tenants.quotaRowsSub")}</p>
              <div className="flex flex-col gap-3">
                {editRows.length === 0 && (
                  <Empty>
                    <EmptyHeader>
                      <EmptyTitle>{t("tenants.quotaEmpty")}</EmptyTitle>
                    </EmptyHeader>
                  </Empty>
                )}
                {editRows.map((row, i) => {
                  const canRemovePool = !!row.pool && existingPools.has(row.pool);
                  return (
                    <Card key={i} className="gap-0 bg-muted p-3">
                      <div className="grid grid-cols-[1fr_1fr_auto_auto] items-end gap-x-3">
                        <Field>
                          <FieldLabel>{t("tenants.qPool")}</FieldLabel>
                          <Select
                            value={row.pool}
                            onValueChange={(val) => setRow(i, { pool: val })}
                          >
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder={t("tenants.qPoolPlaceholder")} />
                            </SelectTrigger>
                            <SelectContent>
                              {POOL_OPTIONS.map((p) => (
                                <SelectItem key={p} value={p}>
                                  {p}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </Field>
                        <Field>
                          <FieldLabel>{t("tenants.qUnit")}</FieldLabel>
                          <Select
                            value={row.unitName}
                            onValueChange={(val) => setRow(i, { unitName: val })}
                          >
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder={t("tenants.qUnitPlaceholder")} />
                            </SelectTrigger>
                            <SelectContent>
                              {UNIT_OPTIONS.map((u) => (
                                <SelectItem key={u} value={u}>
                                  {u}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </Field>
                        <Field>
                          <FieldLabel>{t("tenants.qQuantity")}</FieldLabel>
                          <Input
                            type="number"
                            min={0}
                            className="w-24"
                            value={row.quantity ?? ""}
                            onChange={(e) => setRow(i, { quantity: Number(e.target.value) })}
                          />
                        </Field>
                        <div className="flex items-center gap-1 pb-1">
                          {canRemovePool && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon-sm"
                                  onClick={() => onRemovePool(row.pool!)}
                                >
                                  <Ban />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>{t("tenants.removePool")}</TooltipContent>
                            </Tooltip>
                          )}
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            className="text-destructive"
                            onClick={() => removeRow(i)}
                          >
                            <Trash2 />
                          </Button>
                        </div>
                      </div>
                    </Card>
                  );
                })}
                <Button variant="outline" className="w-full border-dashed" onClick={addRow}>
                  <Plus data-icon="inline-start" />
                  {t("tenants.addQuotaRow")}
                </Button>
              </div>
            </>
          )}
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={onFinish} disabled={saving}>
            {saving && <Spinner data-icon="inline-start" />}
            {t("tenants.saveQuota")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Members drawer (live members → remove / update role) ───────────────────────
function MembersDrawer({
  ident,
  display,
  onAddMember,
  onClose,
}: {
  ident: string;
  display: string;
  onAddMember: () => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [search, setSearch] = useState("");

  const membersQ = useQuery({
    queryKey: ["tenant-members", ident],
    queryFn: async () => {
      const { data, error } = await sdk.listTenantMembers({ path: { name: ident } });
      if (error) throw new Error(errorText(error));
      return data;
    },
  });

  const removeMember = useApiMutation(
    (userId: string) => sdk.removeTenantMember({ path: { name: ident, userId } }),
    { invalidate: [["tenant-members", ident]], success: t("tenants.memberRemoved") },
  );
  const updateRole = useApiMutation(
    (arg: { userId: string; roleName: "tenant-admin" | "user" }) =>
      sdk.updateTenantMember({ path: { name: ident, userId: arg.userId }, body: { roleName: arg.roleName } }),
    { invalidate: [["tenant-members", ident]], success: t("tenants.roleUpdated") },
  );

  const roleLabel = (r: string) => t(`role.${r}`, { defaultValue: r });
  const allMembers = membersQ.data?.items ?? [];
  const members = useMemo(
    () =>
      allMembers.filter((m) => {
        if (!search) return true;
        const name = m.displayName || m.username || m.email || m.userId;
        return name.includes(search);
      }),
    [allMembers, search],
  );

  const columns: Column<sdk.Member>[] = [
    {
      key: "member",
      title: t("tenants.mColMember"),
      render: (m) => {
        const name = m.displayName || m.username || m.email || m.userId;
        const initial = (name || "?").trim().charAt(0).toUpperCase();
        return (
          <div className="flex items-center gap-2.5">
            <span className="grid size-7 place-items-center rounded-full bg-info text-xs font-semibold text-info-foreground">
              {initial}
            </span>
            <span className="text-sm text-foreground">{name}</span>
          </div>
        );
      },
    },
    {
      key: "role",
      title: t("tenants.mColRole"),
      width: 160,
      render: (m) => (
        <Select
          value={m.roleName === "tenant-admin" ? "tenant-admin" : "user"}
          disabled={updateRole.isPending}
          onValueChange={(val) => updateRole.mutate({ userId: m.userId, roleName: val as "tenant-admin" | "user" })}
        >
          <SelectTrigger size="sm" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {MEMBER_ROLES.map((r) => (
              <SelectItem key={r} value={r}>
                {roleLabel(r)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ),
    },
    {
      key: "addedAt",
      title: t("tenants.mColJoined"),
      width: 120,
      render: (m) => (
        <span className="text-muted-foreground">{m.addedAt ? dayjs(m.addedAt).format("YYYY-MM-DD") : "—"}</span>
      ),
    },
    {
      key: "actions",
      title: t("common.actions"),
      width: 90,
      align: "right",
      render: (m) => {
        const name = m.displayName || m.username || m.email || m.userId;
        return (
          <Button
            variant="link"
            size="sm"
            className="text-destructive"
            onClick={() =>
              confirm({
                title: t("tenants.removeMemberTitle", { name }),
                desc: t("tenants.removeMemberDesc"),
                okLabel: t("tenants.confirmRemoveMember"),
                onConfirm: () => removeMember.mutate(m.userId),
              })
            }
          >
            {t("tenants.removeMember")}
          </Button>
        );
      },
    },
  ];

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[680px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("tenants.membersDrawerTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{ident}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <div className="mb-4 flex items-center justify-between gap-3">
            <p className="m-0 text-xs text-muted-foreground">{t("tenants.membersDrawerSub", { name: display })}</p>
            <Button size="sm" onClick={onAddMember}>
              <Plus data-icon="inline-start" />
              {t("tenants.addMember")}
            </Button>
          </div>
          <div className="relative mb-3 max-w-xs">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-8"
              placeholder={t("tenants.memberSearchPlaceholder")}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <DataTable
            columns={columns}
            data={members}
            rowKey={(m) => m.userId}
            loading={membersQ.isLoading}
            error={membersQ.isError}
            empty={t("tenants.membersEmpty")}
          />
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

// ── Add-member drawer (controlled form → addTenantMember) ──────────────────────
interface MemberFormValues {
  account: string;
  roleName: "tenant-admin" | "user";
}

function MemberDrawer({ ident, onClose }: { ident: string; onClose: () => void }) {
  const { t } = useTranslation();
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<MemberFormValues>({ account: "", roleName: "user" });
  const set = <K extends keyof MemberFormValues>(k: K, val: MemberFormValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const add = useApiMutation(
    (body: sdk.MemberCreateRequest) => sdk.addTenantMember({ path: { name: ident }, body }),
    { invalidate: [["tenant-members", ident]], success: t("tenants.memberAdded") },
  );

  const submit = () => {
    setSubmitted(true);
    const account = v.account.trim();
    if (!account) return;
    add.mutate({ account, roleName: v.roleName }, { onSuccess: onClose });
  };

  return (
    <Sheet open onOpenChange={(o) => !o && onClose()}>
      <SheetContent side="right" className="flex w-full flex-col gap-0 p-0 sm:max-w-[480px]">
        <SheetHeader className="border-b">
          <SheetTitle>{t("tenants.memberDrawerTitle")}</SheetTitle>
          <p className="text-xs text-muted-foreground">
            <span className="font-mono">{ident}</span>
          </p>
        </SheetHeader>

        <div className="flex-1 overflow-auto px-6 py-4">
          <FieldSection n={1} title={t("tenants.memberDrawerSub", { name: ident })} />
          <Field className="mb-4">
            <FieldLabel htmlFor="member-account">
              {t("tenants.fAccount")}
              <span className="text-destructive">*</span>
            </FieldLabel>
            <Input
              id="member-account"
              placeholder={t("tenants.fAccountPlaceholder")}
              value={v.account}
              aria-invalid={submitted && !v.account.trim()}
              onChange={(e) => set("account", e.target.value)}
            />
            <FieldDescription>{t("tenants.fAccountHelp")}</FieldDescription>
          </Field>
          <Field className="mb-4">
            <FieldLabel>{t("tenants.fRole")}</FieldLabel>
            <Select value={v.roleName} onValueChange={(val) => set("roleName", val as MemberFormValues["roleName"])}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MEMBER_ROLES.map((r) => (
                  <SelectItem key={r} value={r}>
                    {t(`role.${r}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
        </div>

        <SheetFooter className="flex-row justify-end border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={submit} disabled={add.isPending}>
            {add.isPending && <Spinner data-icon="inline-start" />}
            {t("tenants.addMember")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
