<script setup lang="ts">
import { computed, defineAsyncComponent, h, onMounted, ref, watch } from 'vue';
import { NButton, NTag, useDialog, useMessage } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { loader } from '@guolao/vue-monaco-editor';
import {
  addCaddyServer,
  deleteCaddyServer,
  fetchCaddyConfig,
  fetchCaddyConfigHistory,
  fetchCaddyConfigHistoryDetail,
  fetchCaddyServers,
  previewCaddyConfig,
  rollbackCaddyConfig,
  updateCaddyConfigRaw,
  updateCaddyConfigStructured,
  updateCaddyServer
} from '@/service/api/caddy';
import {
  type WafIntegrationStatusResp,
  applyWafIntegration,
  fetchWafIntegrationStatus
} from '@/service/api/caddy-integration';
import SvgIcon from '@/components/custom/svg-icon.vue';
import ConfigPreviewPanel from './components/ConfigPreviewPanel.vue';
import QuickConfigPanel from './components/QuickConfigPanel.vue';
import RawEditorPanel from './components/RawEditorPanel.vue';
import SimpleWafPanel from './components/SimpleWafPanel.vue';
import WafIntegrationCard from './components/WafIntegrationCard.vue';
import type { CaddyFormModel, Route, RouteMatch, Site } from './types';
import {
  type DiffRow,
  buildCaddyfile,
  buildLineDiff,
  formatCaddyfile,
  genId,
  normalizeModules,
  parseCaddyfileToModules,
  validateStructuredConfig
} from './caddy-config-utils';
import {
  type ComplexSiteSummary,
  type QuickSiteDraft,
  buildQuickConfigState,
  createQuickSiteDraft,
  mergeQuickConfigDrafts
} from './quick-config-utils';

const VueMonacoEditor = defineAsyncComponent(() => import('@guolao/vue-monaco-editor').then(m => m.VueMonacoEditor));
const VueMonacoDiffEditor = defineAsyncComponent(() =>
  import('@guolao/vue-monaco-editor').then(m => m.VueMonacoDiffEditor)
);

// Configure Monaco Editor loader to use npmmirror for better performance in China
loader.config({
  paths: {
    vs: 'https://registry.npmmirror.com/monaco-editor/0.44.0/files/min/vs'
  }
});

// Defines
interface CaddyServer {
  id: number;
  name: string;
  url: string;
  type: string;
  token?: string;
}

interface CaddyConfigHistoryItem {
  id: number;
  serverId: number;
  action: string;
  hash: string;
  createdAt: string;
}

const message = useMessage();
const dialog = useDialog();
const loading = ref(false);
const saving = ref(false);
const servers = ref<CaddyServer[]>([]);
const currentServerId = ref<number | null>(null);
const pageMode = ref<'quick' | 'waf' | 'raw' | 'preview'>('quick');
const lastEditMode = ref<'quick' | 'raw'>('quick');
const configContent = ref('');
const showSettingsDrawer = ref(false);
const structuredAvailable = ref(false);
const createEmptyFormModel = (): CaddyFormModel => ({
  schemaVersion: 1,
  global: { raw: '' },
  upstreams: [],
  sites: []
});
const formModel = ref<CaddyFormModel>(createEmptyFormModel());
const quickSiteDrafts = ref<QuickSiteDraft[]>([]);
const complexSiteSummaries = ref<ComplexSiteSummary[]>([]);
const activeQuickSiteId = ref<string | null>(null);

const showHistoryModal = ref(false);
const historyLoading = ref(false);
const historyList = ref<CaddyConfigHistoryItem[]>([]);
const historyPagination = ref({ page: 1, pageSize: 10, itemCount: 0 });
const showHistoryDetailModal = ref(false);
const showHistoryCompareModal = ref(false);
const historyDetail = ref<{
  id: number;
  createdAt: string;
  action: string;
  hash: string;
  config: string;
} | null>(null);
const historyCompareLeft = ref('');
const historyDiffOnly = ref(false);
const showGlobalCompareModal = ref(false);
const initialGlobalRaw = ref('');
const showGlobalDiffOnly = ref(false);
const diffLeftRef = ref<HTMLElement | null>(null);
const diffRightRef = ref<HTMLElement | null>(null);
let diffSyncing = false;

// Server Management Modal
const showServerModal = ref(false);
const serverModalType = ref<'add' | 'edit'>('add');
const serverFormModel = ref<Omit<CaddyServer, 'id'> & { id?: number }>({
  name: '',
  url: '',
  type: 'local',
  token: ''
});
const wafIntegrationLoading = ref(false);
const wafIntegrationSubmitting = ref(false);
const wafIntegrationPreviewing = ref(false);
const wafIntegrationUnavailable = ref(false);
const wafIntegrationStatus = ref<WafIntegrationStatusResp | null>(null);
const selectedWafIntegrationSites = ref<string[]>([]);
const wafIntegrationPreviewActions = ref<string[]>([]);
const savePreviewVisible = ref(false);
const savePreviewKind = ref<'quick' | 'raw'>('quick');
const savePreviewConfig = ref('');
const savePreviewActions = ref<string[]>([]);
const savePreviewErrors = ref<string[]>([]);
const savePreviewModules = ref('');
const savePreviewOriginal = ref('');

