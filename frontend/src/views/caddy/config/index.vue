<script setup lang="ts">
import { ref, onMounted, computed, watch, h } from 'vue';
import { useMessage, useDialog, NTag, NButton } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { VueMonacoEditor, VueMonacoDiffEditor, loader } from '@guolao/vue-monaco-editor';
import { fetchCaddyServers, fetchCaddyConfig, updateCaddyConfigRaw, updateCaddyConfigStructured, addCaddyServer, updateCaddyServer, deleteCaddyServer, fetchCaddyConfigHistory, fetchCaddyConfigHistoryDetail, rollbackCaddyConfig } from '@/service/api/caddy';
import ConfigPreviewPanel from './components/ConfigPreviewPanel.vue';
import QuickConfigPanel from './components/QuickConfigPanel.vue';
import RawEditorPanel from './components/RawEditorPanel.vue';
import SimpleWafPanel from './components/SimpleWafPanel.vue';
import WafIntegrationCard from './components/WafIntegrationCard.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import type { CaddyFormModel, Route, RouteMatch, Site } from './types';
import { applyWafIntegration, fetchWafIntegrationStatus, type WafIntegrationStatusResp } from '@/service/api/caddy-integration';
import {
  buildQuickConfigState,
  createQuickSiteDraft,
  mergeQuickConfigDrafts,
  type ComplexSiteSummary,
  type QuickSiteDraft
} from './quick-config-utils';

