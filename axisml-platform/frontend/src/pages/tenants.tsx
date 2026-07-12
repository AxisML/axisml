import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { usePagedList, useResourcePools } from "@/api/hooks";
import { useDebouncedValue } from "@/lib/use-debounced";
import { LoadMore } from "@/components/load-more";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { errorText } from "@/lib/errors";
import { cn } from "@/lib/utils";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { PhaseTag } from "@/components/phase-tag";
import { SearchInput } from "@/components/search-input";
import { DataTable, type Column } from "@/components/data-table";
import { USE_MOCK } from "@/api/mock";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { FormDrawer } from "@/components/form-drawer";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import { Spinner } from "@/components/ui/spinner";
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";

const MEMBER_ROLES: ("user" | "tenant-admin")[] = ["user", "tenant-admin"];

interface TenantRow {
  ident: string;
  display: string;
  active: boolean;
  phase?: string;
  pools: { pool: string; allocated: number; used?: number }[];
  members: number;
  activeTasks: number;
  services: number;
  created: string;
}

// pool name → (unit name → quantity). The shared shape edited by QuotaPoolEditor
// and consumed by both the create-tenant and edit-quota drawers.
type QuotaMap = Record<string, Record<string, number>>;

// Human-readable spec line for a resource unit, derived from its requests/limits
// (e.g. "8 GPU · 96 vCPU · 768 GiB"). Mirrors the prototype's unit-card spec text.
function unitSpec(u: sdk.ResourceUnit): string {
  const m = u.requests ?? u.limits ?? {};
  const parts: string[] = [];
  if (m["nvidia.com/gpu"]) parts.push(`${m["nvidia.com/gpu"]} GPU`);
  if (m["cpu"]) parts.push(`${m["cpu"]} vCPU`);
  if (m["memory"])
    parts.push(m["memory"].replace(/Ti$/, " TiB").replace(/Gi$/, " GiB").replace(/Mi$/, " MiB"));
  return parts.join(" · ");
}

// Demo-only quota utilisation ratio (deterministic per pool name). Real quota
// usage comes from the tenant status when present; otherwise the meter only
// renders under mock, and live deployments fall back to honest allocated text.
function mockUsedRatio(pool: string): number {
  const h = [...pool].reduce((a, c) => a + c.charCodeAt(0), 0);
  return 0.5 + ((h % 45) / 100); // 0.50 – 0.94
}

