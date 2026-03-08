import type { CaddyFormModel, Handle, Route, Site } from './types';

export type QuickSiteMode = 'reverse_proxy' | 'file_server' | 'redirect';

export type QuickTlsMode = 'auto' | 'off' | 'internal';

export interface QuickSiteDraft {
  id: string;
  name: string;
  enabled: boolean;
  domains: string[];
  tlsMode: QuickTlsMode;
  mode: QuickSiteMode;
  upstream: string;
  root: string;
  browse: boolean;
  redirectTo: string;
  redirectCode: number;
}

export interface ComplexSiteSummary {
  id: string;
  name: string;
  domains: string[];
  reasons: string[];
}

export interface QuickConfigState {
  simpleSites: QuickSiteDraft[];
  complexSites: ComplexSiteSummary[];
}

function cloneHandle(handle: Handle): Handle {
  return {
    ...handle,
    healthCheck: handle.healthCheck ? { ...handle.healthCheck } : undefined,
    rules: handle.rules ? handle.rules.map(rule => ({ ...rule })) : undefined
  };
}

function cloneRoute(route: Route): Route {
  return {
    ...route,
    match: {
      host: [...route.match.host],
      path: [...route.match.path],
      method: [...route.match.method],
      header: route.match.header.map(item => ({ ...item })),
      query: route.match.query.map(item => ({ ...item })),
      expression: route.match.expression ?? ''
    },
    handles: route.handles.map(cloneHandle),
    logAppend: route.logAppend?.map(item => ({ ...item })) ?? []
  };
}

function cloneSite(site: Site): Site {
  return {
    ...site,
    domains: [...site.domains],
    tls: site.tls ? { ...site.tls } : undefined,
    imports: [...(site.imports ?? [])],
    geoip2Vars: [...(site.geoip2Vars ?? [])],
    encode: [...(site.encode ?? [])],
    headers: site.headers?.map(item => ({ ...item })),
    routes: site.routes.map(cloneRoute)
  };
}

function cloneFormModel(formModel: CaddyFormModel): CaddyFormModel {
  return {
    schemaVersion: formModel.schemaVersion,
    global: {
      ...formModel.global,
      tls: formModel.global.tls ? { ...formModel.global.tls } : undefined,
      headers: formModel.global.headers?.map(item => ({ ...item })),
      rateLimit: formModel.global.rateLimit ? { ...formModel.global.rateLimit } : undefined,
      raw: formModel.global.raw ?? ''
    },
    upstreams: formModel.upstreams.map(item => ({
      ...item,
      targets: [...item.targets],
      healthCheck: item.healthCheck ? { ...item.healthCheck } : undefined
    })),
    sites: formModel.sites.map(cloneSite)
  };
}

function createReasons() {
  return new Set<string>();
}

function normalizeSiteName(site: Site) {
  return site.name?.trim() || site.domains[0] || '未命名站点';
}

export function createQuickSiteDraft(partial?: Partial<QuickSiteDraft>): QuickSiteDraft {
  return {
    id: partial?.id ?? createFallbackId(),
    name: partial?.name ?? '新站点',
    enabled: partial?.enabled ?? true,
    domains: partial?.domains ? [...partial.domains] : [],
    tlsMode: partial?.tlsMode ?? 'auto',
    mode: partial?.mode ?? 'reverse_proxy',
    upstream: partial?.upstream ?? '',
    root: partial?.root ?? '',
    browse: partial?.browse ?? false,
    redirectTo: partial?.redirectTo ?? '',
    redirectCode: partial?.redirectCode ?? 302
  };
}