// Configure Monaco Editor loader to use npmmirror for better performance in China
loader.config({
  paths: {
    vs: 'https://registry.npmmirror.com/monaco-editor/0.44.0/files/min/vs',
  },
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

type DiffRow = {
  left: string | null;
  right: string | null;
  type: 'same' | 'added' | 'removed' | 'changed';
  leftNo: number | null;
  rightNo: number | null;
  key: string;
};

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
const effectiveConfigContent = computed(() => (lastEditMode.value === 'raw' ? configContent.value : generatedQuickCaddyfile.value));
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
const historyDetailFormattedConfig = computed(() => (historyDetail.value ? formatCaddyfile(historyDetail.value.config) : ''));
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
  const { error } = await updateCaddyConfigRaw(currentServerId.value, configContent.value);
  saving.value = false;

  if (error) {
    message.error('保存配置失败');
    return;
  }
  message.success('配置已保存并自动热重载 Caddy');
  structuredAvailable.value = false;
  lastEditMode.value = 'raw';
  pageMode.value = 'preview';
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
  const { error } = await updateCaddyConfigStructured(currentServerId.value, content, modules);
  saving.value = false;

  if (error) {
    message.error('保存配置失败');
    return;
  }

  message.success('配置已保存并自动热重载 Caddy');
  formModel.value = nextFormModel;
  configContent.value = content;
  structuredAvailable.value = true;
  initialGlobalRaw.value = formModel.value.global?.raw ?? '';
  syncQuickStateFromForm(formModel.value);
  lastEditMode.value = 'quick';
  pageMode.value = 'preview';
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

function validateStructuredConfig(model: CaddyFormModel = formModel.value): string[] {
  const errors: string[] = [];
  const pushError = (errorMessage: string) => {
    errors.push(errorMessage);
  };
  const domainRe = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
  const portOnlyRe = /^:\d+$/;
  const methodAllowList = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];
  const isValidPathPattern = (value: string) => {
    if (!value) return false;
    if (value.startsWith('/')) return true;
    if (value.startsWith('*')) return true;
    if (value.startsWith('{') && value.endsWith('}')) return true;
    return false;
  };
  const enabledSites = model.sites.filter(s => s.enabled);
  const hasSites = enabledSites.length > 0;
  const hasGlobalRaw = !!model.global?.raw?.trim();
  if (!hasSites && !hasGlobalRaw) {
    pushError('至少需要一个站点或全局配置');
  }
  const upstreamNames = new Set<string>();
  for (const up of model.upstreams) {
    if (!up.name) pushError('上游名称不能为空');
    if (upstreamNames.has(up.name)) pushError(`上游名称重复: ${up.name}`);
    upstreamNames.add(up.name);
    if (up.targets.length === 0) pushError(`上游 ${up.name} 至少配置一个目标`);
  }
  for (const site of model.sites) {
    if (!site.enabled) continue;
    if (!site.name) pushError('站点名称不能为空');
    if (site.domains.length === 0) pushError(`站点 ${site.name || site.id} 至少配置一个域名`);
    const hasEnabledRoutes = site.routes.some(route => route.enabled);
    const hasImports = (site.imports ?? []).some(item => item.trim().length > 0);
    if (!hasEnabledRoutes && !hasImports) {
      pushError(`站点 ${site.name || site.id} 至少配置一个路由或 import`);
    }
    const invalidDomains = site.domains.filter(d => d && !(domainRe.test(d) || portOnlyRe.test(d)));
    if (invalidDomains.length) pushError(`站点 ${site.name || site.id} 域名格式不合法: ${invalidDomains.join(', ')}`);
    if (site.tls?.mode === 'manual' && (!site.tls.certFile || !site.tls.keyFile)) {
      pushError(`站点 ${site.name || site.id} TLS 手动模式需填写证书和私钥`);
    }
    for (const route of site.routes) {
      if (!route.enabled) continue;
      if (!route.name) pushError(`站点 ${site.name || site.id} 有未命名路由`);
      if (route.handles.length === 0) pushError(`路由 ${route.name || route.id} 至少一个 Handler`);
      if (route.handles.every(h => !h.enabled)) {
        pushError(`路由 ${route.name || route.id} 至少启用一个 Handler`);
      }
      const invalidPaths = route.match.path.filter(p => p && !isValidPathPattern(p));
      if (invalidPaths.length) pushError(`路由 ${route.name || route.id} Path 格式不合法: ${invalidPaths.join(', ')}`);
      const invalidMethods = route.match.method.filter(m => m && !methodAllowList.includes(m.toUpperCase()));
      if (invalidMethods.length) pushError(`路由 ${route.name || route.id} Method 非法: ${invalidMethods.join(', ')}`);
      for (const handle of route.handles) {
        if (!handle.enabled) continue;
        if (handle.type === 'reverse_proxy' && !handle.upstream) {
          pushError(`路由 ${route.name || route.id} 的 reverse_proxy 未选择上游`);
        }
      }
    }
  }
  return errors;
}

function buildCaddyfile(model: CaddyFormModel, options?: { includeDisabled?: boolean; includeGlobal?: boolean }): string {
  const lines: string[] = [];
  const globalRaw = model.global?.raw?.trim();
  if (globalRaw && options?.includeGlobal !== false) {
    lines.push(globalRaw);
    lines.push('');
  }
  if (!model.sites || model.sites.length === 0) {
    return lines.length ? lines.join('\n').trim() : '# No sites defined';
  }
  const upstreamMap = new Map(model.upstreams.map(u => [u.name, u]));
  const includeDisabled = options?.includeDisabled ?? false;
  for (const site of model.sites.filter(s => includeDisabled || s.enabled)) {
    const usedMatcherNames = new Set<string>();
    const hosts = site.domains.join(' ');
    if (!hosts) continue;
    lines.push(`${hosts} {`);
    if (site.geoip2Vars?.length) {
      site.geoip2Vars.forEach(v => {
        if (v) lines.push(`  geoip2_vars ${v}`);
      });
    }
    if (site.imports?.length) {
      site.imports.forEach(v => {
        if (v) lines.push(`  import ${v}`);
      });
    }
    if (site.encode?.length) {
      lines.push(`  encode ${site.encode.join(' ')}`);
    }
    if (site.tls?.mode) {
      if (site.tls.mode === 'off') lines.push(`  tls off`);
      else if (site.tls.mode === 'internal') lines.push(`  tls internal`);
      else if (site.tls.mode === 'manual' && site.tls.certFile && site.tls.keyFile) {
        lines.push(`  tls ${site.tls.certFile} ${site.tls.keyFile}`);
      }
    }
    for (const route of site.routes.filter(r => includeDisabled || r.enabled)) {
      const matcherLines: string[] = [];
      if (route.match.host.length) matcherLines.push(`host ${route.match.host.join(' ')}`);
      if (route.match.path.length) matcherLines.push(`path ${route.match.path.join(' ')}`);
      if (route.match.method.length) matcherLines.push(`method ${route.match.method.join(' ')}`);
      if (route.match.header.length) {
        matcherLines.push(
          `header ${route.match.header.map(h => `${h.key} ${h.value}`).join(' ')}`
        );
      }
      if (route.match.query.length) {
        matcherLines.push(
          `query ${route.match.query.map(q => `${q.key}=${q.value}`).join(' ')}`
        );
      }
      if (route.match.expression) {
        matcherLines.push(`expression ${route.match.expression}`);
      }

      let matcherName = '';
      const rawRouteName = route.name?.trim() ?? '';
      if (rawRouteName.startsWith('@')) {
        const token = rawRouteName.slice(1);
        if (/^[a-zA-Z0-9_-]+$/.test(token)) matcherName = rawRouteName;
      }
      if (!matcherName) {
        const base = `@m_${route.id.slice(0, 6)}`;
        matcherName = base;
        let idx = 1;
        while (usedMatcherNames.has(matcherName)) {
          matcherName = `${base}_${idx}`;
          idx += 1;
        }
      }
      if (matcherLines.length) {
        usedMatcherNames.add(matcherName);
      }

      const enabledHandles = route.handles.filter(hd => includeDisabled || hd.enabled);
      const headerOnly =
        enabledHandles.length > 0 &&
        enabledHandles.every(h => h.type === 'header') &&
        !(route.logAppend?.length);
      const fileServerOnly =
        enabledHandles.length > 0 &&
        enabledHandles.every(h => h.type === 'file_server') &&
        !(route.logAppend?.length);

      if (matcherLines.length) {
        lines.push(`  ${matcherName} {`);
        matcherLines.forEach(l => lines.push(`    ${l}`));
        lines.push(`  }`);
        if (headerOnly) {
          for (const h of enabledHandles) {
            for (const r of h.rules ?? []) {
              if (r.op === 'delete') lines.push(`  header ${matcherName} -${r.key}`);
              else lines.push(`  header ${matcherName} ${r.key} ${r.value ?? ''}`.replace(/\s+$/, ''));
            }
          }
          continue;
        }
        lines.push(`  handle ${matcherName} {`);
      } else if (headerOnly) {
        for (const h of enabledHandles) {
          for (const r of h.rules ?? []) {
            if (r.op === 'delete') lines.push(`  header -${r.key}`);
            else lines.push(`  header ${r.key} ${r.value ?? ''}`.replace(/\s+$/, ''));
          }
        }
        continue;
      } else if (fileServerOnly) {
        for (const h of enabledHandles) {
          if (h.root) lines.push(`  root * ${h.root}`);
          lines.push(`  file_server${h.browse ? ' browse' : ''}`);
        }
        continue;
      } else {
        lines.push(`  handle {`);
      }

      for (const h of enabledHandles) {
        if (h.type === 'reverse_proxy') {
          const up = h.upstream ? upstreamMap.get(h.upstream) : undefined;
          const targets = up?.targets.length
            ? up.targets.join(' ')
            : h.upstream
              ? h.upstream
              : 'localhost:8080';
          const transport = h.transportProtocol || (h.tlsInsecureSkipVerify ? 'http' : '');
          if (transport || h.tlsInsecureSkipVerify) {
            lines.push(`    reverse_proxy ${targets} {`);
            if (transport) {
              lines.push(`      transport ${transport} {`);
              if (h.tlsInsecureSkipVerify) lines.push(`        tls_insecure_skip_verify`);
              lines.push(`      }`);
            }
            lines.push(`    }`);
          } else {
            lines.push(`    reverse_proxy ${targets}`);
          }
        } else if (h.type === 'file_server') {
          if (h.root) lines.push(`    root * ${h.root}`);
          lines.push(`    file_server${h.browse ? ' browse' : ''}`);
        } else if (h.type === 'respond') {
          const body = (h.body ?? '').replaceAll('"', '\\"');
          lines.push(`    respond "${body}" ${h.status ?? 200}`);
        } else if (h.type === 'redirect') {
          const code = h.code ?? 302;
          const codeStr = code === 308 ? 'permanent' : code === 302 ? 'temporary' : String(code);
          lines.push(`    redir ${h.to ?? '/'} ${codeStr}`);
        } else if (h.type === 'header') {
          for (const r of h.rules ?? []) {
            if (r.op === 'delete') lines.push(`    header -${r.key}`);
            else lines.push(`    header ${r.key} ${r.value ?? ''}`.replace(/\s+$/, ''));
          }
        } else if (h.type === 'rewrite') {
          lines.push(`    rewrite * ${h.uri ?? '/'}`);
        }
      }
      if (route.logAppend?.length) {
        for (const item of route.logAppend) {
          if (!item.key) continue;
          lines.push(`    log_append ${item.key} ${item.value ?? ''}`.replace(/\s+$/, ''));
        }
      }

      lines.push(`  }`);
    }
    lines.push(`}`);
    lines.push('');
  }
  const result = lines.join('\n').trim();
  return result || '# No routes defined';
}

function genId() {
  return (crypto as any).randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

const siteDomainRe = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
const siteIpv4Re = /^(?:\d{1,3}\.){3}\d{1,3}$/;

function isSiteToken(token: string) {
  if (!token) return false;
  if (token.startsWith('(') || token.endsWith(')') || token.includes('(') || token.includes(')')) return false;
  if (token.startsWith(':') && /^\:\d+$/.test(token)) return true;
  const [host, port] = token.split(':');
  if (port) {
    if (!/^\d+$/.test(port)) return false;
    return siteDomainRe.test(host) || siteIpv4Re.test(host) || host === 'localhost';
  }
  return siteDomainRe.test(token);
}

function extractGlobalOptionsBlock(content: string): { raw: string; rest: string } {
  const lines = content.split('\n');
  const globalLines: string[] = [];
  const restLines: string[] = [];
  let depth = 0;
  let currentBlock: 'global' | 'site' | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    const sanitized = line.replace(/#.*/, '');
    const openCount = (sanitized.match(/{/g) || []).length;
    const closeCount = (sanitized.match(/}/g) || []).length;

    if (depth === 0 && openCount > 0) {
      const before = sanitized.split('{')[0].trim();
      let blockType: 'global' | 'site' = 'global';
      if (trimmed.startsWith('{')) {
        blockType = 'global';
      } else if (before.startsWith('(')) {
        blockType = 'global';
      } else {
        const tokens = before.replace(/,/g, ' ').split(/\s+/).filter(Boolean);
        const hasSiteToken = tokens.some(t => isSiteToken(t));
        blockType = hasSiteToken ? 'site' : 'global';
      }
      currentBlock = blockType;
    }

    if (depth === 0 && openCount === 0 && currentBlock === null) {
      if (trimmed.length > 0) {
        globalLines.push(line);
      } else {
        restLines.push(line);
      }
      continue;
    }

    if (currentBlock === 'site') {
      restLines.push(line);
    } else {
      globalLines.push(line);
    }

    depth += openCount - closeCount;
    if (depth <= 0) {
      depth = 0;
      currentBlock = null;
    }
  }

  return {
    raw: globalLines.join('\n').trim(),
    rest: restLines.join('\n')
  };
}

function parseCaddyfileToModules(content: string): CaddyFormModel {
  const { raw: globalRaw, rest } = extractGlobalOptionsBlock(content);
  const sites: Site[] = [];
  const matchers: Record<string, RouteMatch> = {};
  const lines = rest.split('\n');
  let depth = 0;
  let currentSite: Site | null = null;
  let currentRoute: Route | null = null;
  let currentHandleBlock = false;
  let handleDepth: number | null = null;
  let currentMatcherName: string | null = null;
  let matcherDepth: number | null = null;
  let reverseProxyDepth: number | null = null;
  let currentReverseProxy: any | null = null;
  let currentSiteRoot = '';

  function ensureDefaultRoute() {
    if (!currentSite) return;
    if (!currentRoute) {
      currentRoute = {
        id: genId(),
        name: '默认路由',
        enabled: true,
        match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
        logAppend: [],
        handles: []
      };
      currentSite.routes.push(currentRoute);
    }
  }

  function cloneMatcher(matcher?: RouteMatch): RouteMatch {
    if (!matcher) return { host: [], path: [], method: [], header: [], query: [], expression: '' };
    return {
      host: [...(matcher.host ?? [])],
      path: [...(matcher.path ?? [])],
      method: [...(matcher.method ?? [])],
      header: [...(matcher.header ?? [])],
      query: [...(matcher.query ?? [])],
      expression: matcher.expression ?? ''
    };
  }

  function createRouteForMatcher(matcherName?: string): Route | null {
    if (!currentSite) return null;
    if (!matcherName) {
      ensureDefaultRoute();
      return currentRoute;
    }
    const match = cloneMatcher(matchers[matcherName]);
    const route: Route = {
      id: genId(),
      name: `@${matcherName}`,
      enabled: true,
      match,
      logAppend: [],
      handles: []
    };
    currentSite.routes.push(route);
    return route;
  }

  for (const raw of lines) {
    const line = raw.replace(/#.*/, '').trim();
    if (!line) continue;
    const openCount = (line.match(/{/g) || []).length;
    const closeCount = (line.match(/}/g) || []).length;
    if (reverseProxyDepth !== null && depth >= reverseProxyDepth) {
      if (line.startsWith('transport ') && currentReverseProxy) {
        const proto = line.replace('transport ', '').replace('{', '').trim();
        if (proto) currentReverseProxy.transportProtocol = proto;
      }
      if (line.includes('tls_insecure_skip_verify') && currentReverseProxy) {
        currentReverseProxy.tlsInsecureSkipVerify = true;
      }
      depth += openCount - closeCount;
      if (depth < reverseProxyDepth) {
        reverseProxyDepth = null;
        currentReverseProxy = null;
      }
      continue;
    }
    if (depth === 0 && line.includes('{') && !line.startsWith('{')) {
      const before = line.split('{')[0].trim();
      if (before) {
        const domains = before
          .replace(/,/g, ' ')
          .split(/\s+/)
          .filter(Boolean)
          .filter(isSiteToken);
        if (domains.length > 0) {
          currentSite = {
            id: genId(),
            name: domains[0],
            enabled: true,
            domains,
            imports: [],
            geoip2Vars: [],
            encode: [],
            tls: { mode: 'auto' },
            routes: []
          };
          sites.push(currentSite);
          currentSiteRoot = '';
          currentRoute = null;
          currentHandleBlock = false;
          handleDepth = null;
          currentMatcherName = null;
          matcherDepth = null;
          reverseProxyDepth = null;
          currentReverseProxy = null;
        }
      }
    }

    if (currentSite) {
      // matcher block: @m { host ... }
      if (line.startsWith('@') && line.includes('{')) {
        const name = line.split('{')[0].trim().slice(1);
        if (name) {
          matchers[name] = { host: [], path: [], method: [], header: [], query: [], expression: '' };
          currentMatcherName = name;
          matcherDepth = depth + openCount;
        }
      } else if (currentMatcherName) {
        const matcher = matchers[currentMatcherName];
        if (line.startsWith('host ')) matcher.host = line.replace('host ', '').split(/\s+/).filter(Boolean);
        if (line.startsWith('path ')) matcher.path = line.replace('path ', '').split(/\s+/).filter(Boolean);
        if (line.startsWith('method ')) matcher.method = line.replace('method ', '').split(/\s+/).filter(Boolean);
        if (line.startsWith('header ')) {
          const parts = line.replace('header ', '').split(/\s+/);
          if (parts.length >= 2) matcher.header.push({ key: parts[0], value: parts.slice(1).join(' ') });
        }
        if (line.startsWith('query ')) {
          const parts = line.replace('query ', '').split(/\s+/);
          parts.forEach(q => {
            const [k, v] = q.split('=');
            if (k) matcher.query.push({ key: k, value: v ?? '' });
          });
        }
        if (line.startsWith('expression ')) {
          matcher.expression = line.replace('expression ', '').trim();
        }
      }

      if (!currentHandleBlock && !currentMatcherName) {
        if (line.startsWith('import ')) {
          currentSite.imports = currentSite.imports || [];
          const value = line.replace('import ', '').trim();
          if (value) currentSite.imports.push(value);
        } else if (line.startsWith('geoip2_vars ')) {
          currentSite.geoip2Vars = currentSite.geoip2Vars || [];
          const value = line.replace('geoip2_vars ', '').trim();
          if (value) currentSite.geoip2Vars.push(value);
        } else if (line.startsWith('encode ')) {
          const enc = line.replace('encode ', '').trim().split(/\s+/).filter(Boolean);
          currentSite.encode = enc;
        } else if (line.startsWith('tls') && !line.startsWith('tls_insecure')) {
          const parts = line.split(/\s+/).filter(Boolean);
          if (parts.length === 1) {
            currentSite.tls = { mode: 'auto' };
          } else if (parts[1] === 'off') {
            currentSite.tls = { mode: 'off' };
          } else if (parts[1] === 'internal') {
            currentSite.tls = { mode: 'internal' };
          } else if (parts.length >= 3) {
            currentSite.tls = { mode: 'manual', certFile: parts[1], keyFile: parts[2] };
          }
        } else if (line.startsWith('root ')) {
          const parts = line.split(/\s+/).filter(Boolean);
          if (parts.length >= 3) {
            let idx = 1;
            if (parts[idx] === '*' || parts[idx].startsWith('@')) idx += 1;
            const rootPath = parts.slice(idx).join(' ');
            if (rootPath) {
              currentSiteRoot = rootPath;
              for (const route of currentSite.routes) {
                for (const h of route.handles) {
                  if (h.type === 'file_server' && !h.root) {
                    h.root = rootPath;
                  }
                }
              }
            }
          }
        } else if (line.startsWith('file_server')) {
          const parts = line.split(/\s+/).filter(Boolean);
          const matcherName = parts[1]?.startsWith('@') ? parts[1].slice(1) : '';
          const browse = parts.includes('browse');
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'file_server',
              enabled: true,
              root: currentSiteRoot || undefined,
              browse
            });
          }
        } else if (line.startsWith('reverse_proxy ')) {
          const parts = line.split(/\s+/).filter(Boolean);
          let matcherName = '';
          let idx = 1;
          if (parts[1]?.startsWith('@')) {
            matcherName = parts[1].slice(1);
            idx = 2;
          }
          const rawTargets = parts.slice(idx);
          const targets: string[] = [];
          for (const t of rawTargets) {
            if (t === '{') break;
            if (t.endsWith('{')) {
              const cleaned = t.slice(0, -1);
              if (cleaned) targets.push(cleaned);
              break;
            }
            targets.push(t);
          }
          const route = createRouteForMatcher(matcherName);
          if (route) {
            const handle = {
              id: genId(),
              type: 'reverse_proxy' as const,
              enabled: true,
              upstream: targets.join(' ').trim(),
              transportProtocol: '',
              tlsInsecureSkipVerify: false
            };
            route.handles.push(handle);
            if (line.includes('{')) {
              reverseProxyDepth = depth + openCount;
              currentReverseProxy = handle;
            }
          }
        } else if (line.startsWith('respond ')) {
          const rest = line.replace('respond ', '').trim();
          const parts = rest.split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          const payload = parts.join(' ');
          const match = payload.match(/^"?(.*?)"?\s+(\d+)?$/);
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'respond',
              enabled: true,
              body: match?.[1] ?? '',
              status: match?.[2] ? Number(match[2]) : 200
            });
          }
        } else if (line.startsWith('redir ')) {
          const parts = line.replace('redir ', '').trim().split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          let code: number | undefined;
          if (parts[1]) {
            if (parts[1] === 'permanent') code = 308;
            else if (parts[1] === 'temporary') code = 302;
            else if (!Number.isNaN(Number(parts[1]))) code = Number(parts[1]);
          }
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'redirect',
              enabled: true,
              to: parts[0] ?? '/',
              code: code ?? 302
            });
          }
        } else if (line.startsWith('rewrite ')) {
          const parts = line.replace('rewrite ', '').trim().split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'rewrite',
              enabled: true,
              uri: parts[1] ?? parts[0]
            });
          }
        } else if (line.startsWith('header ')) {
          let rest = line.replace('header ', '').trim();
          const tokens = rest.split(/\s+/);
          let matcherName = '';
          if (tokens[0]?.startsWith('@')) {
            matcherName = tokens.shift()!.slice(1);
            rest = tokens.join(' ');
          }
          const isDelete = rest.startsWith('-');
          const kv = rest.replace(/^-/, '').split(/\s+/);
          const route = createRouteForMatcher(matcherName);
          if (route) {
            route.handles.push({
              id: genId(),
              type: 'header',
              enabled: true,
              rules: [{ op: isDelete ? 'delete' : 'set', key: kv[0] ?? '', value: kv.slice(1).join(' ') }]
            });
          }
        } else if (line.startsWith('log_append ')) {
          const parts = line.replace('log_append ', '').trim().split(/\s+/);
          let matcherName = '';
          if (parts[0]?.startsWith('@')) {
            matcherName = parts.shift()!.slice(1);
          }
          if (parts[0]) {
            const route = createRouteForMatcher(matcherName);
            if (route) {
              route.logAppend = route.logAppend || [];
              route.logAppend.push({ key: parts[0], value: parts.slice(1).join(' ') });
            }
          }
        }
      }

      if (line.startsWith('handle ') || line === 'handle {') {
        currentRoute = {
          id: genId(),
          name: line.startsWith('handle ') ? line.replace('handle', '').replace('{', '').trim() : '默认路由',
          enabled: true,
          match: { host: [], path: [], method: [], header: [], query: [], expression: '' },
          logAppend: [],
          handles: []
        };
        currentSite.routes.push(currentRoute);
        currentHandleBlock = true;
        handleDepth = depth + openCount;
        const matchName = line.startsWith('handle ')
          ? line.replace('handle', '').replace('{', '').trim().replace(/^@/, '')
          : '';
        if (matchName && currentRoute) {
          const m = matchers[matchName];
          if (m) currentRoute.match = m;
        }
      }

      if (currentHandleBlock && currentRoute) {
        if (line.startsWith('reverse_proxy ')) {
          const rawTargets = line.replace('reverse_proxy ', '').trim().split(/\s+/).filter(Boolean);
          const targets: string[] = [];
          for (const t of rawTargets) {
            if (t === '{') break;
            if (t.endsWith('{')) {
              const cleaned = t.slice(0, -1);
              if (cleaned) targets.push(cleaned);
              break;
            }
            targets.push(t);
          }
          const handle = {
            id: genId(),
            type: 'reverse_proxy' as const,
            enabled: true,
            upstream: targets.join(' ').trim(),
            transportProtocol: '',
            tlsInsecureSkipVerify: false
          };
          currentRoute.handles.push(handle);
          if (line.includes('{')) {
            reverseProxyDepth = depth + openCount;
            currentReverseProxy = handle;
          }
        } else if (line.startsWith('file_server')) {
          currentRoute.handles.push({
            id: genId(),
            type: 'file_server',
            enabled: true,
            browse: line.includes('browse')
          });
        } else if (line.startsWith('respond ')) {
          const parts = line.replace('respond ', '').match(/^"?(.*?)"?\s+(\d+)?$/);
          currentRoute.handles.push({
            id: genId(),
            type: 'respond',
            enabled: true,
            body: parts?.[1] ?? '',
            status: parts?.[2] ? Number(parts[2]) : 200
          });
        } else if (line.startsWith('redir ')) {
          const parts = line.replace('redir ', '').split(/\s+/);
          let code: number | undefined;
          if (parts[1]) {
            if (parts[1] === 'permanent') code = 308;
            else if (parts[1] === 'temporary') code = 302;
            else if (!Number.isNaN(Number(parts[1]))) code = Number(parts[1]);
          }
          currentRoute.handles.push({
            id: genId(),
            type: 'redirect',
            enabled: true,
            to: parts[0] ?? '/',
            code: code ?? 302
          });
        } else if (line.startsWith('rewrite ')) {
          const parts = line.replace('rewrite ', '').split(/\s+/);
          currentRoute.handles.push({
            id: genId(),
            type: 'rewrite',
            enabled: true,
            uri: parts[1] ?? parts[0]
          });
        } else if (line.startsWith('header ')) {
          const rest = line.replace('header ', '').trim();
          const isDelete = rest.startsWith('-');
          const kv = rest.replace(/^-/, '').split(/\s+/);
          currentRoute.handles.push({
            id: genId(),
            type: 'header',
            enabled: true,
            rules: [{ op: isDelete ? 'delete' : 'set', key: kv[0] ?? '', value: kv.slice(1).join(' ') }]
          });
        } else if (line.startsWith('log_append ')) {
          const parts = line.replace('log_append ', '').trim().split(/\s+/);
          if (parts[0]) {
            currentRoute.logAppend = currentRoute.logAppend || [];
            currentRoute.logAppend.push({ key: parts[0], value: parts.slice(1).join(' ') });
          }
        }
      }
    }

    depth += openCount - closeCount;
    if (depth < 0) depth = 0;
    if (matcherDepth !== null && depth < matcherDepth) {
      matcherDepth = null;
      currentMatcherName = null;
    }
    if (handleDepth !== null && depth < handleDepth) {
      handleDepth = null;
      currentHandleBlock = false;
      currentRoute = null;
    }
  }
  return normalizeModules({
    schemaVersion: 1,
    global: { raw: globalRaw },
    upstreams: [],
    sites
  });
}

