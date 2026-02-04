<script setup lang="ts">
import { ref, onMounted, computed, watch, onBeforeUnmount, h, nextTick } from 'vue';
import { useMessage, useDialog, NTag, NButton } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { VueMonacoEditor, VueMonacoDiffEditor, loader } from '@guolao/vue-monaco-editor';
import { fetchCaddyServers, fetchCaddyConfig, updateCaddyConfigRaw, updateCaddyConfigStructured, addCaddyServer, updateCaddyServer, deleteCaddyServer, fetchCaddyConfigHistory, fetchCaddyConfigHistoryDetail, rollbackCaddyConfig } from '@/service/api/caddy';
import ConfigPreviewPanel from './components/ConfigPreviewPanel.vue';
import RawEditorPanel from './components/RawEditorPanel.vue';
import StructuredEditorPanel from './components/StructuredEditorPanel.vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import type { CaddyFormModel, Route, RouteMatch, Site } from './types';

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

type ValidationError = {
  id: string;
  message: string;
  siteId?: string;
  routeId?: string;
  tab?: 'basic' | 'routes' | 'advanced';
};

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
const viewMode = ref<'preview' | 'edit'>('preview');
const editMode = ref<'raw' | 'structured'>('raw');
const configContent = ref('');
const showSiteModal = ref(false);
const structuredAvailable = ref(false);
const createEmptyFormModel = (): CaddyFormModel => ({
  schemaVersion: 1,
  global: { raw: '' },
  upstreams: [],
  sites: []
});
const formModel = ref<CaddyFormModel>(createEmptyFormModel());
const activeSiteId = ref<string | null>(null);
const activeSiteTab = ref<'basic' | 'routes' | 'advanced'>('basic');
const focusRouteId = ref<string | null>(null);
const sidebarWidth = ref<number>(Number(localStorage.getItem('logflux:caddy.sidebarWidth')) || 288);
const resizing = ref(false);
let resizeStartX = 0;
let resizeStartWidth = 0;

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
const showGlobalModal = ref(false);
const showGlobalCompareModal = ref(false);
const initialGlobalRaw = ref('');
const globalPreviewExpanded = ref(false);
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

// Computed
const isPreview = computed(() => viewMode.value === 'preview');
const serverOptions = computed(() => 
  servers.value.map(s => ({ label: s.name, value: s.id }))
);
const activeSite = computed(() => formModel.value.sites.find(s => s.id === activeSiteId.value) || null);
const activeSiteTitle = computed(() => (activeSite.value?.name ? `预览 - ${activeSite.value.name}` : '预览'));
const activeSiteModel = computed<Site | null>({
  get() {
    return activeSite.value;
  },
  set(value) {
    if (!value) return;
    const idx = formModel.value.sites.findIndex(s => s.id === value.id);
    if (idx >= 0) formModel.value.sites[idx] = value;
  }
});
const rawSiteContent = computed(() => {
  if (!activeSite.value || !configContent.value) return configContent.value;
  return extractSiteBlock(configContent.value, activeSite.value.domains) || configContent.value;
});
const formattedConfigContent = computed(() => formatCaddyfile(configContent.value));
const formattedRawSiteContent = computed(() => formatCaddyfile(rawSiteContent.value));
const generatedCaddyfile = computed(() => buildCaddyfile(formModel.value));
const currentServer = computed(() => servers.value.find(s => s.id === currentServerId.value) || null);
const structuredReady = computed(() => {
  if (structuredAvailable.value) return true;
  const model = formModel.value;
  if (model.sites?.length) return true;
  if (model.upstreams?.length) return true;
  return Boolean(model.global?.raw?.trim());
});
const validationErrors = computed<ValidationError[]>(() => (editMode.value === 'structured' ? validateStructuredConfig() : []));
const globalRawChanged = computed(
  () => (formModel.value.global?.raw ?? '').trim() !== (initialGlobalRaw.value ?? '').trim()
);
const globalDiffRows = computed<DiffRow[]>(() => {
  const rows = buildLineDiff(initialGlobalRaw.value ?? '', formModel.value.global?.raw ?? '');
  if (!showGlobalDiffOnly.value) return rows;
  return rows.filter(row => row.type !== 'same');
});
const globalPreviewText = computed(() => formatGlobalPreview(formModel.value.global?.raw ?? ''));
const historyDetailFormattedConfig = computed(() => (historyDetail.value ? formatCaddyfile(historyDetail.value.config) : ''));
const historyCompareLeftFormatted = computed(() => formatCaddyfile(historyCompareLeft.value));
const historyCompareRight = computed(() => formattedConfigContent.value);

