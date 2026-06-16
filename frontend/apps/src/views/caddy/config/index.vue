<script lang="ts" setup>
import type { CaddyBlockDraft, CaddyFormModel, CaddyPageMode, PreservedCaddyBlock } from './types';
import type { QuickSiteDraft } from './quick-config-utils';

import { computed, onMounted, reactive, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  Col,
  Collapse,
  CollapsePanel,
  Descriptions,
  DescriptionsItem,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  List,
  message,
  Modal,
  Popconfirm,
  Radio,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
} from 'ant-design-vue';

import {
  addCaddyServerApi,
  deleteCaddyServerApi,
  getCaddyConfigApi,
  getCaddyConfigHistoryDetailApi,
  getCaddyConfigHistoryListApi,
  getCaddyServerListApi,
  previewCaddyConfigApi,
  pushCaddyConfigApi,
  rollbackCaddyConfigApi,
  updateCaddyServerApi,
} from '#/api/caddy/server';
import {
  applySimpleWafConfigApi,
  getSimpleWafConfigApi,
  previewSimpleWafConfigApi,
  updateSimpleWafConfigApi,
} from '#/api/caddy/simple-waf';

import { parseCaddyfileToBlocks, buildCaddyfileFromBlocks } from './caddy-config-blocks';
import {
  buildLineDiff,
  formatCaddyfile,
  genId,
  normalizeModules,
  validateStructuredConfig,
} from './caddy-config-utils';
import {
  buildQuickConfigState,
  createQuickSiteDraft,
  mergeQuickConfigDrafts,
} from './quick-config-utils';

defineOptions({ name: 'CaddyConfig' });

type CaddyServer = Record<string, any>;
type SavePreviewKind = 'blocks' | 'raw';
type SimpleWafMode = 'detectiononly' | 'off' | 'on';
type SimpleWafStrength = 'balanced' | 'high_blocking' | 'low_fp';
type SimpleWafAudit = 'off' | 'on' | 'relevantonly';

const createEmptyFormModel = (): CaddyFormModel => ({
  schemaVersion: 1,
  global: { raw: '' },
  upstreams: [],
  sites: [],
});

const servers = ref<CaddyServer[]>([]);
const selectedServerId = ref<number>();
const loadingServers = ref(false);
const loadingConfig = ref(false);
const saving = ref(false);
const previewing = ref(false);

const configContent = ref('');
const mode = ref<CaddyPageMode>('blocks');
const lastEditMode = ref<'blocks' | 'raw'>('blocks');
const structuredAvailable = ref(false);
const formModel = ref<CaddyFormModel>(createEmptyFormModel());
const preservedBlocks = ref<PreservedCaddyBlock[]>([]);
const quickSiteDrafts = ref<QuickSiteDraft[]>([]);
const complexSiteSummaries = ref<Array<{ domains: string[]; id: string; name: string; reasons: string[] }>>([]);
const activeQuickSiteId = ref<string>();

const serverModalVisible = ref(false);
const serverModalType = ref<'add' | 'edit'>('add');
const serverForm = reactive<Record<string, any>>({
  id: undefined,
  name: '',
  token: '',
  type: 'local',
  url: 'http://localhost:2019',
});

const savePreview = reactive<{
  actions: string[];
  config: string;
  errors: string[];
  kind: SavePreviewKind;
  modules: string;
  open: boolean;
  original: string;
}>({
  actions: [],
  config: '',
  errors: [],
  kind: 'blocks',
  modules: '',
  open: false,
  original: '',
});

const historyDrawerVisible = ref(false);
const historyLoading = ref(false);
const historyList = ref<Record<string, any>[]>([]);
const historyDetailVisible = ref(false);
const historyCompareVisible = ref(false);
const historyDetail = ref<Record<string, any> | null>(null);
const historyCompareLeft = ref('');
const historyDiffOnly = ref(false);

const wafLoading = ref(false);
const wafSaving = ref(false);
const wafApplying = ref(false);
const wafPreviewing = ref(false);
const wafStatus = ref<Record<string, any> | null>(null);
const wafPreviewVisible = ref(false);
const wafPreviewResult = ref<Record<string, any> | null>(null);
const wafForm = reactive({
  audit: 'relevantonly' as SimpleWafAudit,
  enabled: false,
  mode: 'detectiononly' as SimpleWafMode,
  requestBodyAccess: true,
  requestBodyLimitMB: 10,
  requestBodyNoFilesLimitMB: 1,
  siteAddresses: [] as string[],
  strength: 'balanced' as SimpleWafStrength,
});

const modeOptions = [
  { label: '分块配置', value: 'blocks' },
  { label: '防火墙', value: 'waf' },
  { label: '原始配置', value: 'raw' },
  { label: '预览', value: 'preview' },
];

const quickModeOptions = [
  { label: '反向代理', value: 'reverse_proxy' },
  { label: '静态文件', value: 'file_server' },
  { label: '重定向', value: 'redirect' },
];

const tlsModeOptions = [
  { label: '自动 HTTPS', value: 'auto' },
  { label: '内部证书', value: 'internal' },
  { label: '关闭 TLS', value: 'off' },
];

const lbPolicyOptions = [
  { label: '轮询', value: 'round_robin' },
  { label: '最少连接', value: 'least_conn' },
  { label: 'IP Hash', value: 'ip_hash' },
];

const wafModeOptions = [
  { label: '仅检测', value: 'detectiononly' },
  { label: '阻断', value: 'on' },
  { label: '关闭', value: 'off' },
];

const wafStrengthOptions = [
  { label: '低误报', value: 'low_fp' },
  { label: '平衡', value: 'balanced' },
  { label: '严格', value: 'high_blocking' },
];

const wafAuditOptions = [
  { label: '相关请求', value: 'relevantonly' },
  { label: '全量', value: 'on' },
  { label: '关闭', value: 'off' },
];

const serverOptions = computed(() =>
  servers.value.map((server) => ({
    label: serverLabel(server),
    value: Number(server.id),
  })),
);

const selectedServer = computed(() =>
  servers.value.find((server) => Number(server.id) === selectedServerId.value),
);

const activeQuickSite = computed(() =>
  quickSiteDrafts.value.find((site) => site.id === activeQuickSiteId.value),
);

const globalPreservedBlocks = computed(() =>
  preservedBlocks.value
    .map((block, index) => ({ block, index }))
    .filter(({ block }) => block.kind !== 'site'),
);