function buildLineDiff(leftRaw: string, rightRaw: string): DiffRow[] {
  const left = leftRaw.split('\n');
  const right = rightRaw.split('\n');
  const m = left.length;
  const n = right.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0));
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (left[i - 1] === right[j - 1]) dp[i][j] = dp[i - 1][j - 1] + 1;
      else dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
    }
  }
  const ops: Array<{ left: string | null; right: string | null; type: 'same' | 'added' | 'removed' }> = [];
  let i = m;
  let j = n;
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && left[i - 1] === right[j - 1]) {
      ops.push({ left: left[i - 1], right: right[j - 1], type: 'same' });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      ops.push({ left: null, right: right[j - 1], type: 'added' });
      j--;
    } else if (i > 0) {
      ops.push({ left: left[i - 1], right: null, type: 'removed' });
      i--;
    }
  }
  ops.reverse();

  const rows: Array<{ left: string | null; right: string | null; type: 'same' | 'added' | 'removed' | 'changed' }> = [];
  let k = 0;
  while (k < ops.length) {
    const current = ops[k];
    const next = ops[k + 1];
    if (current.type === 'removed' && next?.type === 'added') {
      rows.push({ left: current.left, right: next.right, type: 'changed' });
      k += 2;
      continue;
    }
    rows.push({ left: current.left, right: current.right, type: current.type });
    k += 1;
  }

  let leftLine = 0;
  let rightLine = 0;
  return rows.map((row, index) => {
    if (row.left !== null) leftLine += 1;
    if (row.right !== null) rightLine += 1;
    return {
      ...row,
      leftNo: row.left !== null ? leftLine : null,
      rightNo: row.right !== null ? rightLine : null,
      key: `${index}-${row.type}`
    };
  });
}