// Computed
const serverOptions = computed(() => servers.value.map(s => ({ label: s.name, value: s.id })));
const structuredReady = computed(() => {
  if (structuredAvailable.value) return true;
  const model = formModel.value;
  if (model.sites?.length) return true;
  if (model.upstreams?.length) return true;
  return Boolean(model.global?.raw?.trim());
});
const mergedQuickFormModel = computed(() => mergeQuickConfigDrafts(formModel.value, quickSiteDrafts.value));
const generatedQuickCaddyfile = computed(() => buildCaddyfile(mergedQuickFormModel.value));
const effectiveConfigContent = computed(() =>
  lastEditMode.value === 'raw' ? configContent.value : generatedQuickCaddyfile.value
);
const formattedConfigContent = computed(() => formatCaddyfile(effectiveConfigContent.value));
const globalRawChanged = computed(
  () => (formModel.value.global?.raw ?? '').trim() !== (initialGlobalRaw.value ?? '').trim()
);
const globalDiffRows = computed<DiffRow[]>(() => {
  const rows = buildLineDiff(initialGlobalRaw.value ?? '', formModel.value.global?.raw ?? '');
  if (!showGlobalDiffOnly.value) return rows;
  return rows.filter(row => row.type !== 'same');
});
const quickValidationErrors = computed(() => {
  if (!structuredReady.value && configContent.value.trim()) {
    return [];
  }
  return validateStructuredConfig(mergedQuickFormModel.value);
});
const historyDetailFormattedConfig = computed(() =>
  historyDetail.value ? formatCaddyfile(historyDetail.value.config) : ''
);
const historyCompareLeftFormatted = computed(() => formatCaddyfile(historyCompareLeft.value));
const historyCompareRight = computed(() => formattedConfigContent.value);
const pageModeOptions = [
  { label: '快速配置', value: 'quick' },
  { label: '防火墙', value: 'waf' },
  { label: '原始配置', value: 'raw' },
  { label: '预览', value: 'preview' }
] as const;
const pageModeSummary = computed(() => {
  if (pageMode.value === 'quick') return '只编辑常用站点能力，复杂配置自动保留。';
  if (pageMode.value === 'waf') return '配置 Coraza / OWASP CRS 的常用开关。';
  if (pageMode.value === 'raw') return '直接维护完整 Caddyfile，适合高级规则。';
  return lastEditMode.value === 'raw' ? '展示当前原始配置内容。' : '展示当前快速配置生成结果。';
});
const moreOptions = computed(() => [
  { label: '更多设置', key: 'settings' },
  { type: 'divider', key: 'divider-1' },
  { label: '添加服务器', key: 'server:add' },
  { label: '编辑当前服务器', key: 'server:edit', disabled: !currentServerId.value },
  { label: '删除当前服务器', key: 'server:delete', disabled: !currentServerId.value },
  { type: 'divider', key: 'divider-2' },
  { label: '查看历史版本', key: 'history', disabled: !currentServerId.value },
  { label: '应用默认模板', key: 'preset' },
  { label: '从原始配置解析', key: 'import-raw' }
]);

// Methods
function syncQuickStateFromForm(model: CaddyFormModel) {
  const { simpleSites, complexSites } = buildQuickConfigState(model);
  const nextActiveId = simpleSites.some(item => item.id === activeQuickSiteId.value)
    ? activeQuickSiteId.value
    : simpleSites[0]?.id || null;

  quickSiteDrafts.value = simpleSites;
  complexSiteSummaries.value = complexSites;
  activeQuickSiteId.value = nextActiveId;
}

async function getServers() {
  const { data, error } = await fetchCaddyServers();
  if (error) {
    message.error('获取服务器列表失败');
    return;
  }
  if (data?.list) {
    servers.value = data.list;
    // 自动选择第一个
    if (servers.value.length > 0) {
      if (!currentServerId.value || !servers.value.find(s => s.id === currentServerId.value)) {
        currentServerId.value = servers.value[0].id;
      }
    } else {
      currentServerId.value = null;
      configContent.value = '';
      formModel.value = createEmptyFormModel();
      syncQuickStateFromForm(formModel.value);
    }
  }
}

async function getConfig() {
  if (!currentServerId.value) return;

  loading.value = true;
  const { data, error } = await fetchCaddyConfig(currentServerId.value);
  loading.value = false;

  if (error) {
    message.error('获取配置失败');
    return;
  }
  if (data) {
    configContent.value = data.config || '';
    structuredAvailable.value = false;
    formModel.value = createEmptyFormModel();
    activeQuickSiteId.value = null;
    if (data.modules) {
      try {
        const parsed = JSON.parse(data.modules);
        if (parsed?.sites || parsed?.global) {
          formModel.value = normalizeModules(parsed);
          structuredAvailable.value = true;
        }
      } catch {
        message.warning('结构化配置解析失败，已忽略');
        formModel.value = createEmptyFormModel();
        structuredAvailable.value = false;
      }
    }
    initialGlobalRaw.value = formModel.value.global?.raw ?? '';
    if (!structuredAvailable.value && configContent.value.trim()) {
      ensureStructuredForEdit(true);
    } else {
      syncQuickStateFromForm(formModel.value);
    }
    pageMode.value = 'quick';
    lastEditMode.value = 'quick';
  }
}