const preservedSiteBlocks = computed(() =>
  preservedBlocks.value
    .map((block, index) => ({ block, index }))
    .filter(({ block }) => block.kind === 'site'),
);

const mergedQuickFormModel = computed(() =>
  mergeQuickConfigDrafts(formModel.value, quickSiteDrafts.value),
);

const hasPreservedContent = computed(() =>
  preservedBlocks.value.some((block) => block.raw.trim().length > 0),
);

const generatedConfig = computed(() => {
  const draft: CaddyBlockDraft = {
    ...mergedQuickFormModel.value,
    preservedBlocks: preservedBlocks.value,
  };
  return buildCaddyfileFromBlocks(draft, { sourceOrder: configContent.value });
});

const previewConfig = computed(() =>
  formatCaddyfile(
    lastEditMode.value === 'raw'
      ? configContent.value
      : generatedConfig.value.trim()
        ? generatedConfig.value
        : configContent.value,
  ),
);

const quickValidationErrors = computed(() =>
  validateStructuredConfig(mergedQuickFormModel.value, hasPreservedContent.value),
);

const wafAvailableSites = computed(() =>
  Array.isArray(wafStatus.value?.availableSites) ? wafStatus.value.availableSites : [],
);

const wafStatusText = computed(() => {
  if (!wafStatus.value) return '未加载';
  if (!wafStatus.value.enabled) return '关闭';
  if (wafStatus.value.mode === 'on') return '阻断';
  if (wafStatus.value.mode === 'detectiononly') return '仅检测';
  return '关闭';
});

const wafStatusColor = computed(() => {
  if (!wafStatus.value?.enabled) return 'default';
  return wafStatus.value.mode === 'on' ? 'green' : 'orange';
});

const historyCompareRows = computed(() => {
  const rows = buildLineDiff(
    formatCaddyfile(historyCompareLeft.value || ''),
    previewConfig.value || '',
  );
  return historyDiffOnly.value ? rows.filter((row) => row.type !== 'same') : rows;
});

function serverLabel(server: CaddyServer) {
  return server.name ?? server.host ?? server.url ?? `Server #${server.id}`;
}

function blockKindLabel(kind: PreservedCaddyBlock['kind']) {
  if (kind === 'global') return '全局';
  if (kind === 'snippet') return 'snippet';
  if (kind === 'site') return 'site';
  return '未知';
}

function blockKindColor(kind: PreservedCaddyBlock['kind']) {
  if (kind === 'global') return 'green';
  if (kind === 'snippet') return 'blue';
  if (kind === 'site') return 'orange';
  return 'default';
}

function shortHash(value: unknown) {
  const text = String(value ?? '').trim();
  if (text.length <= 16) return text;
  return `${text.slice(0, 8)}...${text.slice(-8)}`;
}

function historyDescription(item: Record<string, any>) {
  return item.summary || item.description || shortHash(item.hash);
}

function diffSideClass(
  row: { left: string | null; right: string | null; type: 'added' | 'changed' | 'removed' | 'same' },
  side: 'left' | 'right',
) {
  if (row.type === 'same') return 'same';
  if (side === 'left') {
    if (row.left === null) return 'blank';
    return 'removed';
  }
  if (row.right === null) return 'blank';
  return 'added';
}

function syncQuickStateFromForm(model: CaddyFormModel) {
  const { simpleSites, complexSites } = buildQuickConfigState(model);
  quickSiteDrafts.value = simpleSites;
  complexSiteSummaries.value = complexSites;
  activeQuickSiteId.value = simpleSites.some((site) => site.id === activeQuickSiteId.value)
    ? activeQuickSiteId.value
    : simpleSites[0]?.id;
}

function applyDraft(draft: CaddyBlockDraft) {
  formModel.value = {
    schemaVersion: draft.schemaVersion,
    global: draft.global,
    upstreams: draft.upstreams,
    sites: draft.sites,
  };
  preservedBlocks.value = draft.preservedBlocks ?? [];
  structuredAvailable.value = true;
  syncQuickStateFromForm(formModel.value);
  lastEditMode.value = 'blocks';
  mode.value = 'blocks';
}

function hasFormContent(model: CaddyFormModel) {
  return Boolean(
    model.sites?.length ||
      model.upstreams?.length ||
      model.global?.raw?.trim(),
  );
}

function hasPreservedBlocks(blocks: PreservedCaddyBlock[]) {
  return blocks.some((block) => block.raw.trim().length > 0);
}

function parseRawToBlocks(showSuccess = false) {
  if (!configContent.value.trim()) {
    message.warning('原始配置为空，无法解析');
    return;
  }
  const parsed = parseCaddyfileToBlocks(configContent.value);
  if (!parsed.sites.length && !parsed.global?.raw && !parsed.preservedBlocks.length) {
    message.error('未解析到可用结构化配置');
    return;
  }
  applyDraft(parsed);
  if (showSuccess) message.success('已从原始配置解析');
}

function loadModules(modules: string | undefined) {
  if (!modules) return false;
  try {
    const parsed = JSON.parse(modules);
    const normalized = normalizeModules(parsed);
    const modulePreservedBlocks = Array.isArray(parsed?.preservedBlocks)
      ? parsed.preservedBlocks
      : [];
    if (!hasFormContent(normalized) && !hasPreservedBlocks(modulePreservedBlocks)) {
      return false;
    }
    formModel.value = normalized;
    preservedBlocks.value = modulePreservedBlocks;
    structuredAvailable.value = true;
    syncQuickStateFromForm(formModel.value);
    return true;
  } catch {
    message.warning('结构化配置解析失败，已回退到原始配置解析');
    return false;
  }
}

function mergePreservedBlocksFromRaw(parsed: CaddyBlockDraft) {
  const siteDomains = new Set(formModel.value.sites.flatMap((site) => site.domains));
  const existingRawSet = new Set(
    preservedBlocks.value.map((block) => block.raw.trim()).filter(Boolean),
  );
  const nextBlocks = [...preservedBlocks.value];

  for (const block of parsed.preservedBlocks ?? []) {
    const raw = block.raw.trim();
    if (!raw || existingRawSet.has(raw)) continue;
    if (block.kind === 'site') {
      const firstDomain = block.title.split(/[\s,]+/)[0];
      if (firstDomain && siteDomains.has(firstDomain)) continue;
    }
    nextBlocks.push(block);
    existingRawSet.add(raw);
  }

  preservedBlocks.value = nextBlocks;
}

