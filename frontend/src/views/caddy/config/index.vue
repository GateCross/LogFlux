<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useMessage, useDialog } from 'naive-ui';
import { VueMonacoEditor, loader } from '@guolao/vue-monaco-editor';
import { fetchCaddyServers, fetchCaddyConfig, updateCaddyConfig, addCaddyServer, updateCaddyServer, deleteCaddyServer } from '@/service/api/caddy';
import SiteListPanel from './components/SiteListPanel.vue';
import SiteEditorPanel from './components/SiteEditorPanel.vue';
import UpstreamManager from './components/UpstreamManager.vue';
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

const message = useMessage();
const dialog = useDialog();
const loading = ref(false);
const saving = ref(false);
const servers = ref<CaddyServer[]>([]);
const currentServerId = ref<number | null>(null);
const mode = ref<'preview' | 'edit'>('preview');
const configMode = ref<'raw' | 'structured'>('raw');
const configContent = ref('');
const showSiteModal = ref(false);
const formModel = ref<CaddyFormModel>({
  schemaVersion: 1,
  global: {},
  upstreams: [],
  sites: []
});
const activeSiteId = ref<string | null>(null);

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
const isPreview = computed(() => mode.value === 'preview');
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
const previewModel = computed(() => {
  if (!activeSiteId.value) return formModel.value;
  const site = formModel.value.sites.find(s => s.id === activeSiteId.value);
  if (!site) return formModel.value;
  return { ...formModel.value, sites: [site] };
});
const generatedCaddyfile = computed(() => buildCaddyfile(formModel.value));
const previewCaddyfile = computed(() => buildCaddyfile(previewModel.value, { includeDisabled: true }));
const previewCaddyJSON = computed(() => JSON.stringify(previewModel.value, null, 2));
const currentServer = computed(() => servers.value.find(s => s.id === currentServerId.value) || null);

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
    let structuredLoaded = false;
    if (data.modules) {
      try {
        const parsed = JSON.parse(data.modules);
        if (parsed?.sites) {
          formModel.value = normalizeModules(parsed);
          activeSiteId.value = formModel.value.sites?.[0]?.id || null;
          configMode.value = 'structured';
          structuredLoaded = true;
        }
      } catch {
        message.warning('结构化配置解析失败，已忽略');
      }
    }
    if (!structuredLoaded) {
      const parsed = parseCaddyfileToModules(configContent.value);
      if (parsed.sites.length > 0) {
        formModel.value = parsed;
        activeSiteId.value = parsed.sites[0].id;
        configMode.value = 'structured';
      } else {
        configMode.value = 'raw';
      }
    }
  }
}

async function handleSaveConfig() {
  if (!currentServerId.value) return;

  saving.value = true;
  let modules: string | undefined;
  const parsed = parseCaddyfileToModules(configContent.value);
  if (parsed.sites.length > 0) {
    modules = JSON.stringify(parsed);
  }
  const { error } = await updateCaddyConfig(currentServerId.value, configContent.value, modules);
  saving.value = false;

  if (error) {
    message.error('保存配置失败');
    return;
  }
  message.success('配置已保存');
  mode.value = 'preview';
}