function syncSelectedWafIntegrationSites(status: WafIntegrationStatusResp | null) {
  const available = Array.isArray(status?.availableSites) ? status.availableSites : [];
  const imported = Array.isArray(status?.importedSites) ? status.importedSites : [];

  if (imported.length > 0) {
    selectedWafIntegrationSites.value = imported.filter(item => available.includes(item));
    return;
  }

  const preserved = selectedWafIntegrationSites.value.filter(item => available.includes(item));
  if (preserved.length > 0) {
    selectedWafIntegrationSites.value = preserved;
    return;
  }

  selectedWafIntegrationSites.value = [...available];
}

async function fetchWafIntegrationState() {
  if (!currentServerId.value || wafIntegrationUnavailable.value) {
    return;
  }

  wafIntegrationLoading.value = true;
  try {
    const { data, error } = await fetchWafIntegrationStatus();
    if (!error && data) {
      if (data.serverId && data.serverId !== currentServerId.value) {
        wafIntegrationStatus.value = null;
        selectedWafIntegrationSites.value = [];
        return;
      }
      wafIntegrationStatus.value = data;
      wafIntegrationUnavailable.value = false;
      syncSelectedWafIntegrationSites(data);
      return;
    }

    if (error) {
      const status = Number((error as any)?.response?.status || 0);
      if (status === 404 || status === 405) {
        wafIntegrationUnavailable.value = true;
      }
    }
  } finally {
    wafIntegrationLoading.value = false;
  }
}

function handleRefreshWafIntegrationState() {
  fetchWafIntegrationState();
}

function handleWafIntegrationSiteChange(value: Array<string | number>) {
  selectedWafIntegrationSites.value = value.map(item => String(item));
}

async function submitWafIntegration(enabled: boolean, dryRun: boolean) {
  if (!currentServerId.value) {
    message.warning('请先选择 Caddy 服务器');
    return;
  }
  if (wafIntegrationUnavailable.value) {
    message.warning('当前接入开关接口暂不可用');
    return;
  }

  const siteAddresses = selectedWafIntegrationSites.value.filter(item => item.trim());
  if (siteAddresses.length === 0) {
    message.warning('请至少选择一个站点');
    return;
  }

  if (dryRun) {
    wafIntegrationPreviewing.value = true;
  } else {
    wafIntegrationSubmitting.value = true;
  }

  try {
    const { data, error } = await applyWafIntegration({
      serverId: currentServerId.value,
      enabled,
      siteAddresses,
      dryRun
    });
    if (error || !data) {
      const status = Number((error as any)?.response?.status || 0);
      if (status === 404 || status === 405) {
        wafIntegrationUnavailable.value = true;
        message.warning('当前接入开关接口暂不可用');
      }
      return;
    }

    wafIntegrationPreviewActions.value = data.actions || [];
    if (dryRun) {
      message.success(data.message || '已生成 WAF 接入预览');
      return;
    }

    message.success(data.message || (enabled ? 'WAF 接入已应用' : 'WAF 接入已取消'));
    await getConfig();
    await fetchWafIntegrationState();
  } finally {
    wafIntegrationPreviewing.value = false;
    wafIntegrationSubmitting.value = false;
  }
}

function handlePreviewWafIntegration() {
  return submitWafIntegration(true, true);
}

function handleEnableWafIntegration() {
  return submitWafIntegration(true, false);
}

function handleDisableWafIntegration() {
  return submitWafIntegration(false, false);
}

async function saveRawConfig() {
  if (!currentServerId.value) return;

  saving.value = true;
  const { data: preview, error: previewError } = await previewCaddyConfig(currentServerId.value, {
    mode: 'raw',
    config: configContent.value
  });
  if (previewError || !preview) {
    saving.value = false;
    message.error('预览配置失败');
    return;
  }
  if (!preview.valid) {
    saving.value = false;
    message.error(`校验失败：${preview.errors?.[0] || 'Caddy 配置不可用'}`);
    return;
  }

  savePreviewKind.value = 'raw';
  savePreviewConfig.value = preview.config || configContent.value;
  savePreviewActions.value = preview.actions || [];
  savePreviewErrors.value = preview.errors || [];
  savePreviewOriginal.value = configContent.value;
  savePreviewModules.value = '';
  savePreviewVisible.value = true;
  saving.value = false;
}

function applyStructuredParsed(parsed: CaddyFormModel, notify?: boolean) {
  formModel.value = parsed;
  structuredAvailable.value = true;
  initialGlobalRaw.value = parsed.global?.raw ?? '';
  syncQuickStateFromForm(parsed);
  lastEditMode.value = 'quick';
  pageMode.value = 'quick';
  if (notify) message.success('已从原始配置解析');
}

function confirmOverwriteStructured(actionLabel: string, onConfirm: () => void) {
  if (!structuredReady.value) {
    onConfirm();
    return;
  }
  dialog.warning({
    title: '覆盖确认',
    content: `${actionLabel}将覆盖当前结构化配置，未保存内容会丢失，是否继续？`,
    positiveText: '继续',
    negativeText: '取消',
    onPositiveClick: onConfirm
  });
}

function importRawToStructured() {
  if (!configContent.value.trim()) {
    message.error('原始配置为空，无法解析');
    return;
  }
  confirmOverwriteStructured('从原始配置解析', () => {
    const parsed = parseCaddyfileToModules(configContent.value);
    if (parsed.sites.length === 0 && !parsed.global?.raw) {
      message.error('未解析到可用结构化配置');
      return;
    }
    applyStructuredParsed(parsed, true);
  });
}

