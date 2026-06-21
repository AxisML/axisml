import { useMemo, useState } from "react";
import {
  Table,
  Button,
  Input,
  Space,
  Card,
  Divider,
  Drawer,
  Form,
  InputNumber,
  Tabs,
  Descriptions,
  Tag,
  Empty,
  Spin,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  SearchOutlined,
  DeleteOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useResourcePools } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";

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

  const columns: ColumnsType<sdk.ResourcePool> = [
    {
      title: t("pools.colName"),
      dataIndex: "name",
      render: (_, p) => (
        <button
          type="button"
          className="font-mono font-medium text-accent hover:underline"
          onClick={() => setDrawer({ kind: "detail", pool: p })}
        >
          {p.name}
        </button>
      ),
    },
    {
      title: t("pools.colDesc"),
      dataIndex: "description",
      render: (v: string) => v || <span className="text-muted">—</span>,
    },
    {
      title: t("pools.colSelector"),
      key: "selector",
      render: (_, p) => {
        const pairs = selectorPairs(p.nodeSelector);
        if (!pairs.length) return <span className="text-muted">{t("pools.noSelector")}</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {pairs.map((s) => (
              <Tag key={s} className="!m-0 font-mono">
                {s}
              </Tag>
            ))}
          </div>
        );
      },
    },
    {
      title: t("pools.colUnits"),
      key: "units",
      width: 100,
      align: "right",
      render: (_, p) => (
        <button
          type="button"
          className="text-accent hover:underline"
          onClick={() => setDrawer({ kind: "units", pool: p })}
        >
          {p.units?.length ?? 0}
        </button>
      ),
    },
    {
      title: t("pools.colCreated"),
      dataIndex: "createdAt",
      width: 180,
      render: (v: string) => (
        <span className="text-muted">{v ? dayjs(v).format("YYYY-MM-DD HH:mm") : "—"}</span>
      ),
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 180,
      align: "right",
      render: (_, p) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "detail", pool: p })}
          >
            {t("common.detail")}
          </Button>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "edit", pool: p })}
          >
            {t("common.edit")}
          </Button>
          <Button
            type="link"
            size="small"
            className="!px-1"
            onClick={() => setDrawer({ kind: "units", pool: p })}
          >
            {t("pools.manageUnits")}
          </Button>
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDelete(p)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.systemMgmt"), t("nav.pools")]}
      title={t("pools.title")}
      subtitle={t("pools.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ kind: "new" })}>
          {t("pools.newPool")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("pools.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Button onClick={() => setSearch("")}>{t("common.reset")}</Button>
        </div>
        <Table<sdk.ResourcePool>
          rowKey="name"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{
            pageSize: 20,
            showTotal: (n) => t("pools.total", { count: n }),
            hideOnSinglePage: false,
          }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
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

// ── Detail drawer (Tabs + Descriptions, read-only) ────────────────────────────
function PoolDetailDrawer({ pool, onClose }: { pool: sdk.ResourcePool; onClose: () => void }) {
  const { t } = useTranslation();
  const pairs = selectorPairs(pool.nodeSelector);
  const units = pool.units ?? [];

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("pools.detailTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{pool.name}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
    >
      <Tabs
        items={[
          {
            key: "basic",
            label: t("pools.tabBasic"),
            children: (
              <Descriptions column={1} bordered size="small">
                <Descriptions.Item label={t("pools.dName")}>
                  <span className="font-mono">{pool.name}</span>
                </Descriptions.Item>
                <Descriptions.Item label={t("pools.dDesc")}>
                  {pool.description || <span className="text-muted">—</span>}
                </Descriptions.Item>
                <Descriptions.Item label={t("pools.dSelector")}>
                  {pairs.length ? (
                    <div className="flex flex-wrap gap-1">
                      {pairs.map((s) => (
                        <Tag key={s} className="!m-0 font-mono">
                          {s}
                        </Tag>
                      ))}
                    </div>
                  ) : (
                    <span className="text-muted">{t("pools.noSelector")}</span>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label={t("pools.dNodeCount")}>
                  {pool.nodeCount ?? "—"}
                </Descriptions.Item>
                <Descriptions.Item label={t("pools.dUnitCount")}>{units.length}</Descriptions.Item>
                <Descriptions.Item label={t("pools.dCreated")}>
                  {pool.createdAt ? dayjs(pool.createdAt).format("YYYY-MM-DD HH:mm") : "—"}
                </Descriptions.Item>
                <Descriptions.Item label={t("pools.dUpdated")}>
                  {pool.updatedAt ? dayjs(pool.updatedAt).format("YYYY-MM-DD HH:mm") : "—"}
                </Descriptions.Item>
              </Descriptions>
            ),
          },
          {
            key: "units",
            label: t("pools.tabUnits"),
            children: units.length ? (
              <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
                {units.map((u) => (
                  <Card key={u.name} size="small" className="bg-surface-warm">
                    <div className="font-mono text-sm font-medium text-fg">{u.name}</div>
                    {u.description && (
                      <div className="mt-0.5 text-xs text-muted">{u.description}</div>
                    )}
                    <div className="mt-2 flex flex-wrap gap-1">
                      {Object.entries(u.requests ?? {}).map(([k, v]) => (
                        <Tag key={k} className="!m-0 font-mono text-xs">
                          {k}={v}
                        </Tag>
                      ))}
                    </div>
                  </Card>
                ))}
              </div>
            ) : (
              <Empty description={t("pools.unitsEmpty")} />
            ),
          },
        ]}
      />
    </Drawer>
  );
}

// ── Pool create / edit drawer (numbered FieldSections, mirrors workspace form) ─
interface PoolFormValues {
  name: string;
  description?: string;
  selector?: string;
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
  const [form] = Form.useForm<PoolFormValues>();
  const editing = !!pool;

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

  const onFinish = (v: PoolFormValues) => {
    const nodeSelector = parseSelector(v.selector);
    if (editing) {
      update.mutate(
        {
          pool: pool!.name,
          body: { description: v.description?.trim() || undefined, nodeSelector },
        },
        { onSuccess: onClose },
      );
    } else {
      create.mutate(
        {
          name: v.name.trim(),
          description: v.description?.trim() || undefined,
          nodeSelector,
        },
        { onSuccess: onClose },
      );
    }
  };

  return (
    <Drawer
      open
      width={560}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">
            {editing ? t("pools.drawerEdit") : t("pools.drawerNew")}
          </div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            {editing ? <span className="font-mono">{pool!.name}</span> : t("pools.drawerNewSub")}
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={pending} onClick={() => form.submit()}>
            {editing ? t("common.save") : t("pools.createPool")}
          </Button>
        </div>
      }
    >
      <Form<PoolFormValues>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{
          name: pool?.name ?? "",
          description: pool?.description ?? "",
          selector: selectorPairs(pool?.nodeSelector).join(", "),
        }}
      >
        <FieldSection n={1} title={t("pools.fsBasic")} />
        <Form.Item
          name="name"
          label={t("pools.fName")}
          rules={[{ required: !editing, message: t("pools.fNameHelp") }]}
          extra={!editing ? t("pools.fNameHelp") : undefined}
        >
          <Input className="font-mono" placeholder={t("pools.fNamePlaceholder")} disabled={editing} />
        </Form.Item>
        <Form.Item name="description" label={t("pools.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("pools.fDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("pools.fsSchedule")} />
        <Form.Item name="selector" label={t("pools.fSelector")} extra={t("pools.fSelectorHelp")}>
          <Input className="font-mono" placeholder={t("pools.fSelectorPlaceholder")} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}

// ── Manage-units drawer: CRUD the pool's inline units[] via Form.List ──────────
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
  const [form] = Form.useForm<{ units: UnitFormRow[] }>();
  const existing = useMemo(() => new Set((pool.units ?? []).map((u) => u.name)), [pool.units]);

  const createUnit = useApiMutation(
    (body: sdk.ResourceUnitCreateRequest) =>
      sdk.createResourceUnit({ path: { pool: pool.name }, body }),
    { invalidate: [["resourcepools"]], success: t("pools.unitsSaved") },
  );
  const delUnit = useApiMutation(
    (unit: string) => sdk.deleteResourceUnit({ path: { pool: pool.name, unit } }),
    { invalidate: [["resourcepools"]], success: t("pools.unitsSaved") },
  );

  const onFinish = (v: { units: UnitFormRow[] }) => {
    // Persist only newly added units; existing ones are managed via delete.
    const toCreate = (v.units ?? []).filter((r) => r.name?.trim() && !existing.has(r.name.trim()));
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

  const removeExisting = (name: string, remove: () => void) =>
    confirm({
      title: t("pools.deleteTitle", { name }),
      desc: t("pools.deleteDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => {
        delUnit.mutate(name);
        remove();
      },
    });

  return (
    <Drawer
      open
      width={680}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{t("pools.unitsDrawerTitle")}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            <span className="font-mono">{pool.name}</span>
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={createUnit.isPending} onClick={() => form.submit()}>
            {t("common.save")}
          </Button>
        </div>
      }
    >
      {delUnit.isPending && (
        <div className="mb-3 flex justify-center">
          <Spin size="small" />
        </div>
      )}
      <p className="mb-4 text-xs text-muted">{t("pools.unitsDrawerSub")}</p>
      <Form<{ units: UnitFormRow[] }>
        form={form}
        layout="vertical"
        size="large"
        onFinish={onFinish}
        initialValues={{ units: (pool.units ?? []).map(unitToRow) }}
      >
        <FieldSection n={1} title={t("pools.fsUnits")} />
        <Form.List name="units">
          {(fields, { add, remove }) => (
            <div className="space-y-3">
              {fields.map((field) => {
                const name = form.getFieldValue(["units", field.name, "name"]) as string | undefined;
                const isExisting = !!name && existing.has(name);
                return (
                  <Card key={field.key} size="small" className="bg-surface-warm">
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-semibold text-fg">
                        {name || t("pools.uName")}
                      </span>
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() =>
                          isExisting ? removeExisting(name!, () => remove(field.name)) : remove(field.name)
                        }
                      />
                    </div>
                    <div className="grid grid-cols-2 gap-x-3">
                      <Form.Item
                        name={[field.name, "name"]}
                        label={t("pools.uName")}
                        rules={[{ required: true }]}
                      >
                        <Input
                          className="font-mono"
                          placeholder={t("pools.uNamePlaceholder")}
                          disabled={isExisting}
                        />
                      </Form.Item>
                      <Form.Item name={[field.name, "description"]} label={t("pools.uDesc")}>
                        <Input placeholder={t("pools.uDescPlaceholder")} disabled={isExisting} />
                      </Form.Item>
                      <Form.Item name={[field.name, "cpu"]} label={`${t("pools.uCpu")} (${t("pools.uCpuUnit")})`}>
                        <InputNumber min={0} className="!w-full" disabled={isExisting} />
                      </Form.Item>
                      <Form.Item name={[field.name, "memory"]} label={`${t("pools.uMem")} (${t("pools.uMemUnit")})`}>
                        <InputNumber min={0} className="!w-full" disabled={isExisting} />
                      </Form.Item>
                      <Form.Item name={[field.name, "gpu"]} label={`${t("pools.uGpu")} (${t("pools.uGpuUnit")})`}>
                        <InputNumber min={0} className="!w-full" disabled={isExisting} />
                      </Form.Item>
                    </div>
                  </Card>
                );
              })}
              <Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({})}>
                {t("pools.addUnit")}
              </Button>
            </div>
          )}
        </Form.List>
      </Form>
    </Drawer>
  );
}