// Methods
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
    activeSiteId.value = null;
    if (data.modules) {
      try {
        const parsed = JSON.parse(data.modules);
        if (parsed?.sites || parsed?.global) {
          formModel.value = normalizeModules(parsed);
          activeSiteId.value = formModel.value.sites?.[0]?.id || null;
          structuredAvailable.value = true;
        }
      } catch {
        message.warning('结构化配置解析失败，已忽略');
        formModel.value = createEmptyFormModel();
        activeSiteId.value = null;
        structuredAvailable.value = false;
      }
    }
    activeSiteTab.value = 'basic';
    focusRouteId.value = null;
    initialGlobalRaw.value = formModel.value.global?.raw ?? '';
    if (viewMode.value === 'edit' && editMode.value === 'structured') {
      ensureStructuredForEdit();
    }
  }
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
  message.success('配置已保存');
  structuredAvailable.value = false;
  viewMode.value = 'preview';
}

function applyStructuredToRaw() {
  configContent.value = generatedCaddyfile.value;
  editMode.value = 'raw';
  viewMode.value = 'edit';
}

function applyStructuredParsed(parsed: CaddyFormModel, notify?: boolean) {
  formModel.value = parsed;
  activeSiteId.value = parsed.sites[0]?.id || null;
  structuredAvailable.value = true;
  editMode.value = 'structured';
  viewMode.value = 'edit';
  initialGlobalRaw.value = parsed.global?.raw ?? '';
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

function ensureStructuredForEdit() {
  if (structuredReady.value) return;
  if (!configContent.value.trim()) return;
  const parsed = parseCaddyfileToModules(configContent.value);
  if (parsed.sites.length === 0 && !parsed.global?.raw) return;
  applyStructuredParsed(parsed, false);
}

async function copyText(content: string) {
  try {
    await navigator.clipboard.writeText(content);
    message.success('已复制');
  } catch {
    message.error('复制失败');
  }
}

function downloadText(filename: string, content: string) {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function startResize(event: MouseEvent) {
  resizing.value = true;
  resizeStartX = event.clientX;
  resizeStartWidth = sidebarWidth.value;
  window.addEventListener('mousemove', handleResize);
  window.addEventListener('mouseup', stopResize);
}

function handleResize(event: MouseEvent) {
  if (!resizing.value) return;
  const delta = event.clientX - resizeStartX;
  const next = Math.min(420, Math.max(220, resizeStartWidth + delta));
  sidebarWidth.value = next;
}

function stopResize() {
  if (!resizing.value) return;
  resizing.value = false;
  localStorage.setItem('logflux:caddy.sidebarWidth', String(sidebarWidth.value));
  window.removeEventListener('mousemove', handleResize);
  window.removeEventListener('mouseup', stopResize);
}

function validateStructuredConfig(): ValidationError[] {
  const errors: ValidationError[] = [];
  const pushError = (
    message: string,
    siteId?: string,
    routeId?: string,
    tab?: 'basic' | 'routes' | 'advanced'
  ) => {
    errors.push({
      id: `${siteId ?? 'global'}-${routeId ?? 'none'}-${errors.length}`,
      message,
      siteId,
      routeId,
      tab
    });
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
  const enabledSites = formModel.value.sites.filter(s => s.enabled);
  const hasSites = enabledSites.length > 0;
  const hasGlobalRaw = !!formModel.value.global?.raw?.trim();
  if (!hasSites && !hasGlobalRaw) {
    pushError('至少需要一个站点或全局配置');
  }
  const upstreamNames = new Set<string>();
  for (const up of formModel.value.upstreams) {
    if (!up.name) pushError('上游名称不能为空');
    if (upstreamNames.has(up.name)) pushError(`上游名称重复: ${up.name}`);
    upstreamNames.add(up.name);
    if (up.targets.length === 0) pushError(`上游 ${up.name} 至少配置一个目标`);
  }
  for (const site of formModel.value.sites) {
    if (!site.enabled) continue;
    if (!site.name) pushError('站点名称不能为空', site.id, undefined, 'basic');
    if (site.domains.length === 0) pushError(`站点 ${site.name || site.id} 至少配置一个域名`, site.id, undefined, 'basic');
    if (site.enabled && site.routes.length === 0) {
      pushError(`站点 ${site.name || site.id} 至少配置一个路由`, site.id, undefined, 'routes');
    }
    const invalidDomains = site.domains.filter(d => d && !(domainRe.test(d) || portOnlyRe.test(d)));
    if (invalidDomains.length) pushError(`站点 ${site.name || site.id} 域名格式不合法: ${invalidDomains.join(', ')}`, site.id, undefined, 'basic');
    if (site.tls?.mode === 'manual' && (!site.tls.certFile || !site.tls.keyFile)) {
      pushError(`站点 ${site.name || site.id} TLS 手动模式需填写证书和私钥`, site.id, undefined, 'basic');
    }
    for (const route of site.routes) {
      if (!route.enabled) continue;
      if (!route.name) pushError(`站点 ${site.name || site.id} 有未命名路由`, site.id, route.id, 'routes');
      if (route.handles.length === 0) pushError(`路由 ${route.name || route.id} 至少一个 Handler`, site.id, route.id, 'routes');
      if (route.handles.every(h => !h.enabled)) {
        pushError(`路由 ${route.name || route.id} 至少启用一个 Handler`, site.id, route.id, 'routes');
      }
      const invalidPaths = route.match.path.filter(p => p && !isValidPathPattern(p));
      if (invalidPaths.length) pushError(`路由 ${route.name || route.id} Path 格式不合法: ${invalidPaths.join(', ')}`, site.id, route.id, 'routes');
      const invalidMethods = route.match.method.filter(m => m && !methodAllowList.includes(m.toUpperCase()));
      if (invalidMethods.length) pushError(`路由 ${route.name || route.id} Method 非法: ${invalidMethods.join(', ')}`, site.id, route.id, 'routes');
      for (const handle of route.handles) {
        if (!handle.enabled) continue;
        if (handle.type === 'reverse_proxy' && !handle.upstream) {
          pushError(`路由 ${route.name || route.id} 的 reverse_proxy 未选择上游`, site.id, route.id, 'routes');
        }
      }
    }
  }
  return errors;
}

async function saveStructuredConfig() {
  if (!currentServerId.value) return;
  const content = generatedCaddyfile.value;
  if (!content) {
    message.error('结构化配置为空，无法保存');
    return;
  }
  const errors = validateStructuredConfig();
  if (errors.length > 0) {
    message.error(`校验失败：${errors[0].message}`);
    return;
  }
  saving.value = true;
  const modules = JSON.stringify(formModel.value);
  const { error } = await updateCaddyConfigStructured(currentServerId.value, content, modules);
  saving.value = false;
  if (error) {
    message.error('保存配置失败');
    return;
  }
  message.success('配置已保存');
  configContent.value = content;
  initialGlobalRaw.value = formModel.value.global?.raw ?? '';
  structuredAvailable.value = true;
  viewMode.value = 'preview';
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
              type: 'reverse_proxy',
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
            type: 'reverse_proxy',
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

function formatGlobalPreview(raw: string) {
  const trimmed = raw.trim();
  if (!trimmed) return '';
  const lines = trimmed.split('\n');
  const indents = lines
    .filter(line => line.trim().length > 0)
    .map(line => (line.match(/^[\t ]*/)?.[0]?.length ?? 0));
  const minIndent = indents.length ? Math.min(...indents) : 0;
  if (minIndent === 0) return trimmed;
  return lines.map(line => line.slice(minIndent)).join('\n');
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


function addSite() {
  const id = genId();
  const site: Site = {
    id,
    name: '新站点',
    enabled: true,
    domains: [],
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
            upstream: '',
            lbPolicy: 'round_robin',
            tlsInsecureSkipVerify: false
          }
        ]
      }
    ]
  };
  formModel.value.sites.push(site);
  activeSiteId.value = id;
}

function openPreviewModal() {
  if (!activeSiteId.value) {
    message.warning('请先选择一个站点');
    return;
  }
  showSiteModal.value = true;
}

function applyPreset() {
  confirmOverwriteStructured('应用默认模板', () => {
    const upstreamName = 'default-upstream';
    structuredAvailable.value = true;
    editMode.value = 'structured';
    viewMode.value = 'edit';
    formModel.value.schemaVersion = 1;
    formModel.value.upstreams = [
      { name: upstreamName, targets: ['localhost:8080'], lbPolicy: 'round_robin' }
    ];
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
                upstream: upstreamName,
                lbPolicy: 'round_robin',
                tlsInsecureSkipVerify: false
              }
            ]
          }
        ]
      }
    ];
    activeSiteId.value = siteId;
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