function ensureStructuredForEdit(force = false) {
  if (!force && structuredReady.value && formModel.value.sites.length > 0) return;
  if (!configContent.value.trim()) return;
  const parsed = parseCaddyfileToModules(configContent.value);
  if (parsed.sites.length === 0 && !parsed.global?.raw) return;
  applyStructuredParsed(parsed, false);
}

async function saveQuickConfig() {
  if (!currentServerId.value) return;

  const nextFormModel = mergedQuickFormModel.value;
  const errors = validateStructuredConfig(nextFormModel);
  if (errors.length > 0) {
    message.error(`校验失败：${errors[0]}`);
    return;
  }

  const content = buildCaddyfile(nextFormModel);
  if (!content) {
    message.error('快速配置为空，无法保存');
    return;
  }

  saving.value = true;
  const modules = JSON.stringify(nextFormModel);
  const { data: preview, error: previewError } = await previewCaddyConfig(currentServerId.value, {
    mode: 'quick',
    config: content,
    modules
  });
  if (previewError || !preview) {
    saving.value = false;
    message.error('预览配置失败');
    return;
  }
  if (!preview.valid) {
    saving.value = false;
    message.error(`校验失败：${preview.errors?.[0] || 'Caddy 配置不可用'}`);
    return;
  }

  savePreviewKind.value = 'quick';
  savePreviewConfig.value = preview.config || content;
  savePreviewActions.value = preview.actions || [];
  savePreviewErrors.value = preview.errors || [];
  savePreviewOriginal.value = content;
  savePreviewModules.value = modules;
  savePreviewVisible.value = true;
  saving.value = false;
}

async function confirmSavePreview() {
  if (!currentServerId.value) return;

  saving.value = true;
  let saved = false;
  try {
    if (savePreviewKind.value === 'raw') {
      const { error } = await updateCaddyConfigRaw(
        currentServerId.value,
        savePreviewConfig.value || savePreviewOriginal.value
      );
      if (error) {
        message.error('保存配置失败');
        return;
      }
      message.success('配置已保存并自动热重载 Caddy');
      configContent.value = savePreviewConfig.value || savePreviewOriginal.value;
      structuredAvailable.value = false;
      lastEditMode.value = 'raw';
      pageMode.value = 'preview';
      saved = true;
      return;
    }

    const { error } = await updateCaddyConfigStructured(
      currentServerId.value,
      savePreviewConfig.value || savePreviewOriginal.value,
      savePreviewModules.value
    );
    if (error) {
      message.error('保存配置失败');
      return;
    }

    message.success('配置已保存并自动热重载 Caddy');
    if (savePreviewModules.value) {
      try {
        formModel.value = normalizeModules(JSON.parse(savePreviewModules.value));
      } catch {
        // 保持原表单状态，避免因为快照格式异常中断保存结果
      }
    }
    configContent.value = savePreviewConfig.value || savePreviewOriginal.value;
    structuredAvailable.value = true;
    initialGlobalRaw.value = formModel.value.global?.raw ?? '';
    syncQuickStateFromForm(formModel.value);
    lastEditMode.value = 'quick';
    pageMode.value = 'preview';
    saved = true;
  } finally {
    saving.value = false;
    if (saved) {
      savePreviewVisible.value = false;
    }
  }
}

function closeSavePreview() {
  if (saving.value) return;
  savePreviewVisible.value = false;
}

function handleModeChange(nextMode: 'quick' | 'waf' | 'raw' | 'preview') {
  if (nextMode === pageMode.value) return;

  if (nextMode === 'waf') {
    pageMode.value = 'waf';
    return;
  }

  if (nextMode === 'raw') {
    configContent.value = lastEditMode.value === 'raw' ? configContent.value : generatedQuickCaddyfile.value;
    lastEditMode.value = 'raw';
    pageMode.value = 'raw';
    return;
  }

  if (nextMode === 'quick') {
    if (lastEditMode.value === 'raw') {
      ensureStructuredForEdit(true);
    } else {
      syncQuickStateFromForm(formModel.value);
    }
    lastEditMode.value = 'quick';
    pageMode.value = 'quick';
    return;
  }

  pageMode.value = 'preview';
}

async function handleSimpleWafApplied() {
  await getConfig();
  await fetchWafIntegrationState();
}

function openSettingsDrawer() {
  showSettingsDrawer.value = true;
}

function handleMoreAction(key: string) {
  if (key === 'settings') {
    openSettingsDrawer();
    return;
  }
  if (key === 'server:add') {
    openAddServerModal();
    return;
  }
  if (key === 'server:edit') {
    openEditServerModal();
    return;
  }
  if (key === 'server:delete') {
    void handleDeleteServer();
    return;
  }
  if (key === 'history') {
    void openHistoryModal();
    return;
  }
  if (key === 'preset') {
    applyPreset();
    return;
  }
  if (key === 'import-raw') {
    importRawToStructured();
  }
}