async function fetchServers() {
  loadingServers.value = true;
  try {
    servers.value = await getCaddyServerListApi();
    if (servers.value.length > 0 && !selectedServerId.value) {
      selectedServerId.value = Number(servers.value[0]?.id);
    }
  } catch {
    message.error('获取服务器列表失败');
  } finally {
    loadingServers.value = false;
  }
}

async function fetchConfig() {
  if (!selectedServerId.value) return;
  loadingConfig.value = true;
  try {
    const data = await getCaddyConfigApi(selectedServerId.value);
    configContent.value = data?.config ?? '';
    formModel.value = createEmptyFormModel();
    preservedBlocks.value = [];
    quickSiteDrafts.value = [];
    complexSiteSummaries.value = [];
    activeQuickSiteId.value = undefined;
    structuredAvailable.value = false;

    const loadedModules = loadModules(typeof data?.modules === 'string' ? data.modules : undefined);
    if (configContent.value.trim()) {
      const parsed = parseCaddyfileToBlocks(configContent.value);
      if (loadedModules) {
        mergePreservedBlocksFromRaw(parsed);
      } else if (parsed.sites.length > 0 || parsed.global?.raw || parsed.preservedBlocks.length > 0) {
        applyDraft(parsed);
        return;
      }
    }

    syncQuickStateFromForm(formModel.value);
    mode.value = 'blocks';
    lastEditMode.value = 'blocks';
  } catch {
    message.error('获取配置失败');
  } finally {
    loadingConfig.value = false;
  }
}

function handleServerChange(value: unknown) {
  const next = Number(value);
  if (!Number.isFinite(next)) return;
  selectedServerId.value = next;
}

function openAddServerModal() {
  serverModalType.value = 'add';
  Object.assign(serverForm, {
    id: undefined,
    name: '',
    token: '',
    type: 'local',
    url: 'http://localhost:2019',
  });
  serverModalVisible.value = true;
}

function openEditServerModal() {
  if (!selectedServer.value) return;
  serverModalType.value = 'edit';
  Object.assign(serverForm, {
    id: selectedServer.value.id,
    name: selectedServer.value.name ?? '',
    token: selectedServer.value.token ?? '',
    type: selectedServer.value.type ?? 'local',
    url: selectedServer.value.url ?? '',
  });
  serverModalVisible.value = true;
}

async function saveServer() {
  if (!serverForm.name || !serverForm.url) {
    message.warning('请填写服务器名称和地址');
    return;
  }
  try {
    if (serverModalType.value === 'add') {
      await addCaddyServerApi({
        name: serverForm.name,
        token: serverForm.token,
        type: serverForm.type,
        url: serverForm.url,
      });
      message.success('服务器已添加');
    } else if (serverForm.id) {
      await updateCaddyServerApi(Number(serverForm.id), { ...serverForm });
      message.success('服务器已更新');
    }
    serverModalVisible.value = false;
    await fetchServers();
  } catch {
    message.error('保存服务器失败');
  }
}

function deleteCurrentServer() {
  if (!selectedServerId.value) return;
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除当前 Caddy 服务器吗？',
    async onOk() {
      try {
        await deleteCaddyServerApi(selectedServerId.value!);
        message.success('服务器已删除');
        selectedServerId.value = undefined;
        await fetchServers();
      } catch {
        message.error('删除服务器失败');
      }
    },
  });
}

function addQuickSite() {
  const site = createQuickSiteDraft({
    id: genId(),
    name: `新站点-${quickSiteDrafts.value.length + 1}`,
    domains: [],
    mode: 'reverse_proxy',
    upstream: 'localhost:8080',
  });
  quickSiteDrafts.value.push(site);
  activeQuickSiteId.value = site.id;
}

function duplicateQuickSite(site: QuickSiteDraft) {
  const clone = createQuickSiteDraft({
    ...site,
    domains: [...site.domains],
    id: genId(),
    name: `${site.name || '站点'}-copy`,
  });
  quickSiteDrafts.value.push(clone);
  activeQuickSiteId.value = clone.id;
}

function removeQuickSite(id: string) {
  const idx = quickSiteDrafts.value.findIndex((site) => site.id === id);
  if (idx < 0) return;
  quickSiteDrafts.value.splice(idx, 1);
  if (activeQuickSiteId.value === id) {
    activeQuickSiteId.value = quickSiteDrafts.value[0]?.id;
  }
}

function addUpstream() {
  formModel.value.upstreams.push({
    lbPolicy: 'round_robin',
    name: `upstream-${formModel.value.upstreams.length + 1}`,
    targets: ['localhost:8080'],
  });
}

function removeUpstream(index: number) {
  formModel.value.upstreams.splice(index, 1);
}

function updatePreservedBlock(index: number, raw: string) {
  const block = preservedBlocks.value[index];
  if (!block) return;
  preservedBlocks.value[index] = { ...block, raw };
}

function applyPreset() {
  const site = createQuickSiteDraft({
    id: genId(),
    domains: ['example.com'],
    mode: 'reverse_proxy',
    name: '默认站点',
    upstream: 'localhost:8080',
  });
  formModel.value = {
    schemaVersion: 1,
    global: { raw: '' },
    upstreams: [],
    sites: [],
  };
  preservedBlocks.value = [];
  quickSiteDrafts.value = [site];
  complexSiteSummaries.value = [];
  activeQuickSiteId.value = site.id;
  structuredAvailable.value = true;
  lastEditMode.value = 'blocks';
  mode.value = 'blocks';
}

function handleModeChange(value: unknown) {
  const next = value as CaddyPageMode;
  if (next === mode.value) return;
  if (next === 'waf') {
    mode.value = next;
    return;
  }
  if (next === 'raw') {
    lastEditMode.value = 'raw';
    mode.value = next;
    return;
  }
  if (next === 'blocks') {
    if (lastEditMode.value === 'raw') {
      parseRawToBlocks(false);
    } else {
      syncQuickStateFromForm(formModel.value);
    }
    lastEditMode.value = 'blocks';
    mode.value = next;
    return;
  }
  mode.value = next;
}