function extractSiteBlock(content: string, domains: string[]): string {
  if (!content || domains.length === 0) return '';
  const lines = content.split('\n');
  let depth = 0;
  let capturing = false;
  const result: string[] = [];
  const domainSet = new Set(domains.filter(Boolean));
  for (const raw of lines) {
    const line = raw;
    const trimmed = raw.replace(/#.*/, '').trim();
    const openCount = (trimmed.match(/{/g) || []).length;
    const closeCount = (trimmed.match(/}/g) || []).length;
    if (!capturing && depth === 0 && trimmed.includes('{') && !trimmed.startsWith('{')) {
      const before = trimmed.split('{')[0].trim();
      if (before) {
        const tokens = before.replace(/,/g, ' ').split(/\s+/).filter(Boolean);
        if (tokens.some(t => domainSet.has(t))) {
          capturing = true;
        }
      }
    }
    if (capturing) result.push(line);
    depth += openCount - closeCount;
    if (capturing && depth === 0 && closeCount > 0) break;
  }
  return result.join('\n').trim();
}

function duplicateSite(id: string) {
  const target = formModel.value.sites.find(s => s.id === id);
  if (!target) return;
  const clone = JSON.parse(JSON.stringify(target)) as Site;
  clone.id = genId();
  clone.name = `${clone.name}-copy`;
  formModel.value.sites.push(clone);
  activeSiteId.value = clone.id;
}

function removeSite(id: string) {
  const idx = formModel.value.sites.findIndex(s => s.id === id);
  if (idx >= 0) formModel.value.sites.splice(idx, 1);
  if (activeSiteId.value === id) {
    activeSiteId.value = formModel.value.sites[0]?.id || null;
  }
}

function moveSite(id: string, direction: 'up' | 'down') {
  const idx = formModel.value.sites.findIndex(s => s.id === id);
  if (idx < 0) return;
  const next = direction === 'up' ? idx - 1 : idx + 1;
  if (next < 0 || next >= formModel.value.sites.length) return;
  const temp = formModel.value.sites[idx];
  formModel.value.sites[idx] = formModel.value.sites[next];
  formModel.value.sites[next] = temp;
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

function openGlobalCompare() {
  showGlobalCompareModal.value = true;
}

function openGlobalModal() {
  showGlobalModal.value = true;
}

function toggleGlobalPreview() {
  globalPreviewExpanded.value = !globalPreviewExpanded.value;
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


async function focusValidationError(item: ValidationError) {
  if (item.siteId) {
    activeSiteId.value = item.siteId;
  }
  if (item.tab) {
    activeSiteTab.value = item.tab;
  }
  if (item.routeId) {
    focusRouteId.value = null;
    await nextTick();
    focusRouteId.value = item.routeId;
  }
}

function formatHistoryAction(action: string) {
  return action === 'rollback' ? '回滚' : '更新';
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
  }
});

onMounted(() => {
  getServers();
});

watch([viewMode, editMode], ([nextView, nextEdit]) => {
  if (nextView === 'edit' && nextEdit === 'structured') {
    ensureStructuredForEdit();
  }
});

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleResize);
  window.removeEventListener('mouseup', stopResize);
});
</script>

