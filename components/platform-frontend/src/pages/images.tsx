import { useMemo, useState } from "react";
import {
  Plus,
  Trash2,
  CloudDownload,
  Copy,
  Container,
  X,
  LayoutGrid,
  List as ListIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { useImages, useImageVersions } from "@/api/hooks";
import { useApiMutation } from "@/api/mutations";
import * as sdk from "@/api/generated";
import { useApp } from "@/app/store";
import { useUI } from "@/app/ui";
import { PageContainer } from "@/components/page-container";
import { FieldSection } from "@/components/field-section";
import { StatusDot } from "@/components/phase-tag";
import { SearchInput } from "@/components/search-input";
import { AssetCard } from "@/components/asset-card";
import { CodeBlock } from "@/components/code-block";
import { DataTable, type Column } from "@/components/data-table";
import { USE_MOCK } from "@/api/mock";
import { imageVersions } from "@/api/mock/data";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
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
import { FormDrawer } from "@/components/form-drawer";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Field, FieldLabel, FieldDescription, FieldGroup } from "@/components/ui/field";
import { phaseTone } from "@/lib/phase";
import { cn } from "@/lib/utils";

interface ImageRow {
  name: string;
  desc: string;
  purpose: string;
  latest: string;
  versions: number;
  updated: string;
}

type DrawerState =
  | { kind: "versions"; image: string; desc: string; purpose: string }
  | { kind: "pull"; image: string; version: string; uri: string }
  | { kind: "new" }
  | { kind: "add"; image: string };

