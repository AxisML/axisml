import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  Table,
  Button,
  Input,
  Select,
  Space,
  Card,
  Divider,
  Drawer,
  Form,
  InputNumber,
  Collapse,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined, SearchOutlined, CloseOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useExperiments } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/PageContainer";
import { FieldSection } from "@/components/FieldSection";
import { CardRadio } from "@/components/CardRadio";

interface ExpRow {
  name: string;
  desc: string;
  runCount: number;
  owner: string;
  updated: string;
}

type DrawerMode = "new" | "run" | "edit";

// Recent-run status strip placeholder. Run-status roll-ups are a backend feature
// not yet served, so we render an inert muted strip rather than fabricate states.
function RunStrip() {
  return (
    <div className="flex gap-1" aria-hidden>
      {Array.from({ length: 5 }).map((_, i) => (
        <span key={i} className="h-4 w-2.5 rounded-sm bg-border-soft" />
      ))}
    </div>
  );
}

export default function Experiments() {
  const q = useExperiments();
  const { t } = useTranslation();
  const { confirm } = useUI();
  const [drawer, setDrawer] = useState<{ mode: DrawerMode; name?: string } | null>(null);
  const [search, setSearch] = useState("");
  const [creator, setCreator] = useState<string>("");

  const delExp = useApiMutation((name: string) => sdk.deleteExperiment({ path: { name } }), {
    invalidate: [["experiments"]],
    success: t("experiments.deleted"),
  });
  const triggerExp = useApiMutation(
    (name: string) => sdk.triggerExperimentRun({ path: { name }, body: {} }),
    { invalidate: [["experiments"]], success: t("experiments.runTriggered") },
  );

  const allRows: ExpRow[] = useMemo(
    () =>
      q.data?.items?.map((e) => ({
        name: e.name,
        desc: e.description ?? e.displayName ?? "",
        runCount: 0,
        owner: e.owner ?? "—",
        updated: e.updatedAt ?? e.createdAt ?? "",
      })) ?? [],
    [q.data],
  );

  const creatorOptions = useMemo(
    () => Array.from(new Set(allRows.map((r) => r.owner).filter((o) => o && o !== "—"))),
    [allRows],
  );

  const rows = allRows.filter(
    (r) => (!search || r.name.includes(search)) && (!creator || r.owner === creator),
  );

  const onDelete = (r: ExpRow) =>
    confirm({
      title: t("experiments.deleteTitle", { name: r.name }),
      desc: t("experiments.deleteDesc"),
      info: t("experiments.deleteInfo"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delExp.mutate(r.name),
    });

  const onRun = (r: ExpRow) =>
    confirm({
      title: t("experiments.runTitle", { name: r.name }),
      desc: t("experiments.runDesc"),
      okLabel: t("experiments.confirmRun"),
      danger: false,
      onConfirm: () => triggerExp.mutate(r.name),
    });

  const columns: ColumnsType<ExpRow> = [
    {
      title: t("experiments.colName"),
      dataIndex: "name",
      render: (_, r) => (
        <div className="min-w-0">
          <Link to={`/experiments/${r.name}`} className="font-mono font-medium">
            {r.name}
          </Link>
          {r.desc && <div className="truncate text-xs text-muted">{r.desc}</div>}
        </div>
      ),
    },
    { title: t("experiments.colStatus"), key: "runs", width: 140, render: () => <RunStrip /> },
    { title: t("experiments.colRuns"), dataIndex: "runCount", width: 90, align: "right" },
    { title: t("experiments.colCreator"), dataIndex: "owner", width: 140 },
    {
      title: t("experiments.colUpdated"),
      dataIndex: "updated",
      width: 180,
      render: (v: string) => <span className="text-muted">{v ? dayjs(v).format("YYYY-MM-DD HH:mm") : "—"}</span>,
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 160,
      align: "right",
      render: (_, r) => (
        <Space size={4} split={<Divider type="vertical" className="!mx-0" />}>
          <Button type="link" size="small" className="!px-1" onClick={() => onRun(r)}>
            {t("common.run")}
          </Button>
          <Link to={`/experiments/${r.name}`}>
            <Button type="link" size="small" className="!px-1">
              {t("common.detail")}
            </Button>
          </Link>
          <Button type="link" size="small" className="!px-1" onClick={() => setDrawer({ mode: "edit", name: r.name })}>
            {t("common.edit")}
          </Button>
          <Button type="link" size="small" danger className="!px-1" onClick={() => onDelete(r)}>
            {t("common.delete")}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.trainingCenter"), t("nav.experiments")]}
      title={t("experiments.title")}
      subtitle={t("experiments.subtitle")}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setDrawer({ mode: "new" })}>
          {t("experiments.newExperiment")}
        </Button>
      }
    >
      <Card styles={{ body: { padding: 0 } }} className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-3 border-b border-border-soft p-4">
          <Input
            allowClear
            prefix={<SearchOutlined className="text-muted" />}
            placeholder={t("experiments.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
          <Select
            value={creator || undefined}
            onChange={(v) => setCreator(v ?? "")}
            placeholder={t("experiments.creatorAll")}
            allowClear
            className="min-w-44"
            options={creatorOptions.map((o) => ({ label: o, value: o }))}
          />
          <Button
            onClick={() => {
              setSearch("");
              setCreator("");
            }}
          >
            {t("common.reset")}
          </Button>
        </div>
        <Table<ExpRow>
          rowKey="name"
          columns={columns}
          dataSource={rows}
          loading={q.isLoading}
          pagination={{ pageSize: 20, showTotal: (n) => t("experiments.total", { count: n }), hideOnSinglePage: false }}
          locale={{ emptyText: q.isError ? t("common.loadFailed") : t("common.noData") }}
        />
      </Card>

      {drawer && <ExpDrawer mode={drawer.mode} name={drawer.name} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Create / Run / Edit drawer ────────────────────────────────────────────────
const IMAGES: { value: string; title: string; desc: string }[] = [
  { value: "pytorch:2.3-cu121", title: "pytorch:2.3-cu121", desc: "PyTorch 训练镜像" },
  { value: "megatron:24.05", title: "megatron:24.05", desc: "Megatron-LM 训练镜像" },
];
const UNITS: { value: string; title: string; desc: string }[] = [
  { value: "a100-4x-xlarge", title: "a100-4x-xlarge", desc: "4×A100 · 32 vCPU · 256 GiB" },
  { value: "a100-8x-xlarge-ib", title: "a100-8x-xlarge-ib", desc: "8×A100 · IB · 64 vCPU · 512 GiB" },
];
const POOLS = ["gpu-a100", "gpu-h100"];
const CMD = `torchrun --nproc_per_node=4 sft.py \\
  --base llama3-8b-base --lr {{lr}} --epochs 3`;

interface ExpFormValues {
  name: string;
  description?: string;
  image: string;
  poolName: string;
  unitName: string;
  replicas: number;
  command: string;
  env?: string;
  timeout?: number;
  retries?: number;
}

function parseEnv(text: string): sdk.EnvVar[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
    .map((l) => {
      const [name, ...rest] = l.split("=");
      return { name: name.trim(), value: rest.join("=") };
    })
    .filter((e) => e.name);
}

function parseCommand(text: string): string[] {
  return text
    .replace(/\\\s*\n/g, " ")
    .split(/\s+/)
    .map((s) => s.trim())
    .filter((s) => s && s !== "\\");
}

function ExpDrawer({ mode, name: initialName, onClose }: { mode: DrawerMode; name?: string; onClose: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm<ExpFormValues>();
  const locked = mode === "run";

  const buildSpec = (v: ExpFormValues): sdk.JobSpec => {
    const reps = Number(v.replicas);
    const role: sdk.MlRunRole = {
      name: "worker",
      replicas: Number.isFinite(reps) && reps > 0 ? reps : 1,
      template: {
        image: v.image?.trim() || undefined,
        command: parseCommand(v.command || ""),
        env: parseEnv(v.env || ""),
      },
    };
    return {
      backend: { name: "native", engine: "job" },
      poolName: v.poolName?.trim() || undefined,
      unitName: v.unitName?.trim() || undefined,
      roles: [role],
      runPolicy: {
        activeDeadlineSeconds: v.timeout && v.timeout > 0 ? v.timeout : undefined,
        backoffLimit: v.retries != null && v.retries >= 0 ? v.retries : undefined,
      },
    };
  };

  const create = useApiMutation((body: sdk.ExperimentCreateRequest) => sdk.createExperiment({ body }), {
    invalidate: [["experiments"]],
    success: t("experiments.created"),
  });
  const update = useApiMutation(
    (vars: { name: string; body: sdk.ExperimentPatchRequest }) => sdk.updateExperiment({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["experiments"]], success: t("experiments.saved") },
  );
  const trigger = useApiMutation(
    (vars: { name: string; body: sdk.RunTriggerRequest }) => sdk.triggerExperimentRun({ path: { name: vars.name }, body: vars.body }),
    { invalidate: [["experiments"]], success: t("experiments.runTriggered") },
  );
  const pending = create.isPending || update.isPending || trigger.isPending;

  const onFinish = (v: ExpFormValues) => {
    if (mode === "new") {
      create.mutate(
        { name: v.name.trim(), displayName: v.name.trim() || undefined, description: v.description?.trim() || undefined, spec: buildSpec(v) },
        { onSuccess: onClose },
      );
    } else if (mode === "edit") {
      update.mutate(
        { name: v.name.trim(), body: { description: v.description?.trim() || undefined, spec: buildSpec(v) } },
        { onSuccess: onClose },
      );
    } else {
      trigger.mutate(
        {
          name: v.name.trim(),
          body: { poolName: v.poolName?.trim() || undefined, unitName: v.unitName?.trim() || undefined, roles: [{ name: "worker", args: parseCommand(v.command || ""), env: parseEnv(v.env || "") }] },
        },
        { onSuccess: onClose },
      );
    }
  };

  const title = mode === "new" ? t("experiments.drawerNew") : mode === "run" ? t("experiments.drawerRun") : t("experiments.drawerEdit");
  const submitLabel = mode === "new" ? t("experiments.createExperiment") : mode === "run" ? t("experiments.confirmRun") : t("experiments.save");

  return (
    <Drawer
      open
      width={640}
      onClose={onClose}
      closable={false}
      title={
        <div>
          <div className="text-base font-semibold text-fg">{title}</div>
          <div className="mt-0.5 text-xs font-normal text-muted">
            {mode === "new" ? t("experiments.drawerNewSub") : <span className="font-mono">{initialName}</span>}
          </div>
        </div>
      }
      extra={<Button type="text" icon={<CloseOutlined />} onClick={onClose} />}
      footer={
        <div className="flex justify-end gap-2">
          <Button onClick={onClose}>{t("common.cancel")}</Button>
          <Button type="primary" loading={pending} onClick={() => form.submit()}>
            {submitLabel}
          </Button>
        </div>
      }
    >
      <Form<ExpFormValues>
        form={form}
        layout="vertical"
        size="large"
        disabled={locked}
        onFinish={onFinish}
        initialValues={{
          name: mode === "new" ? "" : initialName,
          image: IMAGES[0].value,
          poolName: POOLS[0],
          unitName: UNITS[0].value,
          replicas: 2,
          command: CMD,
          env: "WANDB_DISABLED=true\nNCCL_DEBUG=INFO",
          timeout: 172800,
          retries: 1,
        }}
      >
        <FieldSection n={1} title={t("experiments.fsBasic")} />
        <Form.Item name="name" label={t("experiments.fName")} rules={[{ required: mode !== "run", message: t("experiments.fNameHelp") }]} extra={mode !== "run" ? t("experiments.fNameHelp") : undefined}>
          <Input className="font-mono" placeholder={t("experiments.fNamePlaceholder")} disabled={locked || mode === "edit"} />
        </Form.Item>
        <Form.Item name="description" label={t("experiments.fDesc")}>
          <Input.TextArea rows={2} placeholder={t("experiments.fDescPlaceholder")} />
        </Form.Item>

        <FieldSection n={2} title={t("experiments.fsImage")} />
        <Form.Item name="image" label={t("experiments.fImage")} rules={[{ required: true }]}>
          <CardRadio options={IMAGES} disabled={locked} />
        </Form.Item>

        <FieldSection n={3} title={t("experiments.fsResource")} />
        <Form.Item name="poolName" label={t("experiments.fPool")} rules={[{ required: true }]}>
          <Select options={POOLS.map((p) => ({ label: p, value: p }))} />
        </Form.Item>
        <Form.Item name="unitName" label={t("experiments.fUnit")} rules={[{ required: true }]}>
          <CardRadio options={UNITS} disabled={locked} />
        </Form.Item>
        <Form.Item name="replicas" label={t("experiments.fReplicas")} rules={[{ required: true }]}>
          <InputNumber min={1} className="!w-40" />
        </Form.Item>

        <FieldSection n={4} title={t("experiments.fsCommand")} />
        <Form.Item name="command" label={mode === "run" ? t("experiments.fCommandRun") : t("experiments.fCommandTpl")} extra={mode === "run" ? t("experiments.fCommandHelpRun") : t("experiments.fCommandHelpTpl")}>
          <Input.TextArea rows={3} className="font-mono" />
        </Form.Item>
        <Form.Item name="env" label={t("experiments.fEnv")} extra={mode !== "run" ? t("experiments.fEnvHelp") : undefined}>
          <Input.TextArea rows={2} className="font-mono" />
        </Form.Item>

        <Collapse
          ghost
          className="!px-0"
          items={[
            {
              key: "adv",
              label: <span className="text-sm font-semibold text-accent">{t("common.advanced")}</span>,
              children: (
                <div className="flex gap-4">
                  <Form.Item name="timeout" label={t("experiments.fTimeout")} className="flex-1">
                    <InputNumber min={0} className="!w-full" />
                  </Form.Item>
                  <Form.Item name="retries" label={t("experiments.fRetries")} className="flex-1">
                    <InputNumber min={0} className="!w-full" />
                  </Form.Item>
                </div>
              ),
            },
          ]}
        />
      </Form>
    </Drawer>
  );
}