<template>
  <div class="h-full overflow-hidden flex flex-col">
    <NCard 
      class="h-full card-wrapper" 
      :content-style="{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }"
    >
      <template #header>
        <div class="caddy-header">
          <div class="caddy-header-title">
            <div class="caddy-title-text">Caddy配置</div>
            <div class="caddy-title-sub">服务器与配置管理</div>
          </div>
          <div class="caddy-header-panels">
            <div class="caddy-header-panel">
              <div class="caddy-panel-label">服务器</div>
              <div class="caddy-panel-controls">
                <NSelect
                  v-model:value="currentServerId"
                  :options="serverOptions"
                  placeholder="选择服务器"
                  class="w-48"
                  size="small"
                />
                <NButton size="small" @click="openAddServerModal">
                  <div class="flex items-center gap-1">
                    <SvgIcon icon="carbon:add" class="caddy-icon" />
                    <span>新增</span>
                  </div>
                </NButton>
                <NButton size="small" :disabled="!currentServerId" @click="openEditServerModal">
                  <div class="flex items-center gap-1">
                    <SvgIcon icon="carbon:edit" class="caddy-icon" />
                    <span>编辑</span>
                  </div>
                </NButton>
                <NButton size="small" :disabled="!currentServerId" @click="handleDeleteServer" type="error" ghost>
                  <div class="flex items-center gap-1">
                    <SvgIcon icon="carbon:trash-can" class="caddy-icon" />
                    <span>删除</span>
                  </div>
                </NButton>
                <NButton size="small" :disabled="!currentServerId" @click="openHistoryModal">
                  <div class="flex items-center gap-1">
                    <SvgIcon icon="carbon:time" class="caddy-icon" />
                    <span>历史版本</span>
                  </div>
                </NButton>
              </div>
            </div>
            <div class="caddy-header-panel caddy-header-panel--flat">
              <div class="caddy-panel-label">模式</div>
              <div class="caddy-panel-controls">
                <NRadioGroup v-model:value="viewMode" size="small">
                  <NRadioButton value="preview">预览模式</NRadioButton>
                  <NRadioButton value="edit">编辑模式</NRadioButton>
                </NRadioGroup>
                <NRadioGroup v-if="viewMode === 'edit'" v-model:value="editMode" size="small">
                  <NRadioButton value="raw">原始编辑</NRadioButton>
                  <NRadioButton value="structured">结构化编辑</NRadioButton>
                </NRadioGroup>
              </div>
            </div>
          </div>
          <div class="caddy-header-save">
            <NTooltip v-if="viewMode === 'edit' && editMode === 'structured'">
              <template #trigger>
                <NButton
                  size="small"
                  secondary
                  circle
                  :disabled="!currentServerId"
                  @click="applyStructuredToRaw"
                >
                  <SvgIcon icon="carbon:direction-straight-right" class="caddy-icon" />
                </NButton>
              </template>
              生成到原始配置
            </NTooltip>
            <NTooltip v-if="viewMode === 'edit' && editMode === 'raw'">
              <template #trigger>
                <NButton 
                  type="primary" 
                  size="small" 
                  circle
                  :loading="saving"
                  :disabled="!currentServerId"
                  @click="saveRawConfig"
                >
                  <SvgIcon icon="carbon:save" class="caddy-icon" />
                </NButton>
              </template>
              保存原始配置
            </NTooltip>
            <NTooltip v-if="viewMode === 'edit' && editMode === 'structured'">
              <template #trigger>
                <NButton
                  size="small"
                  type="primary"
                  circle
                  :loading="saving"
                  :disabled="!currentServerId"
                  @click="saveStructuredConfig"
                >
                  <SvgIcon icon="carbon:save-series" class="caddy-icon" />
                </NButton>
              </template>
              保存结构化配置
            </NTooltip>
          </div>
        </div>
      </template>
      
      <div class="flex-1 min-h-0 flex flex-col gap-4 overflow-hidden">
        <div v-if="servers.length === 0" class="flex flex-col items-center justify-center p-8 text-gray-400 h-full">
           <div class="text-lg">未找到 Caddy 服务器</div>
           <div class="text-sm mt-2">请点击上方“+”按钮添加服务器</div>
        </div>
        
        <NSpin :show="loading" class="h-full" content-class="h-full" v-else>
          <ConfigPreviewPanel
            v-if="viewMode === 'preview'"
            :config-content="formattedConfigContent"
          />
          <RawEditorPanel v-else-if="editMode === 'raw'" v-model="configContent" />
          <StructuredEditorPanel
            v-else
            v-model:active-site-id="activeSiteId"
            v-model:active-site="activeSiteModel"
            v-model:active-tab="activeSiteTab"
            :form-model="formModel"
            :focus-route-id="focusRouteId"
            :sidebar-width="sidebarWidth"
            :structured-available="structuredReady"
            :validation-errors="validationErrors"
            :global-raw-changed="globalRawChanged"
            :global-preview-expanded="globalPreviewExpanded"
            :global-preview-text="globalPreviewText"
            :on-apply-preset="applyPreset"
            :on-import-raw-to-structured="importRawToStructured"
            :on-open-preview-modal="openPreviewModal"
            :on-toggle-global-preview="toggleGlobalPreview"
            :on-open-global-modal="openGlobalModal"
            :on-focus-validation-error="focusValidationError"
            :on-start-resize="startResize"
            :on-add-site="addSite"
            :on-duplicate-site="duplicateSite"
            :on-remove-site="removeSite"
            :on-move-site="moveSite"
          />
        </NSpin>
      </div>
    </NCard>

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

    <NModal v-model:show="showGlobalModal" preset="card" title="全局配置" class="w-[90vw] max-w-4xl">
      <div class="flex items-center justify-between mb-3">
        <div class="text-sm text-gray-500">原样保留最外层 options 块</div>
        <n-space>
          <n-button size="tiny" secondary @click="restoreGlobalRaw" :disabled="!initialGlobalRaw">
            恢复已保存
          </n-button>
          <n-button size="tiny" @click="openGlobalCompare" :disabled="!formModel.global.raw && !initialGlobalRaw">
            对比
          </n-button>
        </n-space>
      </div>
      <div class="relative h-[45vh]">
        <VueMonacoEditor
          v-model:value="formModel.global.raw"
          language="shell"
          theme="vs"
          :options="{
            automaticLayout: true,
            fixedOverflowWidgets: true,
            readOnly: isPreview,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            wordWrap: 'on'
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

    <NModal v-model:show="showSiteModal" preset="card" :title="activeSiteTitle" class="w-[90vw] max-w-5xl">
      <div class="h-[75vh] flex flex-col">
        <div class="flex gap-2 mb-2">
          <n-button size="tiny" @click="copyText(formattedRawSiteContent)">复制</n-button>
          <n-button
            size="tiny"
            @click="downloadText(`Caddyfile-${currentServer?.name || currentServer?.id || 'server'}`, formattedRawSiteContent)"
          >
            下载
          </n-button>
        </div>
        <div class="relative flex-1 min-h-0">
          <VueMonacoEditor
            :value="formattedRawSiteContent"
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
        <div class="flex justify-end gap-2 mt-2">
          <n-button @click="showSiteModal = false">关闭</n-button>
        </div>
      </div>
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

.caddy-header {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 16px;
  align-items: start;
}

.caddy-header-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 2px;
}

.caddy-title-text {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.2;
}

.caddy-title-sub {
  font-size: 12px;
  color: #64748b;
}

.caddy-header-panels {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.caddy-header-panel {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
}

.caddy-panel-label {
  font-size: 12px;
  color: #64748b;
  min-width: 52px;
}

.caddy-panel-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.caddy-header-save {
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  padding-top: 8px;
  gap: 8px;
}

.caddy-icon {
  display: inline-block;
  font-size: 16px;
  line-height: 1;
  vertical-align: middle;
}

@media (max-width: 900px) {
  .caddy-header {
    grid-template-columns: 1fr;
  }
  .caddy-header-panel {
    flex-direction: column;
    align-items: flex-start;
  }
  .caddy-panel-label {
    min-width: auto;
  }
  .caddy-header-save {
    justify-content: flex-start;
    padding-top: 0;
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
