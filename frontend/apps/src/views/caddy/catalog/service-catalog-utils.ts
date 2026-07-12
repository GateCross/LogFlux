/** 服务目录纯函数：复用工作台站点/指标语义，不建平行 discovery 数据源 */

import type { SiteMetricsItem } from '#/api/caddy/server';

import { parseCaddyfileToBlocks } from '../config/caddy-config-blocks';
import { normalizeModules } from '../config/caddy-config-utils';
import {
  buildQuickConfigState,
  type QuickSiteDraft,
  type QuickSiteMode,
} from '../config/quick-config-utils';
import type { CaddyFormModel, PreservedCaddyBlock } from '../config/types';

export const SITE_METRICS_WINDOW_MINUTES = 15;
export const SITE_METRICS_HOST_CHUNK = 50;

export type CatalogSiteKind = 'simple' | 'complex';
export type CatalogSiteMode = QuickSiteMode | 'complex' | 'unknown';

export interface CatalogSiteCard {
  id: string;
  name: string;
  domains: string[];
  primaryHost: string;
  kind: CatalogSiteKind;
  mode: CatalogSiteMode;
  enabled: boolean;
  upstream?: string;
  lbPolicy?: string;
  healthPath?: string;
  reasons?: string[];
}

export type SiteMetricsMap = Map<string, { count4xx: number; count5xx: number }>;

// 节点探测展示语义统一从 shared 复用（配置管理 / 服务目录）
export type { ServerStatusItem } from '../shared/caddy-server-status';
export {
  formatLatency,
  mergeServerStatusRows,
  statusErrorSummary,
  statusLabel,
  statusTagColor,
} from '../shared/caddy-server-status';

/** 从站点域名中提取可用于 metrics / 日志过滤的 host（去掉端口与协议） */
export function normalizeSiteHost(domain: string): string {
  const raw = String(domain ?? '').trim();
  if (!raw) return '';
  // 监听地址如 :8080 不是 host
  if (raw.startsWith(':')) return '';
  try {
    if (raw.includes('://')) {
      const url = new URL(raw);
      return (url.hostname || '').toLowerCase();
    }
  } catch {
    // 解析失败则继续后续处理
  }
  const withoutPath = raw.split('/')[0] ?? raw;
  const hostPart = withoutPath.includes(']')
    ? withoutPath
    : (withoutPath.split(':')[0] ?? withoutPath);
  return hostPart.trim().toLowerCase();
}

/** 站点卡片用：取第一个有效 host */
export function sitePrimaryHost(domains: string[] | undefined): string {
  for (const domain of domains ?? []) {
    const host = normalizeSiteHost(domain);
    if (host) return host;
  }
  return '';
}

/** 收集目录中全部站点 host，去重 */
export function collectCatalogHosts(sites: CatalogSiteCard[]): string[] {
  const hosts = new Set<string>();
  for (const site of sites) {
    for (const domain of site.domains ?? []) {
      const host = normalizeSiteHost(domain);
      if (host) hosts.add(host);
    }
    if (site.primaryHost) hosts.add(site.primaryHost);
  }
  return [...hosts];
}

export function formatMetricCount(value: number | undefined | null): string {
  if (value === undefined || value === null || Number.isNaN(Number(value))) return '—';
  return String(Math.max(0, Math.round(Number(value))));
}

export function metricCountColor(
  count: number | undefined | null,
  kind: '4xx' | '5xx',
): string {
  if (count === undefined || count === null) return 'default';
  if (Number(count) <= 0) return 'default';
  return kind === '5xx' ? 'error' : 'warning';
}

/** 将 API metrics 列表转为 host → 计数 map（无日志 host 补 0） */
export function buildSiteMetricsMap(
  items: SiteMetricsItem[] | undefined | null,
  expectedHosts: string[] = [],
): SiteMetricsMap {
  const next: SiteMetricsMap = new Map();
  for (const item of items ?? []) {
    const host =
      normalizeSiteHost(item.host) || String(item.host ?? '').trim().toLowerCase();
    if (!host) continue;
    next.set(host, {
      count4xx: Number(item.count4xx) || 0,
      count5xx: Number(item.count5xx) || 0,
    });
  }
  for (const host of expectedHosts) {
    const key = normalizeSiteHost(host) || host.trim().toLowerCase();
    if (!key) continue;
    if (!next.has(key)) {
      next.set(key, { count4xx: 0, count5xx: 0 });
    }
  }
  return next;
}

export function siteMetricsForHost(
  map: SiteMetricsMap,
  domains: string[] | undefined,
): { count4xx: number; count5xx: number } | null {
  const host = sitePrimaryHost(domains);
  if (!host) return null;
  return map.get(host) ?? null;
}

function modeLabel(mode: CatalogSiteMode): string {
  if (mode === 'reverse_proxy') return '反向代理';
  if (mode === 'file_server') return '静态文件';
  if (mode === 'redirect') return '重定向';
  if (mode === 'complex') return '复杂站点';
  return '未知';
}

export function catalogModeLabel(mode: CatalogSiteMode): string {
  return modeLabel(mode);
}