async function previewBeforeSave(kind: SavePreviewKind) {
  if (!selectedServerId.value) {
    message.warning('请先选择 Caddy 服务器');
    return;
  }

  const config =
    kind === 'raw' ? configContent.value : generatedConfig.value;
  const modules =
    kind === 'blocks'
      ? JSON.stringify({
          ...mergedQuickFormModel.value,
          preservedBlocks: preservedBlocks.value,
        })
      : '';

  if (kind === 'blocks') {
    const errors = quickValidationErrors.value;
    if (errors.length > 0) {
      message.error(`校验失败：${errors[0]}`);
      return;
    }
  }

  previewing.value = true;
  try {
    const preview = await previewCaddyConfigApi(selectedServerId.value, {
      config,
      mode: kind === 'blocks' ? 'quick' : 'raw',
      modules: modules || undefined,
    });
    if (preview?.valid === false) {
      message.error(`校验失败：${preview.errors?.[0] || 'Caddy 配置不可用'}`);
      return;
    }
    Object.assign(savePreview, {
      actions: preview?.actions ?? [],
      config: preview?.config || config,
      errors: preview?.errors ?? [],
      kind,
      modules,
      open: true,
      original: config,
    });
  } catch (error) {
    message.error(apiErrorMessage(error, '预览配置失败'));
  } finally {
    previewing.value = false;
  }
}

async function confirmSave() {
  if (!selectedServerId.value) return;
  saving.value = true;
  try {
    const content = savePreview.config || savePreview.original;
    await pushCaddyConfigApi(
      selectedServerId.value,
      content,
      savePreview.kind === 'blocks' ? savePreview.modules : undefined,
    );
    message.success('配置已保存并自动热重载 Caddy');
    configContent.value = content;
    if (savePreview.kind === 'blocks' && savePreview.modules) {
      loadModules(savePreview.modules);
      lastEditMode.value = 'blocks';
      mode.value = 'blocks';
    } else {
      structuredAvailable.value = false;
      lastEditMode.value = 'raw';
      mode.value = 'raw';
    }
    savePreview.open = false;
  } catch (error) {
    message.error(apiErrorMessage(error, '保存配置失败'));
  } finally {
    saving.value = false;
  }
}

async function openHistory() {
  if (!selectedServerId.value) return;
  historyDrawerVisible.value = true;
  historyLoading.value = true;
  try {
    const data = await getCaddyConfigHistoryListApi(selectedServerId.value);
    historyList.value = data?.list ?? [];
  } catch {
    message.error('获取历史版本失败');
  } finally {
    historyLoading.value = false;
  }
}

async function openHistoryDetail(id: number) {
  if (!selectedServerId.value) return;
  try {
    historyDetail.value = await getCaddyConfigHistoryDetailApi(selectedServerId.value, id);
    historyDetailVisible.value = true;
  } catch {
    message.error('获取历史配置失败');
  }
}

async function openHistoryCompare(id: number) {
  if (!selectedServerId.value) return;
  try {
    const detail = await getCaddyConfigHistoryDetailApi(selectedServerId.value, id);
    historyCompareLeft.value = detail?.config ?? '';
    historyDiffOnly.value = false;
    historyCompareVisible.value = true;
  } catch {
    message.error('获取历史配置失败');
  }
}

function rollbackHistory(id: number) {
  if (!selectedServerId.value) return;
  Modal.confirm({
    title: '确认回滚',
    content: '确定要回滚到该版本吗？',
    async onOk() {
      try {
        await rollbackCaddyConfigApi(selectedServerId.value!, { historyId: id });
        message.success('回滚成功');
        await fetchConfig();
        await openHistory();
      } catch (error) {
        message.error(apiErrorMessage(error, '回滚失败'));
      }
    },
  });
}

function mbToBytes(value: number) {
  return Math.max(1, Math.round(Number(value || 0))) * 1024 * 1024;
}

function bytesToMB(value: number, fallback: number) {
  if (!value || value <= 0) return fallback;
  return Math.max(1, Math.round(value / 1024 / 1024));
}

function apiErrorMessage(error: unknown, fallback: string) {
  const data = (error as any)?.response?.data ?? (error as any)?.data ?? {};
  const detail = data?.message ?? data?.msg ?? data?.error ?? (error as any)?.message;
  return detail ? `${fallback}：${detail}` : fallback;
}

function syncWafForm(data: Record<string, any>) {
  wafStatus.value = data;
  wafForm.enabled = Boolean(data.enabled);
  wafForm.mode = data.mode === 'off' ? 'detectiononly' : (data.mode || 'detectiononly');
  wafForm.strength = data.strength || 'balanced';
  wafForm.audit = data.audit || 'relevantonly';
  wafForm.requestBodyAccess = data.requestBodyAccess ?? true;
  wafForm.requestBodyLimitMB = bytesToMB(data.requestBodyLimit, 10);
  wafForm.requestBodyNoFilesLimitMB = bytesToMB(data.requestBodyNoFilesLimit, 1);
  wafForm.siteAddresses = data.siteAddresses?.length
    ? [...data.siteAddresses]
    : [...(data.availableSites ?? [])];
}

function buildWafPayload() {
  const enabled = Boolean(wafForm.enabled);
  return {
    audit: wafForm.audit,
    enabled,
    mode: enabled ? wafForm.mode : 'off',
    requestBodyAccess: wafForm.requestBodyAccess,
    requestBodyLimit: mbToBytes(wafForm.requestBodyLimitMB),
    requestBodyNoFilesLimit: mbToBytes(wafForm.requestBodyNoFilesLimitMB),
    serverId: selectedServerId.value,
    siteAddresses: [...wafForm.siteAddresses],
    strength: wafForm.strength,
  };
}

async function fetchWafConfig() {
  if (!selectedServerId.value) return;
  wafLoading.value = true;
  try {
    const data = await getSimpleWafConfigApi(selectedServerId.value);
    syncWafForm(data ?? {});
  } catch {
    message.error('获取防火墙配置失败');
  } finally {
    wafLoading.value = false;
  }
}

async function saveWafConfig() {
  if (!selectedServerId.value) return;
  wafSaving.value = true;
  try {
    await updateSimpleWafConfigApi(buildWafPayload());
    message.success('防火墙设置已保存');
    await fetchWafConfig();
  } catch (error) {
    message.error(apiErrorMessage(error, '保存防火墙设置失败'));
  } finally {
    wafSaving.value = false;
  }
}