export default function Images() {
  const q = useImages();
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { confirm } = useUI();
  const [view, setView] = useState<"cards" | "list">("cards");
  const [search, setSearch] = useState("");
  const [drawer, setDrawer] = useState<DrawerState | null>(null);

  const delImage = useApiMutation(
    (name: string) => sdk.deleteImageDefinition({ path: { tenant, name } }),
    { invalidate: [["images"]], success: t("images.imageDeleted") },
  );

  const allRows: ImageRow[] = useMemo(
    () =>
      q.data?.items?.map((m) => {
        const vs = USE_MOCK ? imageVersions(m.name) : [];
        return {
          name: m.name,
          desc: m.description ?? m.displayName ?? "",
          purpose: (m.labels?.purpose as string) ?? "—",
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

  const openVersions = (r: ImageRow) =>
    setDrawer({ kind: "versions", image: r.name, desc: r.desc, purpose: r.purpose });

  const onDeleteImage = (r: ImageRow) =>
    confirm({
      title: t("images.deleteImageTitle", { name: r.name }),
      desc: t("images.deleteImageDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delImage.mutate(r.name),
    });

  const columns: Column<ImageRow>[] = [
    {
      key: "name",
      title: t("images.colName"),
      render: (r) => (
        <button type="button" className="min-w-0 text-left" onClick={() => openVersions(r)}>
          <div className="font-mono font-medium text-info hover:underline">{r.name}</div>
          {r.desc && <div className="truncate text-xs text-muted-foreground">{r.desc}</div>}
        </button>
      ),
    },
    {
      key: "purpose",
      title: t("images.colPurpose"),
      width: 140,
      render: (r) =>
        r.purpose && r.purpose !== "—" ? (
          <Badge variant="secondary">{r.purpose}</Badge>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
    },
    {
      key: "latest",
      title: t("images.colLatest"),
      width: 130,
      render: (r) => <span className="font-mono text-muted-foreground">{r.latest}</span>,
    },
    {
      key: "versions",
      title: t("images.colVersions"),
      width: 90,
      align: "right",
      render: (r) => <span className="font-mono">{r.versions}</span>,
    },
    {
      key: "updated",
      title: t("images.colUpdated"),
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
          <Button variant="link" size="sm" onClick={() => setDrawer({ kind: "add", image: r.name })}>
            {t("images.addVersion")}
          </Button>
          <Button variant="link" size="sm" className="text-destructive" onClick={() => onDeleteImage(r)}>
            {t("common.delete")}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <PageContainer
      breadcrumb={[t("nav.assetCenter"), t("nav.images")]}
      title={t("images.title")}
      subtitle={t("images.subtitle")}
      extra={
        <Button onClick={() => setDrawer({ kind: "new" })}>
          <Plus data-icon="inline-start" />
          {t("images.newImage")}
        </Button>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <SearchInput
          className="max-w-xs flex-1"
          placeholder={t("images.searchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
        <div className="grow" />
        <ToggleGroup
          type="single"
          variant="outline"
          spacing={0}
          value={view}
          onValueChange={(v) => v && setView(v as "cards" | "list")}
        >
          <ToggleGroupItem value="cards" aria-label={t("images.viewCards")}>
            <LayoutGrid data-icon="inline-start" />
            {t("images.viewCards")}
          </ToggleGroupItem>
          <ToggleGroupItem value="list" aria-label={t("images.viewList")}>
            <ListIcon data-icon="inline-start" />
            {t("images.viewList")}
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
                <Container />
              </EmptyMedia>
              <EmptyTitle>{q.isError ? t("common.loadFailed") : t("common.noData")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
              {rows.map((r) => (
                <AssetCard
                  key={r.name}
                  icon={Container}
                  name={r.name}
                  desc={r.desc}
                  badge={r.purpose}
                  latest={r.latest}
                  updated={r.updated ? dayjs(r.updated).fromNow() : "—"}
                  versionsText={`${r.versions} ${t("images.versionsSuffix")}`}
                  onOpen={() => openVersions(r)}
                  onDelete={() => onDeleteImage(r)}
                  deleteLabel={t("common.delete")}
                />
              ))}
            </div>
            <div className="mt-4 text-sm text-muted-foreground">
              {t("images.total", { count: rows.length })}
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
          image={drawer.image}
          desc={drawer.desc}
          purpose={drawer.purpose}
          onClose={() => setDrawer(null)}
          onPull={(version, uri) => setDrawer({ kind: "pull", image: drawer.image, version, uri })}
          onAdd={() => setDrawer({ kind: "add", image: drawer.image })}
        />
      )}
      {drawer?.kind === "pull" && (
        <PullDrawer image={drawer.image} version={drawer.version} uri={drawer.uri} onClose={() => setDrawer(null)} />
      )}
      {drawer?.kind === "new" && <NewImageDrawer onClose={() => setDrawer(null)} />}
      {drawer?.kind === "add" && <AddVersionDrawer image={drawer.image} onClose={() => setDrawer(null)} />}
    </PageContainer>
  );
}

// ── Version list drawer ───────────────────────────────────────────────────────
function VersionsDrawer({
  image,
  desc,
  purpose,
  onClose,
  onPull,
  onAdd,
}: {
  image: string;
  desc: string;
  purpose: string;
  onClose: () => void;
  onPull: (version: string, uri: string) => void;
  onAdd: () => void;
}) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const { toast, confirm } = useUI();
  const [search, setSearch] = useState("");
  const versQ = useImageVersions(image);

  const delVer = useApiMutation(
    (version: string) => sdk.deleteImage({ path: { tenant, name: image, version } }),
    { invalidate: [["images"]], success: t("images.verDeleted") },
  );

  const items = versQ.data?.items ?? [];
  const filtered = items.filter((v) => {
    const needle = search.trim().toLowerCase();
    if (!needle) return true;
    return v.version.toLowerCase().includes(needle) || (v.description ?? "").toLowerCase().includes(needle);
  });

  const onDeleteVer = (version: string) =>
    confirm({
      title: t("images.deleteVerTitle", { version }),
      desc: t("images.deleteVerDesc"),
      okLabel: t("common.confirmDelete"),
      onConfirm: () => delVer.mutate(version),
    });

  return (
    <FormDrawer
      title={image}
      titleClassName="font-mono"
      subtitle={
        desc || (purpose && purpose !== "—") ? (
          <span className="mt-1 flex flex-col gap-2">
            {desc && <span>{desc}</span>}
            {purpose && purpose !== "—" && (
              <span>
                <Badge variant="secondary">{purpose}</Badge>
              </span>
            )}
          </span>
        ) : undefined
      }
      onClose={onClose}
      bodyClassName="flex flex-col"
    >
      <div className="mb-4 flex items-center gap-3">
        <SearchInput
          className="flex-1"
          placeholder={t("images.verSearchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
        <Button onClick={onAdd}>
          <Plus data-icon="inline-start" />
          {t("images.addVersion")}
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
        <div className="flex flex-col gap-3">
          {filtered.map((v) => {
            const pending = v.status === "Uploading";
            return (
              <div
                key={v.version}
                className="rounded-lg border p-4 transition-colors hover:border-primary/30"
              >
                <div className="flex items-center gap-3">
                  <span className="font-mono text-[15px] font-semibold text-foreground">{v.version}</span>
                  <StatusDot tone={phaseTone(v.status)} pulse={pending}>
                    {t(`images.status${v.status}`, { defaultValue: v.status })}
                  </StatusDot>
                  {v.source && <Badge variant="outline">{v.source}</Badge>}
                  {!pending && (
                    <div className="ml-auto flex items-center gap-0.5">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => onPull(v.version, v.uri ?? "")}
                          >
                            <CloudDownload />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("images.pullTitle")}</TooltipContent>
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
                {v.description && (
                  <div className="mt-2 text-sm text-muted-foreground">{v.description}</div>
                )}
                <div className="mt-2.5 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                  <span className={cn("font-mono break-all", pending ? "" : "text-foreground")}>
                    {v.uri ?? t("images.addrPending")}
                  </span>
                  {v.uri && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => {
                            void navigator.clipboard?.writeText(v.uri ?? "");
                            toast(t("images.addrCopied"));
                          }}
                        >
                          <Copy />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("common.actions")}</TooltipContent>
                    </Tooltip>
                  )}
                  {(v.owner || v.createdAt) && (
                    <span className="ml-auto whitespace-nowrap pl-3">
                      {[v.owner, v.createdAt && dayjs(v.createdAt).fromNow()]
                        .filter(Boolean)
                        .join(" · ")}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </FormDrawer>
  );
}

// ── Pull-command drawer ───────────────────────────────────────────────────────
function PullDrawer({ image, version, uri, onClose }: { image: string; version: string; uri: string; onClose: () => void }) {
  const { t } = useTranslation();
  const ref = uri || `zot.axisml.internal/<tenant>/${image}:${version}`;
  const cmd = `docker login zot.axisml.internal -u <user> -p <token>\ndocker pull ${ref}`;

  return (
    <FormDrawer
      title={t("images.pullTitle")}
      subtitle={<span className="font-mono">{`${image}:${version}`}</span>}
      onClose={onClose}
      footer={<Button onClick={onClose}>{t("images.done")}</Button>}
    >
      <p className="mb-3 text-sm text-muted-foreground">{t("images.pullHint")}</p>
      <CodeBlock copy={cmd}>{cmd}</CodeBlock>
    </FormDrawer>
  );
}

// ── New-image drawer ──────────────────────────────────────────────────────────
interface NewImageValues {
  name: string;
  purpose: string;
  description: string;
}

function NewImageDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [submitted, setSubmitted] = useState(false);
  const [v, setV] = useState<NewImageValues>({ name: "", purpose: "training", description: "" });
  const set = <K extends keyof NewImageValues>(k: K, val: NewImageValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));
  const [customTags, setCustomTags] = useState<Record<string, string>>({});
  const [ctKey, setCtKey] = useState("");
  const [ctVal, setCtVal] = useState("");

  const PURPOSE_OPTIONS = [
    { value: "training", label: t("images.purposeTraining") },
    { value: "inference", label: t("images.purposeInference") },
    { value: "workspace", label: t("images.purposeWorkspace") },
    { value: "custom", label: t("images.purposeCustom") },
  ];

  const create = useApiMutation(
    (body: sdk.ArtifactDefinitionCreateRequest) => sdk.createImageDefinition({ path: { tenant, name: body.name }, body }),
    { invalidate: [["images"]], success: t("images.imageCreated") },
  );

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
    const labels: Record<string, string> = { ...customTags };
    if (v.purpose) labels.purpose = v.purpose;
    create.mutate(
      {
        name: v.name.trim(),
        description: v.description.trim() || undefined,
        labels: Object.keys(labels).length ? labels : undefined,
      },
      { onSuccess: onClose },
    );
  };

  return (
    <FormDrawer
      title={t("images.newImageTitle")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("images.createImage")}
      submitting={create.isPending}
    >
      <FieldSection n={1} title={t("images.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="image-name">
            {t("images.fName")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="image-name"
            className="font-mono"
            placeholder={t("images.fNamePlaceholder")}
            value={v.name}
            aria-invalid={submitted && !v.name.trim()}
            onChange={(e) => set("name", e.target.value)}
          />
          <FieldDescription>{t("images.fNameHelp")}</FieldDescription>
        </Field>
        <Field>
          <FieldLabel>{t("images.fPurpose")}</FieldLabel>
          <Select value={v.purpose} onValueChange={(val) => set("purpose", val)}>
            <SelectTrigger className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {PURPOSE_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field>
          <FieldLabel htmlFor="image-desc">{t("images.fDesc")}</FieldLabel>
          <Textarea
            id="image-desc"
            rows={2}
            placeholder={t("images.fDescPlaceholder")}
            value={v.description}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("images.fsLabels")} />
      <FieldGroup>
        <Field>
          <FieldLabel>{t("images.lCustom")}</FieldLabel>
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
              placeholder={t("images.customKeyPlaceholder")}
              value={ctKey}
              onChange={(e) => setCtKey(e.target.value)}
            />
            <Input
              className="font-mono"
              placeholder={t("images.customValPlaceholder")}
              value={ctVal}
              onChange={(e) => setCtVal(e.target.value)}
            />
            <Button variant="outline" onClick={addTag}>
              {t("images.add")}
            </Button>
          </div>
        </Field>
      </FieldGroup>
    </FormDrawer>
  );
}

// ── Add-version drawer (two methods: external register vs Docker push) ─────────
interface AddVersionValues {
  version: string;
  description: string;
  sourceImageRef: string;
}

function AddVersionDrawer({ image, onClose }: { image: string; onClose: () => void }) {
  const { t } = useTranslation();
  const { tenant } = useApp();
  const [submitted, setSubmitted] = useState(false);
  const [method, setMethod] = useState<"external" | "dockerPush">("external");
  const [v, setV] = useState<AddVersionValues>({ version: "", description: "", sourceImageRef: "" });
  const set = <K extends keyof AddVersionValues>(k: K, val: AddVersionValues[K]) =>
    setV((prev) => ({ ...prev, [k]: val }));

  const initiate = useApiMutation(
    (body: sdk.ImageInitiateRequest) => sdk.initiateImage({ path: { tenant, name: image }, body }),
    { invalidate: [["images"]], success: t("images.versionSubmitted") },
  );

  const submit = () => {
    setSubmitted(true);
    if (!v.version.trim()) return;
    if (method === "external" && !v.sourceImageRef.trim()) return;
    const body: sdk.ImageInitiateRequest =
      method === "external"
        ? {
            version: v.version.trim(),
            spec: {},
            description: v.description.trim() || undefined,
            source: "external",
            sourceImageRef: v.sourceImageRef.trim() || undefined,
          }
        : {
            version: v.version.trim(),
            spec: {},
            description: v.description.trim() || undefined,
            source: "dockerPush",
          };
    initiate.mutate(body, { onSuccess: onClose });
  };

  return (
    <FormDrawer
      title={t("images.addVerTitle")}
      onClose={onClose}
      onSubmit={submit}
      submitLabel={t("images.submit")}
      submitting={initiate.isPending}
    >
      <FieldSection n={1} title={t("images.fsBasic")} />
      <FieldGroup>
        <Field>
          <FieldLabel>{t("images.fImage")}</FieldLabel>
          <Input className="font-mono" value={image} disabled />
        </Field>
        <Field>
          <FieldLabel htmlFor="addver-version">
            {t("images.fVersion")}
            <span className="text-destructive">*</span>
          </FieldLabel>
          <Input
            id="addver-version"
            className="font-mono"
            placeholder={t("images.fVersionPlaceholder")}
            value={v.version}
            aria-invalid={submitted && !v.version.trim()}
            onChange={(e) => set("version", e.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="addver-desc">{t("images.fDesc")}</FieldLabel>
          <Textarea
            id="addver-desc"
            rows={2}
            placeholder={t("images.fAddVerDescPlaceholder")}
            value={v.description}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>
      </FieldGroup>

      <FieldSection n={2} title={t("images.fsMethod")} />
      <Tabs value={method} onValueChange={(k) => setMethod(k as "external" | "dockerPush")}>
        <TabsList>
          <TabsTrigger value="external">{t("images.methodExternal")}</TabsTrigger>
          <TabsTrigger value="dockerPush">{t("images.methodDocker")}</TabsTrigger>
        </TabsList>
        <TabsContent value="external" className="flex flex-col gap-4 pt-4">
          <p className="text-sm text-muted-foreground">{t("images.externalHelp")}</p>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="source-ref">{t("images.fSourceRef")}</FieldLabel>
              <Input
                id="source-ref"
                className="font-mono"
                placeholder={t("images.sourceRefPlaceholder")}
                value={v.sourceImageRef}
                aria-invalid={submitted && method === "external" && !v.sourceImageRef.trim()}
                onChange={(e) => set("sourceImageRef", e.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel>{t("images.fPullCred")}</FieldLabel>
              <Select defaultValue="public">
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="public">{t("images.credPublic")}</SelectItem>
                  <SelectItem value="ngc">{t("images.credNgc")}</SelectItem>
                  <SelectItem value="harbor">{t("images.credHarbor")}</SelectItem>
                  <SelectItem value="new">{t("images.credNew")}</SelectItem>
                </SelectContent>
              </Select>
            </Field>
          </FieldGroup>
        </TabsContent>
        <TabsContent value="dockerPush" className="pt-4">
          <DockerGuide image={image} tenant={tenant} />
        </TabsContent>
      </Tabs>
    </FormDrawer>
  );
}

function DockerGuide({ image, tenant }: { image: string; tenant: string }) {
  const { t } = useTranslation();
  const login = `# 临时凭证有效期 1h
docker login zot.axisml.internal -u <user> -p <token>`;
  const push = `# 1. 为本地镜像打上目标 tag
docker tag <local-image>:<tag> zot.axisml.internal/${tenant}/${image}:<tag>

# 2. 推送到镜像仓
docker push zot.axisml.internal/${tenant}/${image}:<tag>`;
  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-muted-foreground">{t("images.dockerHelp")}</p>
      <div>
        <div className="mb-1 text-sm font-semibold text-foreground">{t("images.dockerStep1")}</div>
        <CodeBlock copy={login}>{login}</CodeBlock>
      </div>
      <div>
        <div className="mb-1 text-sm font-semibold text-foreground">{t("images.dockerStep2")}</div>
        <CodeBlock copy={push}>{push}</CodeBlock>
      </div>
    </div>
  );
}