function draftToCard(draft: QuickSiteDraft): CatalogSiteCard {
  return {
    id: draft.id,
    name: draft.name || '未命名站点',
    domains: [...(draft.domains ?? [])],
    primaryHost: sitePrimaryHost(draft.domains),
    kind: 'simple',
    mode: draft.mode,
    enabled: draft.enabled,
    upstream: draft.mode === 'reverse_proxy' ? draft.upstream : undefined,
    lbPolicy: draft.mode === 'reverse_proxy' ? draft.lbPolicy : undefined,
    healthPath: draft.healthCheck?.path,
  };
}

function domainsFromPreservedTitle(title: string): string[] {
  const raw = String(title ?? '').trim();
  if (!raw || raw === '(未命名块)') return [];
  return raw
    .replace(/,/g, ' ')
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

/**
 * 从工作台同源配置模型构建目录站点卡片。
 * simple + complex（含 preserved site 块）合并为一份只读目录，不复制后端数据源。
 */
export function buildCatalogSitesFromModel(
  formModel: CaddyFormModel,
  preservedBlocks: PreservedCaddyBlock[] = [],
): CatalogSiteCard[] {
  const { simpleSites, complexSites } = buildQuickConfigState(formModel);
  const cards: CatalogSiteCard[] = simpleSites.map(draftToCard);

  for (const item of complexSites) {
    cards.push({
      id: item.id,
      name: item.name || '未命名复杂站点',
      domains: [...(item.domains ?? [])],
      primaryHost: sitePrimaryHost(item.domains),
      kind: 'complex',
      mode: 'complex',
      enabled: true,
      reasons: [...(item.reasons ?? [])],
    });
  }

  const knownDomainKeys = new Set(
    cards.flatMap((c) => c.domains.map((d) => d.trim().toLowerCase()).filter(Boolean)),
  );

  for (const block of preservedBlocks) {
    if (block.kind !== 'site') continue;
    const domains = domainsFromPreservedTitle(block.title);
    const primary = domains[0]?.toLowerCase() ?? '';
    // 已在 complex/simple 中出现的域名不再重复
    if (primary && knownDomainKeys.has(primary)) continue;
    if (domains.some((d) => knownDomainKeys.has(d.toLowerCase()))) continue;

    cards.push({
      id: block.id,
      name: block.title || '保留站点',
      domains,
      primaryHost: sitePrimaryHost(domains),
      kind: 'complex',
      mode: 'complex',
      enabled: true,
      reasons: [block.reason || '复杂配置，只读保留原文'],
    });
    for (const d of domains) knownDomainKeys.add(d.toLowerCase());
  }

  return cards;
}

/**
 * 从 GET /caddy/server/:id/config 响应构建目录站点列表。
 * 优先 modules JSON（与工作台一致），否则解析 Caddyfile。
 */
export function buildCatalogSitesFromConfigPayload(data: {
  config?: string;
  modules?: string;
} | null | undefined): CatalogSiteCard[] {
  const configContent = data?.config ?? '';
  let formModel: CaddyFormModel = {
    schemaVersion: 1,
    global: { raw: '' },
    upstreams: [],
    sites: [],
  };
  let preservedBlocks: PreservedCaddyBlock[] = [];

  const modulesRaw = typeof data?.modules === 'string' ? data.modules : undefined;
  let loadedModules = false;
  if (modulesRaw) {
    try {
      const parsed = JSON.parse(modulesRaw);
      const normalized = normalizeModules(parsed);
      const modulePreserved = Array.isArray(parsed?.preservedBlocks)
        ? (parsed.preservedBlocks as PreservedCaddyBlock[])
        : [];
      const hasContent =
        Boolean(normalized.sites?.length) ||
        Boolean(normalized.upstreams?.length) ||
        Boolean(normalized.global?.raw?.trim()) ||
        modulePreserved.some((b) => b.raw?.trim());
      if (hasContent) {
        formModel = normalized;
        preservedBlocks = modulePreserved;
        loadedModules = true;
      }
    } catch {
      loadedModules = false;
    }
  }

  if (configContent.trim()) {
    const parsed = parseCaddyfileToBlocks(configContent);
    if (!loadedModules) {
      formModel = {
        schemaVersion: parsed.schemaVersion,
        global: parsed.global,
        upstreams: parsed.upstreams,
        sites: parsed.sites,
      };
      preservedBlocks = parsed.preservedBlocks ?? [];
    } else {
      // modules 已加载时合并 raw 中的 preserved site，避免遗漏
      const existingRaw = new Set(
        preservedBlocks.map((b) => b.raw?.trim()).filter(Boolean),
      );
      for (const block of parsed.preservedBlocks ?? []) {
        const raw = block.raw?.trim();
        if (!raw || existingRaw.has(raw)) continue;
        if (block.kind === 'site') {
          preservedBlocks.push(block);
          existingRaw.add(raw);
        }
      }
    }
  }

  return buildCatalogSitesFromModel(formModel, preservedBlocks);
}

/** 配置工作台深链 query（不触发 /load） */
export function buildConfigWorkbenchQuery(options: {
  serverId?: number | string | null;
  mode?: 'blocks' | 'waf' | 'raw' | 'preview';
}): Record<string, string> {
  const query: Record<string, string> = {};
  if (options.serverId !== undefined && options.serverId !== null && options.serverId !== '') {
    query.serverId = String(options.serverId);
  }
  if (options.mode) {
    query.mode = options.mode;
  }
  return query;
}

/** 访问日志深链 query */
export function buildAccessLogQuery(domains: string[] | undefined): { host?: string } {
  const host = sitePrimaryHost(domains);
  return host ? { host } : {};
}