async function previewWafConfig() {
  if (!selectedServerId.value) return;
  wafPreviewing.value = true;
  try {
    wafPreviewResult.value = await previewSimpleWafConfigApi(buildWafPayload());
    wafPreviewVisible.value = true;
  } catch (error) {
    message.error(apiErrorMessage(error, '生成防火墙预览失败'));
  } finally {
    wafPreviewing.value = false;
  }
}

async function applyWafConfig() {
  if (!selectedServerId.value) return;
  wafApplying.value = true;
  try {
    const data = await applySimpleWafConfigApi(buildWafPayload());
    message.success(data?.message || '防火墙配置已应用');
    syncWafForm(data ?? {});
    await fetchConfig();
  } catch (error) {
    message.error(apiErrorMessage(error, '应用防火墙配置失败'));
  } finally {
    wafApplying.value = false;
  }
}

watch(selectedServerId, async () => {
  await fetchConfig();
  if (mode.value === 'waf') {
    await fetchWafConfig();
  }
});

watch(mode, (value) => {
  if (value === 'waf' && !wafStatus.value) {
    fetchWafConfig();
  }
});

onMounted(async () => {
  await fetchServers();
  if (selectedServerId.value) {
    await fetchConfig();
  }
});
</script>

<template>
  <Page description="管理 Caddy 服务器、分块配置、WAF 与历史版本。" title="Caddy 配置">
    <div class="caddy-config-page">
      <Card :bordered="false" class="caddy-shell">
        <div class="caddy-toolbar">
          <div class="toolbar-group primary">
            <Select
              :value="selectedServerId"
              :loading="loadingServers"
              :options="serverOptions"
              placeholder="选择服务器"
              class="server-select"
              @change="handleServerChange"
            />
            <Button @click="fetchServers">刷新</Button>
            <Button type="primary" ghost @click="openAddServerModal">添加</Button>
            <Button :disabled="!selectedServerId" @click="openEditServerModal">编辑</Button>
            <Button danger :disabled="!selectedServerId" @click="deleteCurrentServer">删除</Button>
          </div>
          <div class="toolbar-group actions">
            <Button :disabled="!selectedServerId" @click="openHistory">历史</Button>
            <Button @click="applyPreset">默认模板</Button>
            <Button :disabled="!configContent.trim()" @click="parseRawToBlocks(true)">从原始配置解析</Button>
            <Button
              v-if="mode === 'raw'"
              type="primary"
              :loading="previewing"
              :disabled="!selectedServerId"
              @click="previewBeforeSave('raw')"
            >
              保存原始配置
            </Button>
            <Button
              v-else
              type="primary"
              :loading="previewing"
              :disabled="!selectedServerId"
              @click="previewBeforeSave('blocks')"
            >
              保存分块配置
            </Button>
          </div>
        </div>

        <Spin :spinning="loadingConfig">
          <div v-if="servers.length === 0" class="empty-state">
            <Empty description="未找到 Caddy 服务器">
              <Button type="primary" @click="openAddServerModal">添加服务器</Button>
            </Empty>
          </div>
          <template v-else>
            <div class="mode-strip">
              <Radio.Group :value="mode" button-style="solid" @change="(event: any) => handleModeChange(event.target.value)">
                <Radio.Button v-for="item in modeOptions" :key="item.value" :value="item.value">
                  {{ item.label }}
                </Radio.Button>
              </Radio.Group>
              <span class="mode-summary">
                <template v-if="mode === 'blocks'">编辑常用站点能力，复杂配置自动保留。</template>
                <template v-else-if="mode === 'waf'">配置 Coraza / OWASP CRS 常用开关并应用到 Caddy。</template>
                <template v-else-if="mode === 'raw'">直接维护完整 Caddyfile。</template>
                <template v-else>查看当前将要发布的 Caddyfile。</template>
              </span>
            </div>

            <Alert
              v-if="mode === 'blocks' && quickValidationErrors.length"
              class="mb-3"
              :message="quickValidationErrors[0]"
              show-icon
              type="error"
            />

            <div v-if="mode === 'blocks'" class="blocks-layout">
              <aside class="global-column">
                <Collapse
                  :default-active-key="['global', 'global-preserved', 'upstreams']"
                  class="config-collapse"
                >
                  <CollapsePanel key="global" header="全局配置">
                    <div class="panel-subtitle mb-3">Caddyfile 顶层选项，保存时会和右侧站点配置一起合并。</div>
                    <Input.TextArea
                      v-model:value="formModel.global.raw"
                      :auto-size="{ minRows: 14, maxRows: 22 }"
                      class="code-textarea"
                      placeholder="Caddy 全局选项"
                    />
                  </CollapsePanel>

                  <CollapsePanel v-if="globalPreservedBlocks.length" key="global-preserved" header="Snippet / 保留块">
                    <Alert class="mb-3" type="warning" show-icon message="snippet、全局片段和无法归类的配置会原样保留，可在这里直接编辑。" />
                    <Collapse
                      :default-active-key="globalPreservedBlocks.map((item) => item.block.id)"
                      class="inner-collapse"
                    >
                      <CollapsePanel
                        v-for="item in globalPreservedBlocks"
                        :key="item.block.id"
                      >
                        <template #header>
                          <div class="preserved-title">
                            <Tag :color="blockKindColor(item.block.kind)">{{ blockKindLabel(item.block.kind) }}</Tag>
                            <span>{{ item.block.title }}</span>
                          </div>
                        </template>
                        <div class="text-muted mb-2">{{ item.block.reason }}</div>
                        <Input.TextArea
                          :value="item.block.raw"
                          :auto-size="{ minRows: 5, maxRows: 14 }"
                          class="code-textarea"
                          @update:value="(value: string) => updatePreservedBlock(item.index, value)"
                        />
                      </CollapsePanel>
                    </Collapse>
                  </CollapsePanel>

                  <CollapsePanel key="upstreams">
                    <template #header>
                      <div class="panel-header-line">
                        <span>上游池</span>
                        <Button size="small" @click.stop="addUpstream">新增</Button>
                      </div>
                    </template>
                    <div class="panel-subtitle mb-3">复用后端目标和负载策略。</div>
                    <Empty v-if="formModel.upstreams.length === 0" description="暂无上游池" />
                    <div v-for="(upstream, index) in formModel.upstreams" :key="index" class="upstream-row">
                      <Input v-model:value="upstream.name" placeholder="名称" />
                      <Select v-model:value="upstream.lbPolicy" :options="lbPolicyOptions" />
                      <Select v-model:value="upstream.targets" mode="tags" placeholder="目标地址" />
                      <Button danger size="small" @click="removeUpstream(index)">删除</Button>
                    </div>
                  </CollapsePanel>
                </Collapse>
              </aside>

              <main class="sites-column">
                <Collapse
                  :default-active-key="['sites', 'site-editor', 'site-preserved']"
                  class="config-collapse"
                >
                  <CollapsePanel key="sites">
                    <template #header>
                      <div class="panel-header-line">
                        <span>站点</span>
                        <Button size="small" type="primary" @click.stop="addQuickSite">新建站点</Button>
                      </div>
                    </template>
                    <div class="panel-subtitle mb-3">右侧只维护站点能力；global 和 snippet 在左侧。</div>
                    <div v-if="quickSiteDrafts.length === 0" class="sidebar-empty">
                      <Empty description="暂无简单站点">
                        <Button size="small" type="primary" @click="addQuickSite">新建站点</Button>
                      </Empty>
                    </div>
                    <div v-else class="site-list">
                      <button
                        v-for="site in quickSiteDrafts"
                        :key="site.id"
                        type="button"
                        class="site-item"
                        :class="{ active: site.id === activeQuickSiteId }"
                        @click="activeQuickSiteId = site.id"
                      >
                        <span class="site-name">{{ site.name || '未命名站点' }}</span>
                        <span class="site-domain">{{ site.domains[0] || '未配置域名' }}</span>
                        <Tag class="site-state" :color="site.enabled ? 'green' : 'default'">
                          {{ site.enabled ? '启用' : '停用' }}
                        </Tag>
                      </button>
                    </div>

                    <Alert
                      v-if="complexSiteSummaries.length"
                      class="mt-3"
                      show-icon
                      type="warning"
                      :message="`检测到 ${complexSiteSummaries.length} 个复杂站点，已在右侧保留站点中原样保留。`"
                    />
                  </CollapsePanel>

                  <CollapsePanel key="site-editor">
                    <template #header>
                      <div class="panel-header-line">
                        <span>站点配置</span>
                        <Space v-if="activeQuickSite" @click.stop>
                          <Button size="small" @click="duplicateQuickSite(activeQuickSite)">复制</Button>
                          <Popconfirm title="确认删除该站点？" @confirm="removeQuickSite(activeQuickSite.id)">
                            <Button size="small" danger>删除</Button>
                          </Popconfirm>
                        </Space>
                      </div>
                    </template>
                    <div class="panel-subtitle mb-3">域名、站点类型、TLS 和目标地址。</div>

                    <Empty v-if="!activeQuickSite" description="请选择或新建一个站点" />
                    <Form v-else layout="vertical">
                      <Row :gutter="16">
                        <Col :xs="24" :md="12">
                          <Form.Item label="站点名称">
                            <Input v-model:value="activeQuickSite.name" placeholder="例如：官网反代" />
                          </Form.Item>
                        </Col>
                        <Col :xs="24" :md="12">
                          <Form.Item label="启用状态">
                            <Switch v-model:checked="activeQuickSite.enabled" />
                          </Form.Item>
                        </Col>
                        <Col :xs="24">
                          <Form.Item label="域名 / 监听地址">
                            <Select
                              v-model:value="activeQuickSite.domains"
                              mode="tags"
                              placeholder="example.com 或 :8080"
                            />
                          </Form.Item>
                        </Col>
                        <Col :xs="24" :md="12">
                          <Form.Item label="站点类型">
                            <Select v-model:value="activeQuickSite.mode" :options="quickModeOptions" />
                          </Form.Item>
                        </Col>
                        <Col :xs="24" :md="12">
                          <Form.Item label="TLS">
                            <Select v-model:value="activeQuickSite.tlsMode" :options="tlsModeOptions" />
                          </Form.Item>
                        </Col>
                        <Col v-if="activeQuickSite.mode === 'reverse_proxy'" :xs="24">
                          <Form.Item label="代理目标">
                            <Input v-model:value="activeQuickSite.upstream" placeholder="127.0.0.1:8080 或 https://backend.internal" />
                          </Form.Item>
                        </Col>
                        <template v-if="activeQuickSite.mode === 'file_server'">
                          <Col :xs="24" :md="18">
                            <Form.Item label="站点根目录">
                              <Input v-model:value="activeQuickSite.root" placeholder="/srv/www/site" />
                            </Form.Item>
                          </Col>
                          <Col :xs="24" :md="6">
                            <Form.Item label="目录浏览">
                              <Switch v-model:checked="activeQuickSite.browse" />
                            </Form.Item>
                          </Col>
                        </template>
                        <template v-if="activeQuickSite.mode === 'redirect'">
                          <Col :xs="24" :md="18">
                            <Form.Item label="跳转地址">
                              <Input v-model:value="activeQuickSite.redirectTo" placeholder="https://example.com" />
                            </Form.Item>
                          </Col>
                          <Col :xs="24" :md="6">
                            <Form.Item label="状态码">
                              <InputNumber v-model:value="activeQuickSite.redirectCode" :min="300" :max="399" class="w-full" />
                            </Form.Item>
                          </Col>
                        </template>
                      </Row>
                    </Form>
                  </CollapsePanel>

                  <CollapsePanel v-if="preservedSiteBlocks.length" key="site-preserved" header="保留站点">
                    <Alert class="mb-3" type="warning" show-icon message="复杂站点无法结构化编辑，会跟随分块配置一起保存。" />
                    <Collapse
                      :default-active-key="preservedSiteBlocks.map((item) => item.block.id)"
                      class="inner-collapse"
                    >
                      <CollapsePanel
                        v-for="item in preservedSiteBlocks"
                        :key="item.block.id"
                      >
                        <template #header>
                          <div class="preserved-title">
                            <Tag color="orange">site</Tag>
                            <span>{{ item.block.title }}</span>
                          </div>
                        </template>
                        <div class="text-muted mb-2">{{ item.block.reason }}</div>
                        <Input.TextArea
                          :value="item.block.raw"
                          :auto-size="{ minRows: 5, maxRows: 14 }"
                          class="code-textarea"
                          @update:value="(value: string) => updatePreservedBlock(item.index, value)"
                        />
                      </CollapsePanel>
                    </Collapse>
                  </CollapsePanel>
                </Collapse>
              </main>
            </div>

            <div v-else-if="mode === 'waf'" class="waf-pane">
              <Spin :spinning="wafLoading">
                <Card title="简易 WAF">
                  <template #extra>
                    <Tag :color="wafStatusColor">{{ wafStatusText }}</Tag>
                  </template>
                  <Form layout="vertical">
                    <Row :gutter="16">
                      <Col :xs="24" :md="8">
                        <Form.Item label="启用 WAF">
                          <Switch v-model:checked="wafForm.enabled" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24" :md="8">
                        <Form.Item label="引擎模式">
                          <Select v-model:value="wafForm.mode" :disabled="!wafForm.enabled" :options="wafModeOptions" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24" :md="8">
                        <Form.Item label="规则强度">
                          <Select v-model:value="wafForm.strength" :disabled="!wafForm.enabled" :options="wafStrengthOptions" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24" :md="8">
                        <Form.Item label="审计日志">
                          <Select v-model:value="wafForm.audit" :disabled="!wafForm.enabled" :options="wafAuditOptions" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24" :md="8">
                        <Form.Item label="请求体检测">
                          <Switch v-model:checked="wafForm.requestBodyAccess" :disabled="!wafForm.enabled" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24" :md="4">
                        <Form.Item label="请求体 MB">
                          <InputNumber v-model:value="wafForm.requestBodyLimitMB" :min="1" :disabled="!wafForm.enabled" class="w-full" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24" :md="4">
                        <Form.Item label="无文件 MB">
                          <InputNumber v-model:value="wafForm.requestBodyNoFilesLimitMB" :min="1" :disabled="!wafForm.enabled" class="w-full" />
                        </Form.Item>
                      </Col>
                      <Col :xs="24">
                        <Form.Item label="应用站点">
                          <Select
                            v-model:value="wafForm.siteAddresses"
                            :options="wafAvailableSites.map((item: string) => ({ label: item, value: item }))"
                            :disabled="!wafForm.enabled"
                            mode="multiple"
                            placeholder="默认应用到所有可用站点"
                          />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Space>
                      <Button :loading="wafSaving" @click="saveWafConfig">保存设置</Button>
                      <Button :loading="wafPreviewing" @click="previewWafConfig">预览变更</Button>
                      <Button type="primary" :loading="wafApplying" @click="applyWafConfig">应用到 Caddy</Button>
                      <Button @click="fetchWafConfig">刷新</Button>
                    </Space>
                  </Form>
                </Card>
              </Spin>
            </div>

            <div v-else-if="mode === 'raw'" class="raw-pane">
              <Input.TextArea
                v-model:value="configContent"
                :auto-size="{ minRows: 24 }"
                class="code-textarea"
                placeholder="完整 Caddyfile"
              />
            </div>

            <div v-else class="preview-pane">
              <Input.TextArea :value="previewConfig" :auto-size="{ minRows: 24 }" class="code-textarea" readonly />
            </div>
          </template>
        </Spin>
      </Card>
    </div>

    <Modal
      v-model:open="serverModalVisible"
      :title="serverModalType === 'add' ? '添加服务器' : '编辑服务器'"
      @ok="saveServer"
    >
      <Form layout="vertical">
        <Form.Item label="名称">
          <Input v-model:value="serverForm.name" placeholder="服务器名称" />
        </Form.Item>
        <Form.Item label="地址">
          <Input v-model:value="serverForm.url" placeholder="http://localhost:2019" />
        </Form.Item>
        <Form.Item label="类型">
          <Radio.Group v-model:value="serverForm.type">
            <Radio.Button value="local">本地</Radio.Button>
            <Radio.Button value="remote">远程</Radio.Button>
          </Radio.Group>
        </Form.Item>
        <Form.Item v-if="serverForm.type === 'remote'" label="凭证">
          <Input.Password v-model:value="serverForm.token" placeholder="可选认证凭证" />
        </Form.Item>
      </Form>
    </Modal>

    <Modal
      v-model:open="savePreview.open"
      width="900px"
      title="保存预览"
      :confirm-loading="saving"
      @ok="confirmSave"
    >
      <Space wrap class="mb-3">
        <Tag color="blue">{{ savePreview.kind === 'raw' ? '原始配置' : '分块配置' }}</Tag>
        <Tag v-for="item in savePreview.actions" :key="item">{{ item }}</Tag>
      </Space>
      <Alert v-if="savePreview.errors.length" class="mb-3" type="error" :message="savePreview.errors[0]" show-icon />
      <Input.TextArea :value="formatCaddyfile(savePreview.config)" :auto-size="{ minRows: 18, maxRows: 28 }" class="code-textarea" readonly />
    </Modal>

    <Drawer v-model:open="historyDrawerVisible" title="配置历史" width="720">
      <Spin :spinning="historyLoading">
        <List :data-source="historyList">
          <template #renderItem="{ item }">
            <List.Item>
              <List.Item.Meta>
                <template #title>
                  <Space>
                    <span>{{ item.createdAt ?? item.timestamp ?? `Version #${item.id}` }}</span>
                    <Tag :color="item.action === 'rollback' ? 'orange' : 'blue'">
                      {{ item.action === 'rollback' ? '回滚' : '更新' }}
                    </Tag>
                  </Space>
                </template>
                <template #description>
                  {{ historyDescription(item) }}
                </template>
              </List.Item.Meta>
              <template #actions>
                <Button size="small" type="link" @click="openHistoryDetail(item.id)">查看</Button>
                <Button size="small" type="link" @click="openHistoryCompare(item.id)">对比</Button>
                <Button size="small" type="link" @click="rollbackHistory(item.id)">回滚</Button>
              </template>
            </List.Item>
          </template>
        </List>
      </Spin>
    </Drawer>

    <Modal v-model:open="historyDetailVisible" width="900px" title="历史配置" :footer="null">
      <Descriptions v-if="historyDetail" bordered size="small" class="mb-3">
        <DescriptionsItem label="时间">{{ historyDetail.createdAt }}</DescriptionsItem>
        <DescriptionsItem label="动作">{{ historyDetail.action }}</DescriptionsItem>
        <DescriptionsItem label="Hash">{{ historyDetail.hash }}</DescriptionsItem>
      </Descriptions>
      <Input.TextArea :value="formatCaddyfile(historyDetail?.config || '')" :auto-size="{ minRows: 20, maxRows: 30 }" class="code-textarea" readonly />
    </Modal>

    <Modal v-model:open="historyCompareVisible" width="1100px" title="历史对比" :footer="null">
      <div class="diff-toolbar">
        <span>左侧历史版本，右侧当前预览</span>
        <Switch v-model:checked="historyDiffOnly" checked-children="仅差异" un-checked-children="全部" />
      </div>
      <div class="diff-grid">
        <div class="diff-column">
          <div v-for="row in historyCompareRows" :key="`l-${row.key}`" class="diff-line-row" :class="diffSideClass(row, 'left')">
            <span class="diff-no">{{ row.leftNo ?? '' }}</span>
            <span class="diff-line">{{ row.left ?? '' }}</span>
          </div>
        </div>
        <div class="diff-column">
          <div v-for="row in historyCompareRows" :key="`r-${row.key}`" class="diff-line-row" :class="diffSideClass(row, 'right')">
            <span class="diff-no">{{ row.rightNo ?? '' }}</span>
            <span class="diff-line">{{ row.right ?? '' }}</span>
          </div>
        </div>
      </div>
    </Modal>

    <Drawer v-model:open="wafPreviewVisible" title="WAF 变更预览" width="720">
      <Descriptions v-if="wafPreviewResult" bordered size="small" class="mb-3">
        <DescriptionsItem label="状态">{{ wafPreviewResult.message || '已生成预览' }}</DescriptionsItem>
        <DescriptionsItem label="Coraza">{{ wafPreviewResult.corazaVersion || '-' }}</DescriptionsItem>
        <DescriptionsItem label="CRS">{{ wafPreviewResult.crsVersion || '-' }}</DescriptionsItem>
      </Descriptions>
      <Space v-if="wafPreviewResult?.actions?.length" wrap class="mb-3">
        <Tag v-for="item in wafPreviewResult.actions" :key="item">{{ item }}</Tag>
      </Space>
      <Input.TextArea
        :value="wafPreviewResult?.directives || wafPreviewResult?.config || ''"
        :auto-size="{ minRows: 18, maxRows: 28 }"
        class="code-textarea"
        readonly
      />
    </Drawer>
  </Page>