function applyStructuredToRaw() {
  configContent.value = generatedCaddyfile.value;
  configMode.value = 'raw';
  mode.value = 'edit';
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

function validateStructuredConfig(): string[] {
  const errors: string[] = [];
  const domainRe = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
  const portOnlyRe = /^:\d+$/;
  const methodAllowList = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];
  if (formModel.value.sites.length === 0) {
    errors.push('至少需要一个站点');
  }
  const upstreamNames = new Set<string>();
  for (const up of formModel.value.upstreams) {
    if (!up.name) errors.push('上游名称不能为空');
    if (upstreamNames.has(up.name)) errors.push(`上游名称重复: ${up.name}`);
    upstreamNames.add(up.name);
    if (up.targets.length === 0) errors.push(`上游 ${up.name} 至少配置一个目标`);
  }
  for (const site of formModel.value.sites) {
    if (!site.name) errors.push('站点名称不能为空');
    if (site.domains.length === 0) errors.push(`站点 ${site.name || site.id} 至少配置一个域名`);
    const invalidDomains = site.domains.filter(d => d && !(domainRe.test(d) || portOnlyRe.test(d)));
    if (invalidDomains.length) errors.push(`站点 ${site.name || site.id} 域名格式不合法: ${invalidDomains.join(', ')}`);
    if (site.tls?.mode === 'manual' && (!site.tls.certFile || !site.tls.keyFile)) {
      errors.push(`站点 ${site.name || site.id} TLS 手动模式需填写证书和私钥`);
    }
    for (const route of site.routes) {
      if (!route.name) errors.push(`站点 ${site.name || site.id} 有未命名路由`);
      if (route.handles.length === 0) errors.push(`路由 ${route.name || route.id} 至少一个 Handler`);
      const invalidPaths = route.match.path.filter(p => p && !p.startsWith('/'));
      if (invalidPaths.length) errors.push(`路由 ${route.name || route.id} Path 需以 / 开头: ${invalidPaths.join(', ')}`);
      const invalidMethods = route.match.method.filter(m => m && !methodAllowList.includes(m.toUpperCase()));
      if (invalidMethods.length) errors.push(`路由 ${route.name || route.id} Method 非法: ${invalidMethods.join(', ')}`);
      for (const handle of route.handles) {
        if (handle.type === 'reverse_proxy' && !handle.upstream) {
          errors.push(`路由 ${route.name || route.id} 的 reverse_proxy 未选择上游`);
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
    message.error(`校验失败：${errors[0]}`);
    return;
  }
  saving.value = true;
  const modules = JSON.stringify(formModel.value);
  const { error } = await updateCaddyConfig(currentServerId.value, content, modules);
  saving.value = false;
  if (error) {
    message.error('保存配置失败');
    return;
  }
  message.success('配置已保存');
  mode.value = 'preview';
}

function buildCaddyfile(model: CaddyFormModel, options?: { includeDisabled?: boolean }): string {
  const lines: string[] = [];
  if (!model.sites || model.sites.length === 0) {
    return '# No sites defined';
  }
  const upstreamMap = new Map(model.upstreams.map(u => [u.name, u]));
  const includeDisabled = options?.includeDisabled ?? false;
  for (const site of model.sites.filter(s => includeDisabled || s.enabled)) {
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
      } else if (site.tls.mode === 'auto') {
        lines.push(`  tls`);
      }
    }
    for (const route of site.routes.filter(r => includeDisabled || r.enabled)) {
      const matcherName = `@m_${route.id.slice(0, 6)}`;
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
      if (matcherLines.length) {
        lines.push(`  ${matcherName} {`);
        matcherLines.forEach(l => lines.push(`    ${l}`));
        lines.push(`  }`);
        lines.push(`  handle ${matcherName} {`);
      } else {
        lines.push(`  handle {`);
      }

      for (const h of route.handles.filter(hd => includeDisabled || hd.enabled)) {
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
            else lines.push(`    header ${r.key} ${r.value ?? ''}`.trim());
          }
        } else if (h.type === 'rewrite') {
          lines.push(`    rewrite * ${h.uri ?? '/'}`);
        }
      }
      if (route.logAppend?.length) {
        for (const item of route.logAppend) {
          if (!item.key) continue;
          lines.push(`    log_append ${item.key} ${item.value ?? ''}`.trim());
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

function buildCaddyJSON(model: CaddyFormModel, options?: { includeDisabled?: boolean }) {
  const upstreamMap = new Map(model.upstreams.map(u => [u.name, u]));
  const includeDisabled = options?.includeDisabled ?? false;
  const routes = model.sites
    .filter(s => includeDisabled || s.enabled)
    .flatMap(site =>
      site.routes
        .filter(r => includeDisabled || r.enabled)
        .map(r => {
          const match: Record<string, any> = {};
          if (r.match.host.length) match.host = r.match.host;
          if (r.match.path.length) match.path = r.match.path;
          if (r.match.method.length) match.method = r.match.method;
          if (r.match.header.length) {
            match.header = Object.fromEntries(r.match.header.map(h => [h.key, [h.value]]));
          }
          if (r.match.query.length) {
            match.query = Object.fromEntries(r.match.query.map(q => [q.key, [q.value]]));
          }
          if (r.match.expression) {
            match.expression = [r.match.expression];
          }
          const handlers = r.handles.filter(h => includeDisabled || h.enabled).map(h => {
            if (h.type === 'reverse_proxy') {
              const up = h.upstream ? upstreamMap.get(h.upstream) : undefined;
              const rawTargets = h.upstream ? h.upstream.split(/\s+/).filter(Boolean) : [];
              const transport = h.transportProtocol || (h.tlsInsecureSkipVerify ? 'http' : undefined);
              return {
                handler: 'reverse_proxy',
                upstreams: (up?.targets?.length ? up.targets : rawTargets.length ? rawTargets : ['localhost:8080']).map(t => ({ dial: t })),
                lb_policy: h.lbPolicy,
                transport: transport
                  ? {
                      protocol: transport,
                      tls: h.tlsInsecureSkipVerify ? { insecure_skip_verify: true } : undefined
                    }
                  : undefined
              };
            }
            if (h.type === 'file_server') {
              return { handler: 'file_server', root: h.root, browse: h.browse ? {} : undefined };
            }
            if (h.type === 'respond') {
              return { handler: 'respond', status_code: h.status ?? 200, body: h.body ?? '' };
            }
            if (h.type === 'redirect') {
              return {
                handler: 'static_response',
                status_code: h.code ?? 302,
                headers: { Location: [h.to ?? '/'] }
              };
            }
            if (h.type === 'header') {
              return {
                handler: 'headers',
                header: (h.rules || []).reduce((acc: Record<string, string[]>, item) => {
                  if (item.op === 'delete') acc[item.key] = [];
                  else acc[item.key] = [item.value ?? ''];
                  return acc;
                }, {})
              };
            }
            if (h.type === 'rewrite') {
              return { handler: 'rewrite', uri: h.uri ?? '/' };
            }
            return { handler: h.type };
          });
          const handleList = site.encode?.length
            ? [
                {
                  handler: 'encode',
                  encodings: site.encode.reduce((acc: Record<string, any>, name) => {
                    acc[name] = {};
                    return acc;
                  }, {})
                },
                ...handlers
              ]
            : handlers;
          const route: Record<string, any> = { handle: handleList };
          if (Object.keys(match).length) route.match = [match];
          return route;
        })
    );

  return {
    apps: {
      http: {
        servers: {
          srv0: {
            listen: [':80'],
            routes
          }
        }
      }
    }
  };
}

function genId() {
  return (crypto as any).randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function parseCaddyfileToModules(content: string): CaddyFormModel {
  const sites: Site[] = [];
  const matchers: Record<string, RouteMatch> = {};
  const lines = content.split('\n');
  let depth = 0;
  const domainRe = /^(\*\.)?([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-]+$/;
  const ipv4Re = /^(?:\d{1,3}\.){3}\d{1,3}$/;
  function isSiteToken(token: string) {
    if (!token) return false;
    if (token.startsWith('(') || token.endsWith(')') || token.includes('(') || token.includes(')')) return false;
    if (token.startsWith(':') && /^\:\d+$/.test(token)) return true;
    const [host, port] = token.split(':');
    if (port) {
      if (!/^\d+$/.test(port)) return false;
      return domainRe.test(host) || ipv4Re.test(host) || host === 'localhost';
    }
    return domainRe.test(token);
  }
  let currentSite: Site | null = null;
  let currentRoute: Route | null = null;
  let currentHandleBlock = false;
  let currentMatcherName: string | null = null;
  let reverseProxyDepth: number | null = null;
  let currentReverseProxy: any | null = null;

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
    if (currentMatcherName && depth === 0) currentMatcherName = null;
    if (currentHandleBlock && closeCount > 0 && line.includes('}')) currentHandleBlock = false;
  }
  return normalizeModules({
    schemaVersion: 1,
    global: {},
    upstreams: [],
    sites
  });
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
  const upstreamName = 'default-upstream';
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
}

const importInputRef = ref<HTMLInputElement | null>(null);

function normalizeModules(raw: any): CaddyFormModel {
  const normalized: CaddyFormModel = {
    schemaVersion: raw.schemaVersion ?? 1,
    global: raw.global ?? {},
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

function exportModules() {
  const content = JSON.stringify(formModel.value, null, 2);
  const serverTag = currentServer.value?.name || `server-${currentServerId.value || 'unknown'}`;
  downloadText(`caddy-modules-${serverTag}.json`, content);
}

function triggerImport() {
  importInputRef.value?.click();
}

function onImportChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => {
    try {
      const parsed = JSON.parse(String(reader.result || ''));
      if (!parsed?.sites) {
        message.error('结构化文件不包含 sites 字段');
        return;
      }
      if (parsed.schemaVersion && parsed.schemaVersion !== 1) {
        message.warning(`结构化版本不匹配：当前 v1，导入 v${parsed.schemaVersion}`);
      }
      formModel.value = normalizeModules(parsed);
      activeSiteId.value = formModel.value.sites?.[0]?.id || null;
      configMode.value = 'structured';
      message.success('导入成功');
    } catch {
      message.error('导入失败：JSON 格式不正确');
    }
  };
  reader.readAsText(file);
  input.value = '';
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

// Watchers
watch(currentServerId, () => {
  if (currentServerId.value) {
    getConfig();
  }
});

watch(configMode, () => {
  if (configMode.value === 'structured' && formModel.value.sites.length === 0 && configContent.value) {
    const parsed = parseCaddyfileToModules(configContent.value);
    if (parsed.sites.length > 0) {
      formModel.value = parsed;
      activeSiteId.value = parsed.sites[0].id;
    }
  }
});

onMounted(() => {
  getServers();
});
</script>

<template>
  <div class="h-full overflow-hidden flex flex-col">
    <NCard 
      title="Caddy配置" 
      class="h-full card-wrapper" 
      :content-style="{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }"
    >
      <template #header-extra>
        <div class="flex flex-wrap items-center gap-2 max-w-full">
           <NSelect
              v-model:value="currentServerId"
              :options="serverOptions"
              placeholder="选择服务器"
              class="w-48"
              size="small"
           />
           <div class="flex items-center gap-2">
             <NButton size="small" @click="openAddServerModal">
               <div class="flex items-center gap-1">
                 <span class="i-carbon-add" />
                 <span>新增</span>
               </div>
             </NButton>
             <NButton size="small" :disabled="!currentServerId" @click="openEditServerModal">
               <div class="flex items-center gap-1">
                 <span class="i-carbon-edit" />
                 <span>编辑</span>
               </div>
             </NButton>
             <NButton size="small" :disabled="!currentServerId" @click="handleDeleteServer" type="error" ghost>
               <div class="flex items-center gap-1">
                 <span class="i-carbon-trash-can" />
                 <span>删除</span>
               </div>
             </NButton>
           </div>
           
           <div class="w-1px h-4 bg-gray-200 mx-2"></div>

           <NRadioGroup v-model:value="mode" size="small">
             <NRadioButton value="preview">预览</NRadioButton>
             <NRadioButton value="edit">编辑</NRadioButton>
           </NRadioGroup>
           <NRadioGroup v-model:value="configMode" size="small" class="ml-2">
             <NRadioButton value="raw">原始配置</NRadioButton>
             <NRadioButton value="structured">结构化编辑</NRadioButton>
           </NRadioGroup>
           <NButton 
             v-if="!isPreview" 
             type="primary" 
             size="small" 
             :loading="saving"
             :disabled="!currentServerId || configMode === 'structured'"
             @click="handleSaveConfig"
           >
             保存配置
           </NButton>
           <NButton
             v-if="configMode === 'structured'"
             size="small"
             type="primary"
             :disabled="!currentServerId"
             @click="applyStructuredToRaw"
           >
             生成到原始配置
           </NButton>
           <NPopover v-if="configMode === 'structured' && !isPreview" trigger="hover">
             <template #trigger>
               <NButton
                 size="small"
                 type="primary"
                 :loading="saving"
                 :disabled="!currentServerId"
                 @click="saveStructuredConfig"
               >
                 直接保存
               </NButton>
             </template>
             <div class="text-xs">保存前会进行结构化校验</div>
           </NPopover>
        </div>
      </template>
      
      <div class="flex-1 min-h-0 flex flex-col gap-4 overflow-hidden">
        <div v-if="servers.length === 0" class="flex flex-col items-center justify-center p-8 text-gray-400 h-full">
           <div class="text-lg">未找到 Caddy 服务器</div>
           <div class="text-sm mt-2">请点击上方“+”按钮添加服务器</div>
        </div>
        
        <NSpin :show="loading" class="h-full" content-class="h-full" v-else>
          <div class="h-full relative" v-if="configMode === 'raw'">
            <VueMonacoEditor
              v-model:value="configContent"
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
          <div v-else class="h-full flex flex-col lg:flex-row gap-4 overflow-hidden min-w-0">
            <div class="w-full lg:w-72 flex-shrink-0 min-w-0">
              <SiteListPanel
                class="h-full"
                :sites="formModel.sites"
                :active-id="activeSiteId"
                @select="activeSiteId = $event"
                @add="addSite"
                @duplicate="duplicateSite"
                @remove="removeSite"
                @move="moveSite"
              />
            </div>
            <div class="flex-1 min-w-0 flex flex-col gap-3 overflow-auto">
              <div class="flex flex-wrap gap-2 items-center">
                <n-button size="small" @click="applyPreset">应用默认模板</n-button>
                <n-button size="small" @click="exportModules">导出结构化 JSON</n-button>
                <n-button size="small" @click="triggerImport">导入结构化 JSON</n-button>
                <n-button size="small" @click="openPreviewModal">预览 Caddyfile</n-button>
                <input ref="importInputRef" type="file" accept="application/json" class="hidden" @change="onImportChange" />
              </div>
              <SiteEditorPanel v-model:site="activeSiteModel" />
              <n-collapse class="mt-2">
                <n-collapse-item title="上游池管理" name="upstreams">
                  <UpstreamManager :upstreams="formModel.upstreams" />
                </n-collapse-item>
                <n-collapse-item title="结构化预览" name="preview">
                  <n-tabs type="line" size="small">
                    <n-tab-pane name="caddyfile" tab="Caddyfile">
                      <div class="flex gap-2 mb-2">
                        <n-button size="tiny" @click="copyText(previewCaddyfile)">复制</n-button>
                        <n-button
                          size="tiny"
                          @click="downloadText(`Caddyfile-structured-${currentServer?.name || currentServer?.id || 'server'}`, previewCaddyfile)"
                        >
                          下载
                        </n-button>
                      </div>
                      <div class="relative h-64">
                        <VueMonacoEditor
                          :value="previewCaddyfile"
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
                    </n-tab-pane>
                    <n-tab-pane name="json" tab="JSON">
                      <div class="flex gap-2 mb-2">
                        <n-button size="tiny" @click="copyText(previewCaddyJSON)">复制</n-button>
                        <n-button
                          size="tiny"
                          @click="downloadText(`caddy-${currentServer?.name || currentServer?.id || 'server'}.json`, previewCaddyJSON)"
                        >
                          下载
                        </n-button>
                      </div>
                      <div class="relative h-64">
                        <VueMonacoEditor
                          :value="previewCaddyJSON"
                          language="json"
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
                    </n-tab-pane>
                  </n-tabs>
                </n-collapse-item>
              </n-collapse>
            </div>
          </div>
        </NSpin>
      </div>
    </NCard>

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
        <n-tabs type="line" size="small" class="h-full" style="display: flex; flex-direction: column; min-height: 0;">
          <n-tab-pane name="raw" tab="原始Caddyfile" class="h-full flex flex-col" style="flex: 1; min-height: 0;">
            <div class="flex gap-2 mb-2">
              <n-button size="tiny" @click="copyText(rawSiteContent)">复制</n-button>
              <n-button
                size="tiny"
                @click="downloadText(`Caddyfile-${currentServer?.name || currentServer?.id || 'server'}`, rawSiteContent)"
              >
                下载
              </n-button>
            </div>
            <div class="relative flex-1 min-h-0">
              <VueMonacoEditor
                :value="rawSiteContent"
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
          </n-tab-pane>
          <n-tab-pane name="caddyfile" tab="结构化Caddyfile" class="h-full flex flex-col" style="flex: 1; min-height: 0;">
            <div class="flex gap-2 mb-2">
              <n-button size="tiny" @click="copyText(previewCaddyfile)">复制</n-button>
              <n-button
                size="tiny"
                @click="downloadText(`Caddyfile-structured-${currentServer?.name || currentServer?.id || 'server'}`, previewCaddyfile)"
              >
                下载
              </n-button>
            </div>
            <div class="relative flex-1 min-h-0">
              <VueMonacoEditor
                :value="previewCaddyfile"
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
          </n-tab-pane>
        </n-tabs>
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
</style>
