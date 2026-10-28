import type { CaddyFormModel, Handle, HealthCheck, Route, Site } from './types';
import { CLASSIFIED_IMPORTS } from './site-capabilities';

export type QuickSiteMode = 'reverse_proxy' | 'file_server' | 'redirect';

export type QuickTlsMode = 'auto' | 'off' | 'internal';

export type QuickLbPolicy = 'round_robin' | 'least_conn' | 'ip_hash';

/** Simple 路径允许的负载策略（不单独判 complex） */
const SIMPLE_LB_POLICIES = new Set<string>(['round_robin', 'least_conn', 'ip_hash']);

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
  /** reverse_proxy 负载策略；round-trip 不得静默丢弃 */
  lbPolicy?: QuickLbPolicy;
  /** reverse_proxy 健康检查；round-trip 不得静默丢弃 */
  healthCheck?: HealthCheck;
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
    redirectCode: partial?.redirectCode ?? 302,
    lbPolicy: partial?.lbPolicy,
    healthCheck: partial?.healthCheck
      ? {
          path: partial.healthCheck.path,
          interval: partial.healthCheck.interval,
          timeout: partial.healthCheck.timeout
        }
      : undefined
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

function analyzeHandleMode(handle: Handle, _upstreamNames: Set<string>, reasons: Set<string>): QuickSiteMode | null {
  if (handle.type === 'reverse_proxy') {
    // healthCheck、已定义上游池引用、常见 lbPolicy 不再单独判 complex（Req 2.1）
    // transport http 与 tls_insecure_skip_verify 是详情页基本信息 Tab 的结构化开关
    // （upstreamHttps / upstreamSkipVerify），不算复杂；仅自定义非 http 传输才降级。
    if (handle.transportProtocol && handle.transportProtocol !== 'http') {
      reasons.add(`反向代理包含自定义传输协议 ${handle.transportProtocol}`);
    }
    if (handle.lbPolicy && !SIMPLE_LB_POLICIES.has(handle.lbPolicy)) {
      reasons.add('反向代理包含负载策略');
    }
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
  const draft = createQuickSiteDraft({
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

  if (mode === 'reverse_proxy') {
    if (handle.lbPolicy && SIMPLE_LB_POLICIES.has(handle.lbPolicy)) {
      draft.lbPolicy = handle.lbPolicy as QuickLbPolicy;
    }
    if (handle.healthCheck) {
      draft.healthCheck = {
        path: handle.healthCheck.path,
        interval: handle.healthCheck.interval,
        timeout: handle.healthCheck.timeout
      };
    }
  }

  return draft;
}

export function analyzeSiteForQuickConfig(site: Site, upstreamNames: Set<string>) {
  const reasons = createReasons();

  // 已归类 snippet 的 import（waf_protect/cloudflare/geo_auth 等能力挂载行）是
  // 系统自己生成的写法，且基本信息保存走外科编辑不会破坏它们，不算复杂；
  // 只有无法归类的 import 才降级为复杂站点。
  //
  // 这类 import 不进 reasons：它是"简单模式保存会写进 imports: []、从而丢行"的
  // 内部约束，不是用户需要读的说明。单独用标志位降级，界面不提示。
  const unclassifiedImports = (site.imports ?? []).filter(
    (name) => !CLASSIFIED_IMPORTS.has(name),
  );
  const hasUnclassifiedImport = !isEmptyArray(unclassifiedImports);

  // dns（cloudflare DNS 签发）由 import cloudflare 表达，同样不算高级模式。
  if (!isEmptyArray(site.geoip2Vars)) reasons.add('站点包含 GeoIP2 变量');
  if (!isEmptyArray(site.encode)) reasons.add('站点包含 encode 配置');
  if (!isEmptyArray(site.headers)) reasons.add('站点包含 header 配置');

  const tlsMode = site.tls?.mode ?? 'auto';
  if (!['auto', 'off', 'internal', 'dns'].includes(tlsMode)) {
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
    // 解析器为无 matcher 的 handle 块生成的名字是"未命名路由"，历史配置可能是
    // "默认路由"，两者都是生成名而非用户命名，不算自定义。
    if (
      route.name?.trim() &&
      route.name.trim() !== '默认路由' &&
      route.name.trim() !== '未命名路由'
    ) {
      reasons.add('主路由包含自定义命名');
    }
    if (hasRouteMatchers(route)) reasons.add('主路由包含 matcher');
    if (!isEmptyArray(route.logAppend)) reasons.add('主路由包含日志追加配置');
    if (route.handles.length !== 1) reasons.add('主路由包含多个处理器');

    const handle = route.handles[0];
    if (!handle) {
      reasons.add('主路由未配置处理器');
    } else {
      if (!handle.enabled) reasons.add('主处理器已禁用');
      const mode = analyzeHandleMode(handle, upstreamNames, reasons);
      // 有不可归类 import 时即使其余检查全过也不能进简单模式：
      // buildSiteFromQuickDraft 会写 imports: []，保存即丢掉那些 import 行。
      if (reasons.size === 0 && mode && !hasUnclassifiedImport) {
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
            // 保留草稿中的 lbPolicy；未设置时默认 round_robin，禁止静默覆盖已有策略
            lbPolicy: draft.lbPolicy ?? 'round_robin',
            // 保留健康检查语义，不得丢弃为空
            healthCheck: draft.healthCheck
              ? {
                  path: draft.healthCheck.path,
                  interval: draft.healthCheck.interval,
                  timeout: draft.healthCheck.timeout
                }
              : undefined,
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