</template>

<style scoped>
.caddy-config-page {
  padding: 16px;
}

.caddy-shell {
  min-height: calc(100vh - 140px);
}

.caddy-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px solid #eef0f4;
}

.toolbar-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.toolbar-group.actions {
  justify-content: flex-end;
}

.server-select {
  min-width: 260px;
}

.mode-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding: 10px 12px;
  border: 1px solid #eef0f4;
  border-radius: 8px;
  background: #fbfcfd;
}

.diff-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.mode-summary,
.text-muted {
  color: #667085;
  font-size: 12px;
}

.blocks-layout {
  display: grid;
  grid-template-columns: minmax(320px, 0.9fr) minmax(0, 1.45fr);
  gap: 16px;
  align-items: start;
}

.global-column,
.sites-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.config-collapse {
  border: 1px solid #eef0f4;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.config-collapse :deep(.ant-collapse-item) {
  border-bottom-color: #eef0f4;
}

.config-collapse :deep(.ant-collapse-header) {
  align-items: center;
  padding: 12px 16px;
  color: #1f2937;
  font-weight: 600;
}

.config-collapse :deep(.ant-collapse-content-box) {
  padding: 14px 16px 16px;
}

.inner-collapse {
  border-color: #eef0f4;
  background: #fbfcfd;
}