function createFallbackId() {
  return (crypto as any).randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function isEmptyArray(value?: unknown[]) {
  return !Array.isArray(value) || value.length === 0;
}

function hasRouteMatchers(route: Route) {
  return (
    route.match.host.length > 0 ||
    route.match.path.length > 0 ||
    route.match.method.length > 0 ||
    route.match.header.length > 0 ||
    route.match.query.length > 0 ||
    Boolean(route.match.expression?.trim())
  );
}

function analyzeHandleMode(handle: Handle, upstreamNames: Set<string>, reasons: Set<string>): QuickSiteMode | null {
  if (handle.type === 'reverse_proxy') {
    if (handle.healthCheck) reasons.add('反向代理包含健康检查');
    if (handle.transportProtocol) reasons.add('反向代理包含自定义传输协议');
    if (handle.tlsInsecureSkipVerify) reasons.add('反向代理包含 TLS 跳过校验');
    if (handle.lbPolicy && handle.lbPolicy !== 'round_robin') reasons.add('反向代理包含负载策略');
    if (handle.upstream && upstreamNames.has(handle.upstream)) reasons.add('反向代理引用了上游池');
    return 'reverse_proxy';
  }

  if (handle.type === 'file_server') {
    return 'file_server';
  }

  if (handle.type === 'redirect') {
    return 'redirect';
  }

  reasons.add(`包含 ${handle.type} 高级处理器`);
  return null;
}

function buildQuickDraftFromSite(site: Site, handle: Handle, mode: QuickSiteMode): QuickSiteDraft {
  return createQuickSiteDraft({
    id: site.id,
    name: normalizeSiteName(site),
    enabled: site.enabled,
    domains: [...site.domains],
    tlsMode: (site.tls?.mode === 'off' || site.tls?.mode === 'internal' ? site.tls.mode : 'auto') as QuickTlsMode,
    mode,
    upstream: handle.upstream ?? '',
    root: handle.root ?? '',
    browse: handle.browse ?? false,
    redirectTo: handle.to ?? '',
    redirectCode: handle.code ?? 302
  });
}

export function analyzeSiteForQuickConfig(site: Site, upstreamNames: Set<string>) {
  const reasons = createReasons();

  if (!isEmptyArray(site.imports)) reasons.add('站点包含 import');
  if (!isEmptyArray(site.geoip2Vars)) reasons.add('站点包含 GeoIP2 变量');
  if (!isEmptyArray(site.encode)) reasons.add('站点包含 encode 配置');
  if (!isEmptyArray(site.headers)) reasons.add('站点包含 header 配置');

  const tlsMode = site.tls?.mode ?? 'auto';
  if (!['auto', 'off', 'internal'].includes(tlsMode)) {
    reasons.add('TLS 使用了高级模式');
  }

  if (site.routes.length !== 1) {
    reasons.add('站点包含多条路由');
  }

  const route = site.routes[0];
  if (!route) {
    reasons.add('站点未配置主路由');
  } else {
    if (!route.enabled) reasons.add('主路由已禁用');
    if (route.name && route.name.trim() && route.name.trim() !== '默认路由') reasons.add('主路由包含自定义命名');
    if (hasRouteMatchers(route)) reasons.add('主路由包含 matcher');
    if (!isEmptyArray(route.logAppend)) reasons.add('主路由包含日志追加配置');
    if (route.handles.length !== 1) reasons.add('主路由包含多个处理器');

    const handle = route.handles[0];
    if (!handle) {
      reasons.add('主路由未配置处理器');
    } else {
      if (!handle.enabled) reasons.add('主处理器已禁用');
      const mode = analyzeHandleMode(handle, upstreamNames, reasons);
      if (reasons.size === 0 && mode) {
        return {
          kind: 'simple' as const,
          draft: buildQuickDraftFromSite(site, handle, mode)
        };
      }
    }
  }

  return {
    kind: 'complex' as const,
    summary: {
      id: site.id,
      name: normalizeSiteName(site),
      domains: [...site.domains],
      reasons: [...reasons]
    }
  };
}

export function buildQuickConfigState(formModel: CaddyFormModel): QuickConfigState {
  const upstreamNames = new Set(formModel.upstreams.map(item => item.name).filter(Boolean));
  const simpleSites: QuickSiteDraft[] = [];
  const complexSites: ComplexSiteSummary[] = [];

  for (const site of formModel.sites) {
    const analyzed = analyzeSiteForQuickConfig(site, upstreamNames);
    if (analyzed.kind === 'simple') {
      simpleSites.push(analyzed.draft);
      continue;
    }
    complexSites.push(analyzed.summary);
  }

  return {
    simpleSites,
    complexSites
  };
}

export function buildSiteFromQuickDraft(draft: QuickSiteDraft): Site {
  const handle: Handle =
    draft.mode === 'file_server'
      ? {
          id: `${draft.id}-handle`,
          type: 'file_server',
          enabled: true,
          root: draft.root.trim(),
          browse: draft.browse
        }
      : draft.mode === 'redirect'
        ? {
            id: `${draft.id}-handle`,
            type: 'redirect',
            enabled: true,
            to: draft.redirectTo.trim(),
            code: draft.redirectCode || 302
          }
        : {
            id: `${draft.id}-handle`,
            type: 'reverse_proxy',
            enabled: true,
            upstream: draft.upstream.trim(),
            lbPolicy: 'round_robin',
            tlsInsecureSkipVerify: false,
            transportProtocol: ''
          };

  return {
    id: draft.id,
    name: draft.name.trim() || '未命名站点',
    enabled: draft.enabled,
    domains: draft.domains.map(item => item.trim()).filter(Boolean),
    tls: { mode: draft.tlsMode },
    imports: [],
    geoip2Vars: [],
    encode: [],
    routes: [
      {
        id: `${draft.id}-route`,
        name: '默认路由',
        enabled: true,
        match: {
          host: [],
          path: [],
          method: [],
          header: [],
          query: [],
          expression: ''
        },
        handles: [handle],
        logAppend: []
      }
    ]
  };
}

export function mergeQuickConfigDrafts(formModel: CaddyFormModel, drafts: QuickSiteDraft[]): CaddyFormModel {
  const next = cloneFormModel(formModel);
  const draftMap = new Map(drafts.map(item => [item.id, buildSiteFromQuickDraft(item)]));
  const quickState = buildQuickConfigState(formModel);
  const simpleIds = new Set(quickState.simpleSites.map(item => item.id));
  const mergedSites: Site[] = [];

  for (const site of next.sites) {
    if (simpleIds.has(site.id)) {
      const replacement = draftMap.get(site.id);
      if (replacement) {
        mergedSites.push(replacement);
      }
      continue;
    }
    mergedSites.push(site);
  }

  for (const draft of drafts) {
    if (!simpleIds.has(draft.id)) {
      mergedSites.push(buildSiteFromQuickDraft(draft));
    }
  }

  next.sites = mergedSites;
  return next;
}