function applyPreset() {
  confirmOverwriteStructured('应用默认模板', () => {
    structuredAvailable.value = true;
    formModel.value.schemaVersion = 1;
    formModel.value.upstreams = [];
    const siteId = genId();
    formModel.value.sites = [
      {
        id: siteId,
        name: '默认站点',
        enabled: true,
        domains: ['example.com'],
        imports: [],
        geoip2Vars: [],
        encode: [],
        tls: { mode: 'auto' },
        routes: [
          {
            id: genId(),
            name: '默认路由',
            enabled: true,
            match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
            logAppend: [],
            handles: [
              {
                id: genId(),
                type: 'reverse_proxy',
                enabled: true,
                upstream: 'localhost:8080',
                lbPolicy: 'round_robin',
                tlsInsecureSkipVerify: false
              }
            ]
          }
        ]
      }
    ];
    initialGlobalRaw.value = formModel.value.global?.raw ?? '';
    syncQuickStateFromForm(formModel.value);
    activeQuickSiteId.value = siteId;
    lastEditMode.value = 'quick';
    pageMode.value = 'quick';
  });
}

function addQuickSite() {
  const draft = createQuickSiteDraft({
    id: genId(),
    name: `新站点-${quickSiteDrafts.value.length + 1}`,
    domains: [],
    mode: 'reverse_proxy',
    upstream: 'localhost:8080'
  });
  quickSiteDrafts.value.push(draft);
  activeQuickSiteId.value = draft.id;
  lastEditMode.value = 'quick';
  pageMode.value = 'quick';
}

function duplicateQuickSite(id: string) {
  const target = quickSiteDrafts.value.find(item => item.id === id);
  if (!target) return;
  const clone = createQuickSiteDraft({
    ...target,
    domains: [...target.domains],
    id: genId(),
    name: `${target.name || '站点'}-copy`
  });
  quickSiteDrafts.value.push(clone);
  activeQuickSiteId.value = clone.id;
}

function removeQuickSite(id: string) {
  const idx = quickSiteDrafts.value.findIndex(item => item.id === id);
  if (idx < 0) return;
  quickSiteDrafts.value.splice(idx, 1);
  if (activeQuickSiteId.value === id) {
    activeQuickSiteId.value = quickSiteDrafts.value[0]?.id || null;
  }
}

function switchToRawFromQuick() {
  handleModeChange('raw');
}

function openGlobalCompare() {
  showGlobalCompareModal.value = true;
}

// Server Management Methods
function openAddServerModal() {
  serverModalType.value = 'add';
  serverFormModel.value = { name: '', url: 'http://localhost:2019', type: 'local', token: '' };
  showServerModal.value = true;
}

function openEditServerModal() {
  const server = servers.value.find(s => s.id === currentServerId.value);
  if (!server) return;
  serverModalType.value = 'edit';
  serverFormModel.value = { ...server };
  showServerModal.value = true;
}

async function handleDeleteServer() {
  if (!currentServerId.value) return;

  dialog.warning({
    title: '确认删除',
    content: '确定要删除此服务器吗？',
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: async () => {
      const { error } = await deleteCaddyServer(currentServerId.value!);
      if (error) {
        message.error('删除服务器失败');
        return;
      }
      message.success('服务器已删除');
      await getServers();
    }
  });
}

async function handleSaveServer() {
  let error;
  if (serverModalType.value === 'add') {
    const res = await addCaddyServer(serverFormModel.value);
    error = res.error;
  } else {
    const res = await updateCaddyServer(serverFormModel.value as CaddyServer);
    error = res.error;
  }

  if (error) {
    message.error('保存服务器失败');
    return;
  }
  message.success(serverModalType.value === 'add' ? '添加成功' : '更新成功');
  showServerModal.value = false;
  await getServers();
}

function restoreGlobalRaw() {
  dialog.warning({
    title: '恢复确认',
    content: '确定将全局配置恢复为已保存版本吗？',
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: () => {
      formModel.value.global.raw = initialGlobalRaw.value ?? '';
    }
  });
}

function syncDiffScroll(side: 'left' | 'right') {
  if (diffSyncing) return;
  const source = side === 'left' ? diffLeftRef.value : diffRightRef.value;
  const target = side === 'left' ? diffRightRef.value : diffLeftRef.value;
  if (!source || !target) return;
  diffSyncing = true;
  target.scrollTop = source.scrollTop;
  target.scrollLeft = source.scrollLeft;
  requestAnimationFrame(() => {
    diffSyncing = false;
  });
}