function formatCaddyfile(content: string) {
  if (!content.trim()) return content;
  const lines = content.split('\n');
  const out: string[] = [];
  let indent = 0;
  const indentUnit = '  ';

  const stripComment = (line: string) => {
    let inQuote = false;
    let escaped = false;
    let result = '';
    for (const ch of line) {
      if (!escaped && ch === '"') inQuote = !inQuote;
      if (!inQuote && ch === '#') break;
      result += ch;
      escaped = ch === '\\' && !escaped;
      if (ch !== '\\') escaped = false;
    }
    return result;
  };

  const countBraces = (line: string) => {
    let openCount = 0;
    let closeCount = 0;
    let inQuote = false;
    let escaped = false;
    const sanitized = stripComment(line);
    const isBoundary = (ch?: string) => !ch || /\s/.test(ch);
    for (let i = 0; i < sanitized.length; i += 1) {
      const ch = sanitized[i];
      if (!escaped && ch === '"') inQuote = !inQuote;
      if (!inQuote && (ch === '{' || ch === '}')) {
        const prev = sanitized[i - 1];
        const next = sanitized[i + 1];
        if (isBoundary(prev) && isBoundary(next)) {
          if (ch === '{') openCount += 1;
          if (ch === '}') closeCount += 1;
        }
      }
      escaped = ch === '\\' && !escaped;
      if (ch !== '\\') escaped = false;
    }
    return { openCount, closeCount };
  };

  for (const raw of lines) {
    const trimmed = raw.trim();
    if (!trimmed) {
      out.push('');
      continue;
    }
    const { openCount, closeCount } = countBraces(raw);
    const nextIndent = Math.max(indent - closeCount, 0);
    out.push(`${indentUnit.repeat(nextIndent)}${trimmed}`);
    indent = nextIndent + openCount;
  }
  return out.join('\n').trim();
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

function normalizeModules(raw: any): CaddyFormModel {
  const globalRaw = typeof raw?.global?.raw === 'string' ? raw.global.raw : '';
  const normalized: CaddyFormModel = {
    schemaVersion: raw.schemaVersion ?? 1,
    global: { ...(raw.global ?? {}), raw: globalRaw },
    upstreams: (raw.upstreams ?? []).map((u: any) => ({
      name: u.name || `upstream-${genId().slice(0, 6)}`,
      targets: Array.isArray(u.targets) ? u.targets : [],
      lbPolicy: u.lbPolicy ?? 'round_robin',
      healthCheck: u.healthCheck
    })),
    sites: (raw.sites ?? []).map((s: any) => ({
      id: s.id || genId(),
      name: s.name || '未命名站点',
      enabled: s.enabled ?? true,
      domains: Array.isArray(s.domains) ? s.domains : [],
      tls: s.tls ?? { mode: 'auto' },
      imports: s.imports ?? [],
      geoip2Vars: s.geoip2Vars ?? [],
      encode: s.encode ?? [],
      headers: s.headers,
      routes: (s.routes ?? []).map((r: any) => ({
        id: r.id || genId(),
        name: r.name || '未命名路由',
        enabled: r.enabled ?? true,
        match: {
          host: r.match?.host ?? [],
          path: r.match?.path ?? [],
          method: r.match?.method ?? [],
          header: r.match?.header ?? [],
          query: r.match?.query ?? [],
          expression: r.match?.expression ?? ''
        },
        logAppend: r.logAppend ?? [],
        handles: (r.handles ?? []).map((h: any) => ({
          id: h.id || genId(),
          type: h.type || 'reverse_proxy',
          enabled: h.enabled ?? true,
          upstream: h.upstream ?? '',
          lbPolicy: h.lbPolicy ?? 'round_robin',
          healthCheck: h.healthCheck,
          transportProtocol: h.transportProtocol ?? '',
          tlsInsecureSkipVerify: h.tlsInsecureSkipVerify ?? false,
          root: h.root,
          browse: h.browse,
          status: h.status,
          body: h.body,
          to: h.to,
          code: h.code,
          rules: h.rules ?? [],
          uri: h.uri
        }))
      }))
    }))
  };
  return normalized;
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
    const res = await updateCaddyServer(serverFormModel.value);
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
  <div class="h-full overflow-hidden flex flex-col">
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
              class="w-full max-w-72"
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

      <div class="flex-1 min-h-0 flex flex-col gap-4 overflow-hidden">
        <div class="caddy-mode-strip">
          <NRadioGroup :value="pageMode" size="small" @update:value="handleModeChange">
            <NRadioButton
              v-for="option in pageModeOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </NRadioButton>
          </NRadioGroup>
          <div class="text-xs text-gray-500">
            {{ pageModeSummary }}
          </div>
        </div>

        <div v-if="servers.length === 0" class="flex flex-col items-center justify-center p-8 text-gray-400 h-full">
          <div class="text-lg">未找到 Caddy 服务器</div>
          <div class="text-sm mt-2">先添加一个服务器，再开始管理配置。</div>
          <NButton class="mt-4" type="primary" @click="openAddServerModal">添加服务器</NButton>
        </div>

        <NSpin :show="loading" class="flex-1 min-h-0" content-class="h-full min-h-0" v-else>
          <n-alert
            v-if="pageMode === 'quick' && quickValidationErrors.length"
            type="error"
            :show-icon="true"
            class="mb-3"
          >
            {{ quickValidationErrors[0] }}
          </n-alert>

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
          <RawEditorPanel
            v-else-if="pageMode === 'raw'"
            v-model="configContent"
          />
          <ConfigPreviewPanel
            v-else
            :config-content="formattedConfigContent"
          />
        </NSpin>
      </div>
    </NCard>

    <NDrawer v-model:show="showSettingsDrawer" :width="560" placement="right">
      <NDrawerContent title="更多设置" closable>
        <n-space vertical size="large">
          <n-card size="small" :bordered="false">
            <template #header>配置工具</template>
            <div class="flex flex-wrap gap-2">
              <n-button size="small" @click="applyPreset">应用默认模板</n-button>
              <n-button size="small" @click="importRawToStructured">从原始配置解析</n-button>
              <n-button size="small" @click="openHistoryModal" :disabled="!currentServerId">查看历史版本</n-button>
            </div>
          </n-card>

          <n-card size="small" :bordered="false">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <span>全局配置</span>
                <div class="flex items-center gap-2">
                  <n-tag v-if="globalRawChanged" type="warning" size="small" :bordered="false">未保存</n-tag>
                  <n-button size="tiny" secondary @click="restoreGlobalRaw" :disabled="!initialGlobalRaw">
                    恢复已保存
                  </n-button>
                  <n-button size="tiny" @click="openGlobalCompare" :disabled="!formModel.global.raw && !initialGlobalRaw">
                    对比
                  </n-button>
                </div>
              </div>
            </template>

            <n-alert v-if="pageMode !== 'quick'" type="info" :show-icon="true" class="mb-3">
              全局配置仅在“快速配置”模式下可编辑；原始配置模式请直接维护完整 Caddyfile。
            </n-alert>

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
          </n-card>

          <n-card size="small" :bordered="false">
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
          </n-card>
        </n-space>
      </NDrawerContent>
    </NDrawer>

    <NModal v-model:show="showHistoryModal" preset="card" title="配置历史" class="w-[90vw] max-w-3xl">
      <n-data-table
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
      class="w-[90vw] max-w-5xl"
    >
      <div class="flex flex-wrap items-center gap-3 text-xs text-gray-500 mb-3">
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

    <NModal v-model:show="showHistoryCompareModal" preset="card" title="历史配置对比" class="w-[90vw] max-w-5xl">
      <div class="diff-head">
        <div>历史版本</div>
        <div class="flex items-center justify-between">
          <span>当前配置</span>
          <n-switch v-model:value="historyDiffOnly" size="small">
            <template #checked>仅差异</template>
            <template #unchecked>全部</template>
          </n-switch>
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

    <NModal v-model:show="showGlobalCompareModal" preset="card" title="全局配置对比" class="w-[90vw] max-w-4xl">
      <div class="diff-head">
        <div>已保存</div>
        <div class="flex items-center justify-between">
          <span>当前</span>
          <n-switch v-model:value="showGlobalDiffOnly" size="small">
            <template #checked>仅差异</template>
            <template #unchecked>全部</template>
          </n-switch>
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

    <!-- Server Management Modal -->
    <!-- ... modal content ... -->
    <NModal v-model:show="showServerModal" preset="card" :title="serverModalType === 'add' ? '添加服务器' : '编辑服务器'" class="w-500px">
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
        <NFormItem label="凭证" path="token" v-if="serverFormModel.type === 'remote'">
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
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
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