.inner-collapse :deep(.ant-collapse-header) {
  padding: 8px 12px;
  font-weight: 500;
}

.inner-collapse :deep(.ant-collapse-content-box) {
  padding: 10px 12px 12px;
}

.panel-header-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}

.panel-subtitle {
  margin-top: 2px;
  color: #667085;
  font-size: 12px;
}

.preserved-title {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.preserved-title span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-empty {
  padding: 18px 0;
}

.site-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.site-item {
  position: relative;
  width: 100%;
  min-height: 72px;
  padding: 12px 72px 12px 12px;
  text-align: left;
  cursor: pointer;
  background: #fbfcfd;
  border: 1px solid #eef0f4;
  border-radius: 6px;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    background 0.16s ease;
}

.site-item.active {
  border-color: #1677ff;
  background: #f0f7ff;
  box-shadow: 0 0 0 2px rgb(22 119 255 / 8%);
}

.site-state {
  position: absolute;
  top: 12px;
  right: 12px;
}

.site-name,
.site-domain {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.site-name {
  font-weight: 600;
}

.site-domain {
  margin-top: 4px;
  color: #667085;
  font-size: 12px;
}

.upstream-row {
  display: grid;
  grid-template-columns: minmax(110px, 0.8fr) minmax(120px, 0.7fr) minmax(160px, 1fr) auto;
  gap: 8px;
  margin-bottom: 8px;
}

.empty-state {
  min-height: 420px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.code-textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 13px;
}

.diff-toolbar {
  margin-bottom: 12px;
}

.diff-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
}

.diff-column {
  max-height: 65vh;
  overflow: auto;
}

.diff-column + .diff-column {
  border-left: 1px solid #e5e7eb;
}

.diff-line-row {
  display: grid;
  grid-template-columns: 40px 1fr;
  gap: 8px;
  padding: 5px 8px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  white-space: pre-wrap;
  border-left: 3px solid transparent;
}

.diff-line-row.added {
  color: #166534;
  background: #dcfce7;
  border-left-color: #16a34a;
}

.diff-line-row.removed {
  color: #991b1b;
  background: #fee2e2;
  border-left-color: #b91c1c;
}

.diff-line-row.blank {
  color: transparent;
  background: #f8fafc;
}

.diff-no {
  color: #98a2b3;
  text-align: right;
  user-select: none;
}

.diff-line {
  overflow-wrap: anywhere;
}

@media (max-width: 960px) {
  .caddy-toolbar,
  .blocks-layout {
    grid-template-columns: 1fr;
  }

  .toolbar-group.actions {
    justify-content: flex-start;
  }

  .upstream-row {
    grid-template-columns: 1fr;
  }
}
</style>