const historyColumns: DataTableColumns<CaddyConfigHistoryItem> = [
  {
    title: '时间',
    key: 'createdAt',
    width: 180
  },
  {
    title: '动作',
    key: 'action',
    width: 100,
    render(row) {
      const label = formatHistoryAction(row.action);
      const type = row.action === 'rollback' ? 'warning' : 'info';
      return h(
        NTag,
        {
          type,
          size: 'small'
        },
        { default: () => label }
      );
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 220,
    render(row) {
      return h(
        'div',
        {
          class: 'flex gap-2'
        },
        [
          h(
            NButton,
            {
              size: 'tiny',
              onClick: () => openHistoryDetail(row.id)
            },
            { default: () => '查看' }
          ),
          h(
            NButton,
            {
              size: 'tiny',
              onClick: () => openHistoryCompare(row.id)
            },
            { default: () => '对比' }
          ),
          h(
            NButton,
            {
              size: 'tiny',
              type: 'primary',
              onClick: () => handleRollback(row.id)
            },
            { default: () => '回滚' }
          )
        ]
      );
    }
  }
];

function formatHistoryAction(action: string) {
  return action === 'rollback' ? '回滚' : '更新';
}

async function openHistoryModal() {
  if (!currentServerId.value) return;
  showHistoryModal.value = true;
  historyPagination.value.page = 1;
  await fetchHistory();
}

async function fetchHistory() {
  if (!currentServerId.value) return;
  historyLoading.value = true;
  const { data, error } = await fetchCaddyConfigHistory(currentServerId.value, {
    page: historyPagination.value.page,
    pageSize: historyPagination.value.pageSize
  });
  historyLoading.value = false;
  if (error) {
    message.error('获取历史版本失败');
    return;
  }
  historyList.value = data?.list || [];
  historyPagination.value.itemCount = data?.total || 0;
}

async function fetchHistoryDetail(historyId: number) {
  if (!currentServerId.value) return null;
  const { data, error } = await fetchCaddyConfigHistoryDetail(currentServerId.value, historyId);
  if (error) {
    message.error('获取历史配置失败');
    return null;
  }
  return data;
}

async function openHistoryDetail(historyId: number) {
  const detail = await fetchHistoryDetail(historyId);
  if (!detail) return;
  historyDetail.value = {
    id: detail.id,
    createdAt: detail.createdAt,
    action: detail.action,
    hash: detail.hash,
    config: detail.config || ''
  };
  showHistoryDetailModal.value = true;
}

async function openHistoryCompare(historyId: number) {
  const detail = await fetchHistoryDetail(historyId);
  if (!detail) return;
  historyCompareLeft.value = detail.config || '';
  historyDiffOnly.value = false;
  showHistoryCompareModal.value = true;
}

function handleHistoryPageChange(page: number) {
  historyPagination.value.page = page;
  fetchHistory();
}

async function handleRollback(historyId: number) {
  if (!currentServerId.value) return;
  dialog.warning({
    title: '确认回滚',
    content: '确定要回滚到该版本吗？',
    positiveText: '确认',
    negativeText: '取消',
    onPositiveClick: async () => {
      const { error } = await rollbackCaddyConfig(currentServerId.value!, historyId);
      if (error) {
        message.error('回滚失败');
        return;
      }
      message.success('回滚成功');
      await getConfig();
      await fetchHistory();
    }
  });
}

// Watchers
watch(currentServerId, () => {
  if (currentServerId.value) {
    getConfig();
    wafIntegrationUnavailable.value = false;
    fetchWafIntegrationState();
    return;
  }
  configContent.value = '';
  formModel.value = createEmptyFormModel();
  syncQuickStateFromForm(formModel.value);
});

onMounted(() => {
  getServers();
});
</script>

<template>
  <div class="h-full flex flex-col overflow-hidden">
    <NCard
      class="h-full card-wrapper"
      :content-style="{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }"
    >
      <template #header>
        <div class="caddy-toolbar">
          <div class="min-w-0 flex-1">
            <NSelect
              v-model:value="currentServerId"
              :options="serverOptions"
              placeholder="选择服务器"
              class="max-w-72 w-full"
              size="small"
            />
          </div>
          <div class="caddy-toolbar-actions">
            <NButton
              v-if="pageMode === 'quick'"
              type="primary"
              size="small"
              :loading="saving"
              :disabled="!currentServerId"
              @click="saveQuickConfig"
            >
              保存快速配置
            </NButton>
            <NButton
              v-else-if="pageMode === 'raw'"
              type="primary"
              size="small"
              :loading="saving"
              :disabled="!currentServerId"
              @click="saveRawConfig"
            >
              保存原始配置
            </NButton>
            <NTag v-else-if="pageMode === 'waf'" size="small" type="warning" :bordered="false">防火墙设置</NTag>
            <NTag v-else size="small" type="info" :bordered="false">预览模式</NTag>

            <NDropdown :options="moreOptions" @select="handleMoreAction">
              <NButton size="small" secondary>
                <div class="flex items-center gap-1">
                  <span>更多</span>
                  <SvgIcon icon="carbon:chevron-down" class="caddy-icon" />
                </div>
              </NButton>
            </NDropdown>
          </div>
        </div>
      </template>

      <div class="min-h-0 flex flex-col flex-1 gap-4 overflow-hidden">
        <div class="caddy-mode-strip">
          <NRadioGroup :value="pageMode" size="small" @update:value="handleModeChange">
            <NRadioButton v-for="option in pageModeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </NRadioButton>
          </NRadioGroup>
          <div class="text-xs text-gray-500">
            {{ pageModeSummary }}
          </div>
        </div>

        <div v-if="servers.length === 0" class="h-full flex flex-col items-center justify-center p-8 text-gray-400">
          <div class="text-lg">未找到 Caddy 服务器</div>
          <div class="mt-2 text-sm">先添加一个服务器，再开始管理配置。</div>
          <NButton class="mt-4" type="primary" @click="openAddServerModal">添加服务器</NButton>
        </div>

        <NSpin v-else :show="loading" class="min-h-0 flex-1" content-class="h-full min-h-0">
          <NAlert
            v-if="pageMode === 'quick' && quickValidationErrors.length"
            type="error"
            :show-icon="true"
            class="mb-3"
          >
            {{ quickValidationErrors[0] }}
          </NAlert>

          <QuickConfigPanel
            v-if="pageMode === 'quick'"
            v-model:active-site-id="activeQuickSiteId"
            :sites="quickSiteDrafts"
            :complex-sites="complexSiteSummaries"
            @add="addQuickSite"
            @duplicate="duplicateQuickSite"
            @remove="removeQuickSite"
            @switch-raw="switchToRawFromQuick"
          />
          <SimpleWafPanel
            v-else-if="pageMode === 'waf'"
            :server-id="currentServerId"
            :on-applied="handleSimpleWafApplied"
          />
          <RawEditorPanel v-else-if="pageMode === 'raw'" v-model="configContent" />
          <ConfigPreviewPanel v-else :config-content="formattedConfigContent" />
        </NSpin>
      </div>
    </NCard>

    <NDrawer v-model:show="showSettingsDrawer" :width="560" placement="right">
      <NDrawerContent title="更多设置" closable>
        <NSpace vertical size="large">
          <NCard size="small" :bordered="false">
            <template #header>配置工具</template>
            <div class="flex flex-wrap gap-2">
              <NButton size="small" @click="applyPreset">应用默认模板</NButton>
              <NButton size="small" @click="importRawToStructured">从原始配置解析</NButton>
              <NButton size="small" :disabled="!currentServerId" @click="openHistoryModal">查看历史版本</NButton>
            </div>
          </NCard>

          <NCard size="small" :bordered="false">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <span>全局配置</span>
                <div class="flex items-center gap-2">
                  <NTag v-if="globalRawChanged" type="warning" size="small" :bordered="false">未保存</NTag>
                  <NButton size="tiny" secondary :disabled="!initialGlobalRaw" @click="restoreGlobalRaw">
                    恢复已保存
                  </NButton>
                  <NButton
                    size="tiny"
                    :disabled="!formModel.global.raw && !initialGlobalRaw"
                    @click="openGlobalCompare"
                  >
                    对比
                  </NButton>
                </div>
              </div>
            </template>

            <NAlert v-if="pageMode !== 'quick'" type="info" :show-icon="true" class="mb-3">
              全局配置仅在“快速配置”模式下可编辑；原始配置模式请直接维护完整 Caddyfile。
            </NAlert>

            <div class="relative h-[240px]">
              <VueMonacoEditor
                v-model:value="formModel.global.raw"
                language="shell"
                theme="vs"
                :options="{
                  automaticLayout: true,
                  fixedOverflowWidgets: true,
                  readOnly: pageMode !== 'quick',
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  wordWrap: 'on'
                }"
                class="absolute inset-0"
              />
            </div>
          </NCard>

          <NCard size="small" :bordered="false">
            <template #header>高级集成</template>
            <WafIntegrationCard
              :loading="wafIntegrationLoading"
              :submitting="wafIntegrationSubmitting"
              :previewing="wafIntegrationPreviewing"
              :unavailable="wafIntegrationUnavailable"
              :status="wafIntegrationStatus"
              :selected-sites="selectedWafIntegrationSites"
              :preview-actions="wafIntegrationPreviewActions"
              :on-refresh="handleRefreshWafIntegrationState"
              :on-preview="handlePreviewWafIntegration"
              :on-enable="handleEnableWafIntegration"
              :on-disable="handleDisableWafIntegration"
              :on-site-change="handleWafIntegrationSiteChange"
            />
          </NCard>
        </NSpace>
      </NDrawerContent>
    </NDrawer>

    <NModal v-model:show="showHistoryModal" preset="card" title="配置历史" class="max-w-3xl w-[90vw]">
      <NDataTable
        :columns="historyColumns"
        :data="historyList"
        :loading="historyLoading"
        :pagination="{
          page: historyPagination.page,
          pageSize: historyPagination.pageSize,
          itemCount: historyPagination.itemCount,
          onUpdatePage: handleHistoryPageChange
        }"
        size="small"
      />
    </NModal>

    <NModal
      v-model:show="showHistoryDetailModal"
      preset="card"
      :title="historyDetail ? `历史配置预览 - ${historyDetail.createdAt}` : '历史配置预览'"
      class="max-w-5xl w-[90vw]"
    >
      <div class="mb-3 flex flex-wrap items-center gap-3 text-xs text-gray-500">
        <span>动作：{{ historyDetail ? formatHistoryAction(historyDetail.action) : '-' }}</span>
        <span>时间：{{ historyDetail?.createdAt ?? '-' }}</span>
      </div>
      <div class="relative h-[60vh]">
        <VueMonacoEditor
          :value="historyDetailFormattedConfig"
          language="shell"
          theme="vs"
          :options="{
            automaticLayout: true,
            fixedOverflowWidgets: true,
            readOnly: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on'
          }"
          class="absolute inset-0"
        />
      </div>
    </NModal>

    <NModal v-model:show="showHistoryCompareModal" preset="card" title="历史配置对比" class="max-w-5xl w-[90vw]">
      <div class="diff-head">
        <div>历史版本</div>
        <div class="flex items-center justify-between">
          <span>当前配置</span>
          <NSwitch v-model:value="historyDiffOnly" size="small">
            <template #checked>仅差异</template>
            <template #unchecked>全部</template>
          </NSwitch>
        </div>
      </div>
      <div class="relative h-[65vh]">
        <VueMonacoDiffEditor
          :original="historyCompareLeftFormatted"
          :modified="historyCompareRight"
          language="shell"
          theme="vs"
          :options="{
            automaticLayout: true,
            readOnly: true,
            renderSideBySide: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on',
            hideUnchangedRegions: { enabled: historyDiffOnly }
          }"
          class="absolute inset-0"
        />
      </div>
    </NModal>

    <NModal v-model:show="showGlobalCompareModal" preset="card" title="全局配置对比" class="max-w-4xl w-[90vw]">
      <div class="diff-head">
        <div>已保存</div>
        <div class="flex items-center justify-between">
          <span>当前</span>
          <NSwitch v-model:value="showGlobalDiffOnly" size="small">
            <template #checked>仅差异</template>
            <template #unchecked>全部</template>
          </NSwitch>
        </div>
      </div>
      <div class="diff-body diff-two">
        <div ref="diffLeftRef" class="diff-column" @scroll="syncDiffScroll('left')">
          <div v-for="row in globalDiffRows" :key="row.key" class="diff-line-row" :class="row.type">
            <span class="diff-no">{{ row.leftNo ?? '' }}</span>
            <span class="diff-line">{{ row.left !== null ? row.left : '' }}</span>
          </div>
        </div>
        <div ref="diffRightRef" class="diff-column" @scroll="syncDiffScroll('right')">
          <div v-for="row in globalDiffRows" :key="row.key" class="diff-line-row" :class="row.type">
            <span class="diff-no">{{ row.rightNo ?? '' }}</span>
            <span class="diff-line">{{ row.right !== null ? row.right : '' }}</span>
          </div>
        </div>
      </div>
    </NModal>

    <NModal v-model:show="savePreviewVisible" preset="card" title="保存预览" class="max-w-5xl w-[90vw]">
      <div class="mb-3 flex flex-wrap items-center gap-2 text-xs text-gray-500">
        <NTag size="small" type="info" :bordered="false">
          {{ savePreviewKind === 'quick' ? '快速配置' : '原始配置' }}
        </NTag>
        <span v-if="savePreviewActions.length">动作：{{ savePreviewActions.join(' / ') }}</span>
      </div>
      <NAlert v-if="savePreviewErrors.length" type="error" :show-icon="true" class="mb-3">
        {{ savePreviewErrors[0] }}
      </NAlert>
      <div class="relative h-[60vh]">
        <VueMonacoEditor
          :value="savePreviewConfig"
          language="shell"
          theme="vs"
          :options="{
            automaticLayout: true,
            fixedOverflowWidgets: true,
            readOnly: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on'
          }"
          class="absolute inset-0"
        />
      </div>
      <div class="mt-4 flex justify-end gap-2">
        <NButton secondary :disabled="saving" @click="closeSavePreview">取消</NButton>
        <NButton type="primary" :loading="saving" @click="confirmSavePreview">确认保存</NButton>
      </div>
    </NModal>

    <!-- Server Management Modal -->
    <!-- ... modal content ... -->
    <NModal
      v-model:show="showServerModal"
      preset="card"
      :title="serverModalType === 'add' ? '添加服务器' : '编辑服务器'"
      class="w-500px"
    >
      <!-- ... form content unrelated to layout ... -->
      <NForm label-placement="left" label-width="80">
        <NFormItem label="名称" path="name">
          <NInput v-model:value="serverFormModel.name" placeholder="服务器名称" />
        </NFormItem>
        <NFormItem label="地址" path="url">
          <NInput v-model:value="serverFormModel.url" placeholder="http://localhost:2019" />
        </NFormItem>
        <NFormItem label="类型" path="type">
          <NRadioGroup v-model:value="serverFormModel.type">
            <NRadioButton value="local">本地</NRadioButton>
            <NRadioButton value="remote">远程</NRadioButton>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="serverFormModel.type === 'remote'" label="凭证" path="token">
          <NInput v-model:value="serverFormModel.token" placeholder="可选认证凭证" />
        </NFormItem>
        <div class="flex justify-end gap-2">
          <NButton @click="showServerModal = false">取消</NButton>
          <NButton type="primary" @click="handleSaveServer">保存</NButton>
        </div>
      </NForm>
    </NModal>
  </div>