// Per-pool quota row: pool tag + used/allocated number + water-level bar — mirrors
// the prototype's `.q-meters` (tag · number · bar). When no usage is known the
// bar is dropped and only the honest allocated quantity is shown.
function QuotaMeter({ pool, allocated, used }: { pool: string; allocated: number; used?: number }) {
  if (used == null) {
    return (
      <div className="grid grid-cols-[92px_1fr] items-center gap-2.5">
        <Badge variant="outline" className="justify-start font-mono">
          {pool}
        </Badge>
        <span className="font-mono text-xs text-muted-foreground">{allocated} 单元</span>
      </div>
    );
  }
  const pct = allocated === 0 ? 0 : Math.min(100, Math.round((used / allocated) * 100));
  const fill = pct >= 80 ? "bg-destructive" : pct >= 60 ? "bg-warning" : "bg-success";
  return (
    <div className="grid grid-cols-[92px_52px_1fr] items-center gap-2.5">
      <Badge variant="outline" className="justify-start font-mono">
        {pool}
      </Badge>
      <span className="text-right font-mono text-xs text-muted-foreground">
        {used} / {allocated}
      </span>
      <div className="h-[9px] overflow-hidden rounded-full bg-muted">
        <div className={`h-full rounded-full ${fill}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

type DrawerKind =
  | { kind: "tenant" }
  | { kind: "edit"; ident: string; display: string; description: string }
  | { kind: "quota"; ident: string }
  | { kind: "members"; ident: string }
  | { kind: "member"; ident: string };

export default function Tenants() {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<DrawerKind | null>(null);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<"all" | "active" | "suspended">("all");

  // Server-side search + pagination (with the workload-stats roll-ups). The
  // active/suspended facet maps to suspend state (not a list param), so it stays
  // a client refinement over loaded rows.
  const dq = useDebouncedValue(search, 300);
  const q = usePagedList<sdk.Tenant>(
    ["tenants", "stats", dq],
    (page) => sdk.listTenants({ query: { stats: true, q: dq || undefined, ...page } }),
    { scoped: false },
  );

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

  const onEditTenant = (ident: string) => {
    const tenant = q.items.find((x) => x.identifier === ident);
    setDrawer({
      kind: "edit",
      ident,
      display: tenant?.displayName ?? "",
      description: tenant?.description ?? "",
    });
  };

  const allRows: TenantRow[] = useMemo(
    () =>
      q.items.map((tenant) => ({
        ident: tenant.identifier,
        display: tenant.displayName,
        active: !tenant.suspended,
        phase: tenant.phase,
        pools: (tenant.quotas ?? []).map((quota) => {
          const allocated = (quota.units ?? []).reduce((sum, u) => sum + (u.quantity ?? 0), 0);
          const statusUnits = tenant.status?.quotas?.find((s) => s.pool === quota.pool)?.units ?? [];
          const hasReal = statusUnits.some((u) => u.used != null);
          return {
            pool: quota.pool,
            allocated,
            used: hasReal
              ? statusUnits.reduce((sum, u) => sum + (u.used ?? 0), 0)
              : USE_MOCK
                ? Math.round(allocated * mockUsedRatio(quota.pool))
                : undefined,
          };
        }),
        members: tenant.memberCount ?? 0,
        activeTasks: (tenant.activeJobRuns ?? 0) + (tenant.activeExperimentRuns ?? 0),
        services: tenant.onlineServices ?? 0,
        created: tenant.createdAt,
      })),
    [q.items],
  );

  // Only the suspend-state facet is client-side; search is applied server-side.
  const rows = useMemo(
    () => allRows.filter((r) => status === "all" || (status === "active" ? r.active : !r.active)),
    [allRows, status],
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
            onClick={() => setDrawer({ kind: "quota", ident: r.ident })}
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
      // Show the real tenant phase (Creating / Active / Suspended / Failed);
      // fall back to the suspend flag only when the phase isn't populated.
      render: (r) => <PhaseTag phase={r.phase ?? (r.active ? "Active" : "Suspended")} />,
    },
    {
      key: "quota",
      title: t("tenants.colQuota"),
      width: 300,
      render: (r) =>
        r.pools.length === 0 ? (
          <span className="text-muted-foreground">{t("tenants.noQuota")}</span>
        ) : (
          <div className="flex flex-col gap-2">
            {r.pools.map((p) => (
              <QuotaMeter key={p.pool} pool={p.pool} allocated={p.allocated} used={p.used} />
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
      width: 300,
      align: "right",
      render: (r) => (
        <div className="flex items-center justify-end gap-0.5">
          <Button variant="link" size="sm" onClick={() => onEditTenant(r.ident)}>
            {t("tenants.edit")}
          </Button>
          <Button
            variant="link"
            size="sm"
            onClick={() => setDrawer({ kind: "quota", ident: r.ident })}
          >
            {t("tenants.editQuota")}
          </Button>
          <Button
            variant="link"
            size="sm"
            onClick={() => setDrawer({ kind: "members", ident: r.ident })}
          >
            {t("tenants.manageMembers")}
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
          <SearchInput
            className="max-w-xs flex-1"
            placeholder={t("tenants.searchPlaceholder")}
            value={search}
            onChange={setSearch}
          />
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
      <LoadMore hasMore={q.hasMore} loading={q.isFetchingMore} onClick={q.loadMore} />

      {drawer?.kind === "tenant" && <TenantDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "edit" && (
        <TenantEditDrawer
          ident={drawer.ident}
          display={drawer.display}
          description={drawer.description}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === "quota" && (
        <QuotaDrawer ident={drawer.ident} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "members" && (
        <MembersDrawer
          ident={drawer.ident}
          onAddMember={() => setDrawer({ kind: "member", ident: drawer.ident })}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === "member" && <MemberDrawer ident={drawer.ident} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Quota editor: per-pool tabs, each listing the pool's resource units as cards
// with an inline quantity stepper (zero rows dimmed). Mirrors the prototype's
// `.pool-tabs` / `.qp-units`. Shared by the create-tenant and edit-quota flows. ──
function QuotaPoolEditor({
  pools,
  value,
  onChange,
}: {
  pools: sdk.ResourcePool[];
  value: QuotaMap;
  onChange: (next: QuotaMap) => void;
}) {
  const { t } = useTranslation();
  const [tab, setTab] = useState(pools[0]?.name ?? "");

  if (pools.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{t("tenants.noPools")}</EmptyTitle>
        </EmptyHeader>
      </Empty>
    );
  }

  const setQty = (pool: string, unit: string, qty: number) =>
    onChange({ ...value, [pool]: { ...value[pool], [unit]: Math.max(0, qty) } });

  return (
    <Tabs value={tab || pools[0].name} onValueChange={setTab}>
      <TabsList>
        {pools.map((p) => (
          <TabsTrigger key={p.name} value={p.name} className="font-mono">
            {p.name}
          </TabsTrigger>
        ))}
      </TabsList>
      {pools.map((p) => {
        const units = p.units ?? [];
        return (
          <TabsContent key={p.name} value={p.name} className="pt-4">
            {p.description && <p className="mb-3 text-xs text-muted-foreground">{p.description}</p>}
            {units.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyTitle>{t("tenants.noUnits")}</EmptyTitle>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className="flex flex-col gap-3">
                {units.map((u) => {
                  const qty = value[p.name]?.[u.name] ?? 0;
                  const zero = qty === 0;
                  return (
                    <div key={u.name} className="flex items-center gap-4">
                      <Card
                        className={cn(
                          "flex-1 gap-0 p-3 transition-colors",
                          zero ? "border-border/60 bg-muted/40" : "bg-card",
                        )}
                      >
                        <div
                          className={cn(
                            "font-mono text-sm font-medium",
                            zero ? "text-muted-foreground" : "text-foreground",
                          )}
                        >
                          {u.name}
                        </div>
                        <div className="mt-0.5 text-xs text-muted-foreground">
                          {u.description || unitSpec(u) || "—"}
                        </div>
                      </Card>
                      <label className="flex shrink-0 items-center gap-2">
                        <span className="text-muted-foreground">×</span>
                        <Input
                          type="number"
                          min={0}
                          aria-label={t("tenants.qQuantity")}
                          className="w-14 text-center font-mono"
                          value={qty}
                          onChange={(e) => setQty(p.name, u.name, Number(e.target.value) || 0)}
                        />
                      </label>
                    </div>
                  );
                })}
              </div>
            )}
          </TabsContent>
        );
      })}
    </Tabs>
  );
}

// Collapse a QuotaMap into the API's quota array, dropping zero-quantity units
// and pools that end up empty.
function quotasFromMap(map: QuotaMap): sdk.Quota[] {
  return Object.entries(map)
    .map(([pool, units]) => ({
      pool,
      units: Object.entries(units)
        .filter(([, qty]) => qty > 0)
        .map(([unitName, quantity]) => ({ unitName, quantity })),
    }))
    .filter((quota) => quota.units.length > 0);
}

// ── Edit-tenant drawer (display metadata → updateTenant) ────────────────────────
// Only display metadata is editable; the identifier (= namespace) is immutable.
function TenantEditDrawer({
  ident,
  display,
  description,
  onClose,
}: {
  ident: string;
  display: string;
  description: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [displayName, setDisplayName] = useState(display);
  const [desc, setDesc] = useState(description);

  const update = useApiMutation(
    (body: sdk.TenantPatchRequest) => sdk.updateTenant({ path: { name: ident }, body }),
    { invalidate: [["tenants"]], success: t("tenants.edited") },
  );

  const submit = () =>
    update.mutate(
      { displayName: displayName.trim() || undefined, description: desc.trim() || undefined },
      { onSuccess: onClose },
    );

  return (
    <FormDrawer
      title={t("tenants.drawerEdit")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("common.save")}
      submitting={update.isPending}
    >
      <FieldSection n={1} title={t("tenants.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="tenant-edit-ident">{t("tenants.fIdentifier")}</FieldLabel>
          <Input id="tenant-edit-ident" className="font-mono" value={ident} readOnly disabled />
        </Field>
        <Field>
          <FieldLabel htmlFor="tenant-edit-display">{t("tenants.fDisplayName")}</FieldLabel>
          <Input
            id="tenant-edit-display"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="tenant-edit-desc">{t("tenants.fDesc")}</FieldLabel>
          <Textarea
            id="tenant-edit-desc"
            rows={2}
            value={desc}
            onChange={(e) => setDesc(e.target.value)}
          />
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}

// ── Create-tenant drawer (basic info + initial per-pool quota → createTenant) ───
interface TenantFormValues {
  displayName: string;
  identifier: string;
  initialAdmin: string;
}

function TenantDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const poolsQ = useResourcePools();
  const pools = poolsQ.data?.items ?? [];
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<TenantFormValues>({
    displayName: "",
    identifier: "",
    initialAdmin: "",
  });
  const [quota, setQuota] = useState<QuotaMap>({});
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
    const quotas = quotasFromMap(quota);
    create.mutate(
      {
        displayName: display,
        identifier: ident,
        initialAdmin: admin,
        // The identifier is a dns1123 slug; reuse it as the physical namespace
        // (tenant name = namespace convention).
        kubernetesNamespace: ident,
        quotas: quotas.length ? quotas : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <FormDrawer
      title={t("tenants.drawerNew")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("tenants.createTenant")}
      submitting={create.isPending}
    >
      <FieldSection n={1} title={t("tenants.fsBasic")} />
      <FieldGroup>
        <Field>
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
        <Field>
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
        <Field>
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
      </FieldGroup>

      <FieldSection n={2} title={t("tenants.fsQuota")} />
      <p className="mb-3 text-xs text-muted-foreground">{t("tenants.quotaHint")}</p>
      {poolsQ.isLoading ? (
        <div className="flex justify-center py-8">
          <Spinner className="size-7 text-muted-foreground" />
        </div>
      ) : (
        <QuotaPoolEditor pools={pools} value={quota} onChange={setQuota} />
      )}
    </FormDrawer>
  );
}

// ── Quota editor drawer (live quotas hydrate the per-pool editor) ──────────────
function QuotaDrawer({ ident, onClose }: { ident: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { toast } = useUI();
  const poolsQ = useResourcePools();
  const pools = poolsQ.data?.items ?? [];

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
    { invalidate: [["tenant-quotas", ident], ["tenants"]] },
  );
  const createQuota = useApiMutation(
    (arg: { pool: string; units: sdk.QuotaUnit[] }) =>
      sdk.createTenantQuota({ path: { name: ident }, body: { pool: arg.pool, units: arg.units } }),
    { invalidate: [["tenant-quotas", ident], ["tenants"]] },
  );
  const delQuota = useApiMutation((pool: string) => sdk.deleteTenantQuota({ path: { name: ident, pool } }), {
    invalidate: [["tenant-quotas", ident], ["tenants"]],
  });

  const quotas = quotasQ.data?.items ?? [];
  const existingPools = useMemo(() => new Set(quotas.map((quota) => quota.pool)), [quotas]);

  // Editable map hydrates once from the live quotas.
  const initial: QuotaMap = useMemo(() => {
    const m: QuotaMap = {};
    for (const quota of quotas) {
      m[quota.pool] = {};
      for (const u of quota.units ?? []) m[quota.pool][u.unitName] = u.quantity;
    }
    return m;
  }, [quotas]);
  const [value, setValue] = useState<QuotaMap | null>(null);
  const editValue = value ?? initial;

  const onFinish = async () => {
    const desired = new Map<string, sdk.QuotaUnit[]>();
    for (const quota of quotasFromMap(editValue)) desired.set(quota.pool, quota.units ?? []);

    const ops: Promise<unknown>[] = [];
    for (const [pool, units] of desired) {
      ops.push(
        existingPools.has(pool)
          ? updateQuota.mutateAsync({ pool, units })
          : createQuota.mutateAsync({ pool, units }),
      );
    }
    // Pools that previously had a quota but were zeroed out are removed.
    for (const pool of existingPools) {
      if (!desired.has(pool)) ops.push(delQuota.mutateAsync(pool));
    }

    if (ops.length === 0) {
      onClose();
      return;
    }
    try {
      await Promise.all(ops);
      toast(t("tenants.quotaSaved"));
      onClose();
    } catch {
      // Per-mutation errors already surfaced as toasts by useApiMutation.
    }
  };

  const loading = quotasQ.isLoading || poolsQ.isLoading;
  const saving = updateQuota.isPending || createQuota.isPending || delQuota.isPending;

  return (
    <FormDrawer
      title={t("tenants.editQuota")}
      subtitle={<span className="font-mono">{ident}</span>}
      onClose={onClose}
      onSubmit={onFinish}
      submitLabel={t("tenants.saveQuota")}
      submitting={saving}
    >
      {loading ? (
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
        <QuotaPoolEditor pools={pools} value={editValue} onChange={setValue} />
      )}
    </FormDrawer>
  );
}

// ── Members drawer (live members → remove / update role) ───────────────────────
function MembersDrawer({
  ident,
  onAddMember,
  onClose,
}: {
  ident: string;
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
    <FormDrawer
      title={t("tenants.manageMembers")}
      subtitle={<span className="font-mono">{ident}</span>}
      onClose={onClose}
      footer={
        <Button variant="outline" onClick={onClose}>
          {t("common.close")}
        </Button>
      }
    >
      <div className="mb-3 flex items-center gap-3">
        <SearchInput
          className="max-w-xs flex-1"
          placeholder={t("tenants.memberSearchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
        <Button size="sm" onClick={onAddMember}>
          <Plus data-icon="inline-start" />
          {t("tenants.addMember")}
        </Button>
      </div>
      <DataTable
        columns={columns}
        data={members}
        rowKey={(m) => m.userId}
        loading={membersQ.isLoading}
        error={membersQ.isError}
        empty={t("tenants.membersEmpty")}
      />
    </FormDrawer>
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

  // Typeahead against the platform user directory. Suggestions only (free text is
  // still accepted); a 403/501 for non-system-admins just yields no suggestions.
  const usersQ = useQuery({
    queryKey: ["users", v.account],
    retry: false,
    queryFn: async () => {
      const { data, error } = await sdk.listUsers({ query: v.account ? { q: v.account } : {} });
      if (error) throw error;
      return data;
    },
  });
  const userOptions = usersQ.data?.items ?? [];

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
    <FormDrawer
      title={t("tenants.memberDrawerTitle")}
      size="compact"
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("tenants.addMember")}
      submitting={add.isPending}
    >
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="member-account">
            {t("tenants.fAccount")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="member-account"
            list="member-user-options"
            placeholder={t("tenants.fAccountPlaceholder")}
            value={v.account}
            aria-invalid={submitted && !v.account.trim()}
            onChange={(e) => set("account", e.target.value)}
          />
          <datalist id="member-user-options">
            {userOptions.map((u) => (
              <option key={u.id} value={u.username}>
                {u.displayName || u.email || u.username}
              </option>
            ))}
          </datalist>
          <FieldDescription>{t("tenants.fAccountHelp")}</FieldDescription>
        </Field>
        <Field>
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
      </FieldGroup>
    </FormDrawer>
  );
}
