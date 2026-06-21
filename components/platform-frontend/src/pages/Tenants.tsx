import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Table,
  Button,
  Input,
  Select,
  Space,
  Card,
  Tooltip,
  Divider,
  Drawer,
  Form,
  InputNumber,
  Empty,
  Spin,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  UserAddOutlined,
  StopOutlined,
  DeleteOutlined,
  CloseOutlined,
} from "@ant-design/icons";
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
import { USE_MOCK } from "@/api/mock";

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
        <span className="w-[88px] shrink-0 truncate font-mono text-xs text-fg-2">{pool}</span>
        <span className="font-mono text-xs text-muted">{allocated} 单元</span>
      </div>
    );
  }
  const pct = allocated === 0 ? 0 : Math.min(100, Math.round((used / allocated) * 100));
  const fill = pct >= 80 ? "bg-danger" : pct >= 60 ? "bg-warn" : "bg-success";
  return (
    <div className="flex items-center gap-2">
      <span className="w-[88px] shrink-0 truncate font-mono text-xs text-fg-2">{pool}</span>
      <div className="h-[7px] flex-1 overflow-hidden rounded-full bg-surface">
        <div className={`h-full rounded-full ${fill}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="w-11 shrink-0 text-right font-mono text-xs text-muted">{used}/{allocated}</span>
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

  const columns: ColumnsType<TenantRow> = [
    {
      title: t("tenants.colTenant"),
      dataIndex: "ident",
      render: (_, r) => (
        <div className="min-w-0">
          <button
            type="button"
            className="font-mono font-medium text-accent hover:underline"
            onClick={() => setDrawer({ kind: "quota", ident: r.ident, display: r.display })}
          >
            {r.ident}
          </button>
          <div className="truncate text-xs text-muted">{r.display}</div>
        </div>
      ),
    },
    {
      title: t("tenants.colStatus"),
      key: "status",
      width: 110,
      render: (_, r) => <PhaseTag phase={r.active ? "Active" : "Suspended"} />,
    },
    {
      title: t("tenants.colQuota"),
      key: "quota",
      width: 240,
      render: (_, r) =>
        r.pools.length === 0 ? (
          <span className="text-muted">{t("tenants.noQuota")}</span>
        ) : (
          <div className="flex flex-col gap-1.5">
            {r.pools.map((p) => (
              <QuotaBar key={p.pool} pool={p.pool} allocated={p.allocated} used={p.used} />
            ))}
          </div>
        ),
    },
    { title: t("tenants.colMembers"), dataIndex: "members", width: 80, align: "right" },
    { title: t("tenants.colActiveTasks"), dataIndex: "activeTasks", width: 90, align: "right" },
    { title: t("tenants.colServices"), dataIndex: "services", width: 90, align: "right" },
    {
      title: t("tenants.colCreated"),
      dataIndex: "created",
      width: 160,
      render: (v: string) => (
        <span className="text-muted">{v ? dayjs(v).format("YYYY-MM-DD") : "—"}</span>
      ),
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 180,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "quota", ident: r.ident, display: r.display })}
          >
            {t("tenants.editQuota")}
          </Button>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "members", ident: r.ident, display: r.display })}
          >
            {t("tenants.manageMembers")}
          </Button>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "member", ident: r.ident })}
          >
            {t("tenants.addMember")}
          </Button>
          {r.active ? (
            <Button type="link" size="small" className="!px-1" onClick={() => onSuspend(r)}>
              {t("tenants.suspend")}
            </Button>
          ) : (
            <Button type="link" size="small" className="!px-1" onClick={() => resume.mutate(r.ident)}>
              {t("tenants.resume")}
            </Button>
          )}
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.systemMgmt"), t("nav.tenants")]}
      title={t("tenants.title")}
      subtitle={t("tenants.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ kind: "tenant" })}>
          {t("tenants.newTenant")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("tenants.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={status}
            onChange={setStatus}
            className="min-w-40"
            options={[
              { label: t("tenants.statusAll"), value: "all" },
              { label: t("tenants.statusActive"), value: "active" },
              { label: t("tenants.statusSuspended"), value: "suspended" },
            ]}
          />
          <Button
            onClick={() => {
              setSearch("");
              setStatus("all");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <Table<TenantRow>
          rowKey="ident"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{
            pageSize: 20,
            showTotal: (n) => t("tenants.total", { count: n }),
            hideOnSinglePage: false,
          }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
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
  const [form] = Form.useForm<TenantFormValues>();

  const create = useApiMutation((body: sdk.TenantCreateRequest) => sdk.createTenant({ body }), {
    invalidate: [["tenants"]],
    success: t("tenants.created"),
  });

  const onFinish = (v: TenantFormValues) => {
    const ident = v.identifier.trim();
    create.mutate(
      {
        displayName: v.displayName.trim(),
        identifier: ident,
        initialAdmin: v.initialAdmin.trim(),
        // The identifier is a dns1123 slug; reuse it as the physical namespace
        // (tenant name = namespace convention).
        kubernetesNamespace: ident,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <Drawer
      open
      width={560}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("tenants.drawerNew")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">{t("tenants.drawerNewSub")}</div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={create.isPending} onClick={() => form.submit()}>
            {t("tenants.createTenant")}
          </Button>
        </div>
      }
    >
      <Form<TenantFormValues> form={form} layout="vertical" size="large" onFinish={onFinish}>
        <FieldSection n={1} title={t("tenants.fsBasic")} />
        <Form.Item
          name="displayName"
          label={t("tenants.fDisplayName")}
          rules={[{ required: true }]}
          extra={t("tenants.fDisplayNameHelp")}
        >
          <Input placeholder={t("tenants.fDisplayNamePlaceholder")} />
        </Form.Item>
        <Form.Item
          name="identifier"
          label={t("tenants.fIdentifier")}
          rules={[{ required: true }]}
          extra={t("tenants.fIdentifierHelp")}
        >
          <Input className="font-mono" placeholder={t("tenants.fIdentifierPlaceholder")} />
        </Form.Item>
        <Form.Item name="initialAdmin" label={t("tenants.fAdmin")} rules={[{ required: true }]}>
          <Input placeholder={t("tenants.fAdminPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("tenants.fsQuota")} />
        <p className="text-xs text-muted">{t("tenants.quotaHint")}</p>
      </Form>
    </Drawer>
  );
}

// ── Quota editor drawer (live quotas → Form.List rows grouped by pool) ─────────
interface QuotaRow {
  pool?: string;
  unitName?: string;
  quantity?: number;
}

function QuotaDrawer({ ident, display, onClose }: { ident: string; display: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [form] = Form.useForm<{ rows: QuotaRow[] }>();

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

  const onFinish = (v: { rows: QuotaRow[] }) => {
    // Group rows by pool, then upsert each pool (PATCH if it already exists,
    // POST otherwise). Backend owns the (pool/unit/quantity) vocabulary.
    const byPool = new Map<string, sdk.QuotaUnit[]>();
    for (const r of v.rows ?? []) {
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
    <Drawer
      open
      width={680}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("tenants.quotaDrawerTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{ident}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={saving} onClick={() => form.submit()}>
            {t("tenants.saveQuota")}
          </Button>
        </div>
      }
    >
      <p className="mb-4 text-xs text-muted">{t("tenants.quotaDrawerSub", { name: display })}</p>
      {quotasQ.isLoading ? (
        <div className="flex justify-center py-8">
          <Spin />
        </div>
      ) : quotasQ.isError ? (
        <Empty description={t("common.loadFailed")} />
      ) : (
        <>
          {delQuota.isPending && (
            <div className="mb-3 flex justify-center">
              <Spin size="small" />
            </div>
          )}
          <Form<{ rows: QuotaRow[] }>
            form={form}
            layout="vertical"
            size="large"
            onFinish={onFinish}
            initialValues={{ rows: initialRows }}
          >
            <FieldSection n={1} title={t("tenants.fsQuotaRows")} />
            <p className="mb-3 text-xs text-muted">{t("tenants.quotaRowsSub")}</p>
            <Form.List name="rows">
              {(fields, { add, remove }) => (
                <div className="space-y-3">
                  {fields.length === 0 && <Empty description={t("tenants.quotaEmpty")} />}
                  {fields.map((field) => {
                    const pool = form.getFieldValue(["rows", field.name, "pool"]) as string | undefined;
                    const canRemovePool = !!pool && existingPools.has(pool);
                    return (
                      <Card key={field.key} size="small" className="bg-surface-warm">
                        <div className="grid grid-cols-[1fr_1fr_auto_auto] items-end gap-x-3">
                          <Form.Item
                            name={[field.name, "pool"]}
                            label={t("tenants.qPool")}
                            rules={[{ required: true }]}
                          >
                            <Select
                              placeholder={t("tenants.qPoolPlaceholder")}
                              options={POOL_OPTIONS.map((p) => ({ label: p, value: p }))}
                            />
                          </Form.Item>
                          <Form.Item
                            name={[field.name, "unitName"]}
                            label={t("tenants.qUnit")}
                            rules={[{ required: true }]}
                          >
                            <Select
                              placeholder={t("tenants.qUnitPlaceholder")}
                              options={UNIT_OPTIONS.map((u) => ({ label: u, value: u }))}
                            />
                          </Form.Item>
                          <Form.Item name={[field.name, "quantity"]} label={t("tenants.qQuantity")}>
                            <InputNumber min={0} className="!w-24" />
                          </Form.Item>
                          <Form.Item label=" ">
                            <Space size="small">
                              {canRemovePool && (
                                <Tooltip title={t("tenants.removePool")}>
                                  <Button
                                    type="text"
                                    size="small"
                                    icon={<StopOutlined />}
                                    onClick={() => onRemovePool(pool!)}
                                  />
                                </Tooltip>
                              )}
                              <Button
                                type="text"
                                size="small"
                                danger
                                icon={<DeleteOutlined />}
                                onClick={() => remove(field.name)}
                              />
                            </Space>
                          </Form.Item>
                        </div>
                      </Card>
                    );
                  })}
                  <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ quantity: 0 })}>
                    {t("tenants.addQuotaRow")}
                  </Button>
                </div>
              )}
            </Form.List>
          </Form>
        </>
      )}
    </Drawer>
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

  const columns: ColumnsType<sdk.Member> = [
    {
      title: t("tenants.mColMember"),
      key: "member",
      render: (_, m) => {
        const name = m.displayName || m.username || m.email || m.userId;
        const initial = (name || "?").trim().charAt(0).toUpperCase();
        return (
          <div className="flex items-center gap-2.5">
            <span className="grid h-7 w-7 place-items-center rounded-full bg-accent text-xs font-semibold text-accent-on">
              {initial}
            </span>
            <span className="text-sm text-fg">{name}</span>
          </div>
        );
      },
    },
    {
      title: t("tenants.mColRole"),
      key: "role",
      width: 160,
      render: (_, m) => (
        <Select
          size="small"
          value={m.roleName === "tenant-admin" ? "tenant-admin" : "user"}
          disabled={updateRole.isPending}
          className="w-full"
          onChange={(v) => updateRole.mutate({ userId: m.userId, roleName: v as "tenant-admin" | "user" })}
          options={MEMBER_ROLES.map((r) => ({ label: roleLabel(r), value: r }))}
        />
      ),
    },
    {
      title: t("tenants.mColJoined"),
      dataIndex: "addedAt",
      width: 120,
      render: (v: string) => <span className="text-muted">{v ? dayjs(v).format("YYYY-MM-DD") : "—"}</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 80,
      align: "right",
      render: (_, m) => {
        const name = m.displayName || m.username || m.email || m.userId;
        return (
          <Button
            type="link"
            size="small"
            danger
            className="!px-1"
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
    <Drawer
      open
      width={680}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("tenants.membersDrawerTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{ident}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
        </div>
      }
    >
      <div className="mb-4 flex items-center justify-between gap-3">
        <p className="m-0 text-xs text-muted">{t("tenants.membersDrawerSub", { name: display })}</p>
        <Button type="primary" size="small" icon={<UserAddOutlined />} onClick={onAddMember}>
          {t("tenants.addMember")}
        </Button>
      </div>
      <Input
        allowClear
        prefix={<SearchOutlined className="text-muted" />}
        placeholder={t("tenants.memberSearchPlaceholder")}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="mb-3 max-w-xs"
      />
      <Table<sdk.Member>
        rowKey="userId"
        size="small"
        columns={columns}
        dataSource={members}
        loading={membersQ.isLoading}
        pagination={false}
        locale={{ emptyText: membersQ.isError ? t("common.loadFailed") : t("tenants.membersEmpty") }}
      />
    </Drawer>
  );
}

// ── Add-member drawer (controlled form → addTenantMember) ──────────────────────
interface MemberFormValues {
  account: string;
  roleName: "tenant-admin" | "user";
}

function MemberDrawer({ ident, onClose }: { ident: string; onClose: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm<MemberFormValues>();

  const add = useApiMutation(
    (body: sdk.MemberCreateRequest) => sdk.addTenantMember({ path: { name: ident }, body }),
    { invalidate: [["tenant-members", ident]], success: t("tenants.memberAdded") },
  );

  const onFinish = (v: MemberFormValues) =>
    add.mutate({ account: v.account.trim(), roleName: v.roleName }, { onSuccess: onClose });

  return (
    <Drawer
      open
      width={480}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("tenants.memberDrawerTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{ident}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={add.isPending} onClick={() => form.submit()}>
            {t("tenants.addMember")}
          </Button>
        </div>
      }
    >
      <Form<MemberFormValues>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{ roleName: "user" }}
      >
        <FieldSection n={1} title={t("tenants.memberDrawerSub", { name: ident })} />
        <Form.Item
          name="account"
          label={t("tenants.fAccount")}
          rules={[{ required: true }]}
          extra={t("tenants.fAccountHelp")}
        >
          <Input placeholder={t("tenants.fAccountPlaceholder")} />
        </Form.Item>
        <Form.Item name="roleName" label={t("tenants.fRole")} rules={[{ required: true }]}>
          <Select options={MEMBER_ROLES.map((r) => ({ label: t(`role.${r}`), value: r }))} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