</template>

<style scoped>
:deep(.n-card__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
}
:deep(.n-spin-content) {
  height: 100%;
}

/* Ensure Monaco widgets (like search) are on top */
:deep(.monaco-editor-overlay) {
  z-index: 1000 !important;
}

.caddy-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.caddy-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.caddy-mode-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 12px 14px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.caddy-icon {
  display: inline-block;
  font-size: 16px;
  line-height: 1;
  vertical-align: middle;
}

@media (max-width: 900px) {
  .caddy-toolbar {
    align-items: stretch;
  }
  .caddy-toolbar-actions {
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}

.diff-head {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  font-size: 12px;
  color: #64748b;
  margin-bottom: 8px;
}

.diff-body {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  overflow: hidden;
  max-height: 60vh;
  background: #ffffff;
}

.diff-two {
  display: grid;
  grid-template-columns: 1fr 1fr;
}

.diff-column {
  overflow: auto;
  max-height: 60vh;
}

.diff-column + .diff-column {
  border-left: 1px solid #e2e8f0;
}

.diff-line-row {
  display: grid;
  grid-template-columns: 32px 1fr;
  gap: 8px;
  padding: 6px 10px;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  white-space: pre-wrap;
}

.diff-line-row.added {
  background: #ecfdf3;
}

.diff-line-row.removed {
  background: #fef2f2;
}

.diff-line-row.changed {
  background: #fff7ed;
}

.diff-line {
  display: block;
  overflow-wrap: anywhere;
}

.diff-no {
  color: #94a3b8;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
</style>
