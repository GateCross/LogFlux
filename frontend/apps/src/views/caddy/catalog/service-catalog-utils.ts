/** 服务目录纯函数：复用工作台站点/指标语义，不建平行 discovery 数据源 */

import type { CaddyCertificateApi } from '#/api/caddy/certificate';
import type { SiteMetricsItem } from '#/api/caddy/server';

import { extractSiteBlocks, parseCaddyfileToBlocks } from '../config/caddy-config-blocks';
import { extractSiteCapabilities, type SiteCapabilitiesSnapshot } from '../config/site-capabilities';
import { normalizeModules } from '../config/caddy-config-utils';
import {
  buildQuickConfigState,
  type QuickSiteDraft,
  type QuickSiteMode,
} from '../config/quick-config-utils';
import type { CaddyFormModel, PreservedCaddyBlock } from '../config/types';

/** 站点指标分块大小：后端单次最多 50 个 host */
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
  /** 站点块原文（能力识别与抽屉编辑的事实来源） */
  raw?: string;
  /** 从块原文识别出的能力状态 */
  capabilities?: SiteCapabilitiesSnapshot;
}

/** 站点窗口指标计数；total/blocked 为 window=24h 契约字段，后端未返回时为 undefined */
export interface SiteMetricsCounters {
  count4xx: number;
  count5xx: number;
  /** 窗口内总请求数（今日请求） */
  total?: number;
  /** 窗口内 WAF 拦截数（今日拦截） */
  blocked?: number;
}

export type SiteMetricsMap = Map<string, SiteMetricsCounters>;

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

/** 将 API metrics 列表转为 host → 计数 map（无日志 host 补 0；total/blocked 原样透传） */
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
      total: item.total == null ? undefined : Math.max(0, Number(item.total)),
      blocked: item.blocked == null ? undefined : Math.max(0, Number(item.blocked)),
    });
  }
  for (const host of expectedHosts) {
    const key = normalizeSiteHost(host) || host.trim().toLowerCase();
    if (!key) continue;
    if (!next.has(key)) {
      // 窗口内无任何日志行：后端 GROUP BY 不会返回该 host，按 0 补齐
      next.set(key, { count4xx: 0, count5xx: 0, total: 0, blocked: 0 });
    }
  }
  return next;
}

export function siteMetricsForHost(
  map: SiteMetricsMap,
  domains: string[] | undefined,
): SiteMetricsCounters | null {
  const host = sitePrimaryHost(domains);
  if (!host) return null;
  return map.get(host) ?? null;
}

/** 站点全部域名（去重、去端口协议）对应的 host 列表 */
function siteHosts(domains: string[] | undefined): string[] {
  const hosts = new Set<string>();
  for (const domain of domains ?? []) {
    const host = normalizeSiteHost(domain);
    if (host) hosts.add(host);
  }
  return [...hosts];
}

/**
 * 按站点聚合全域名指标：一个站点挂多个域名时，各域名的流量都应计入本站点。
 * total/blocked 任一域名缺值即视为整站缺值（后端未上线该字段时按 — 降级）；
 * 4xx/5xx 恒为数值，缺失按 0 累加。
 */
export function siteMetricsForDomains(
  map: SiteMetricsMap,
  domains: string[] | undefined,
): SiteMetricsCounters | null {
  const hosts = siteHosts(domains);
  if (hosts.length === 0) return null;

  let count4xx = 0;
  let count5xx = 0;
  let total: number | undefined = 0;
  let blocked: number | undefined = 0;

  for (const host of hosts) {
    const metrics = map.get(host);
    count4xx += metrics?.count4xx ?? 0;
    count5xx += metrics?.count5xx ?? 0;
    // 任一无数据即整体退回 undefined，避免半有半无时给出失真的小计
    total = total === undefined || metrics?.total === undefined
      ? undefined
      : total + metrics.total;
    blocked = blocked === undefined || metrics?.blocked === undefined
      ? undefined
      : blocked + metrics.blocked;
  }

  return { count4xx, count5xx, total, blocked };
}

/**
 * 站点命中的证书剩余天数：按域名匹配证书 SAN（支持 *.example.com 泛域名后缀），
 * 多张命中取最小剩余天数（最紧迫者决定展示）；未命中或无数据返回 null。
 */
export function siteCertDaysLeft(
  certs: CaddyCertificateApi.CertificateItem[] | undefined | null,
  domains: string[] | undefined,
): number | null {
  if (!certs?.length || !domains?.length) return null;
  const hosts = new Set<string>();
  for (const domain of domains) {
    const host = normalizeSiteHost(domain);
    if (host) hosts.add(host);
  }
  if (hosts.size === 0) return null;
  let best: number | null = null;
  for (const cert of certs) {
    const sans = String(cert.domains ?? '')
      .split(',')
      .map((item) => item.trim().toLowerCase())
      .filter(Boolean);
    const covered = [...hosts].some((host) =>
      sans.some(
        (san) =>
          san === host ||
          (san.startsWith('*.') &&
            (host === san.slice(2) || host.endsWith(`.${san.slice(2)}`))),
      ),
    );
    if (!covered) continue;
    const days = Number(cert.daysLeft);
    if (!Number.isFinite(days)) continue;
    if (best === null || days < best) best = days;
  }
  return best;
}

/** 卡片/详情头防护模式徽章：站点挂载状态 × 全局 WAF 引擎状态 */
export interface SiteProtectBadge {
  label: string;
  color: string;
  /** 补充说明（Tooltip） */
  title?: string;
}

/**
 * 从站点能力快照推导防护徽章。
 *
 * 引擎模式是站点级的：由站点挂的 snippet（waf_protect / waf_protect_detect）决定，
 * 不看全局策略。维护模式优先级最高。
 */
export function siteProtectBadge(
  site: { capabilities?: Partial<Pick<SiteCapabilitiesSnapshot, 'maintenance' | 'waf'>> | null },
): SiteProtectBadge {
  const caps = site.capabilities;
  if (!caps) {
    return { label: '未知', color: 'default', title: '未取到站点块原文，能力未知，进入详情查看' };
  }
  // 维护模式优先级最高：整站返回 503，其余防护状态都不再是本站在做的事
  if (caps.maintenance?.status === 'on') {
    return {
      label: '维护模式',
      color: 'red',
      title: '展示维护页面，任何人都将无法访问您的应用',
    };
  }
  const waf = caps.waf;
  if (!waf || waf.mode === 'off') {
    return { label: '未防护', color: 'default', title: '该站点未开启 WAF 防护' };
  }
  if (waf.mode === 'unknown') {
    return {
      label: 'WAF?',
      color: 'default',
      title: '检测到内联 coraza_waf，无法判断引擎模式，请在详情查看',
    };
  }
  if (waf.mode === 'detect') {
    return { label: '观察模式', color: 'orange', title: '发现攻击后仅记录攻击事件，不拦截' };
  }
  return { label: '防护模式', color: 'green', title: '发现攻击后将自动拦截，并记录攻击事件' };
}

function modeLabel(mode: CatalogSiteMode): string {
  if (mode === 'reverse_proxy') return '反向代理';
  if (mode === 'file_server') return '静态文件';
  if (mode === 'redirect') return '重定向';
  // 不再使用"复杂站点"说法：无法结构化归纳的站点统称自定义配置，
  // 基本信息仍表单可编辑，无法识别的项在对应字段上就地提示。
  if (mode === 'complex') return '自定义';
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
      // 无 reason 表示解析器自保（如未识别指令），不对用户提示
      reasons: block.reason ? [block.reason] : [],
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

  if (configContent.trim()) {
    // 有配置原文时一律新鲜解析：modules 是落库时的分类快照，分类规则升级后
    // （如 snippet 风格站点改判 simple）旧快照会把站点继续标成复杂，必须以
    // 当前规则重解析为准；preserved 块也能从原文原地提取，不依赖快照。
    const parsed = parseCaddyfileToBlocks(configContent);
    formModel = {
      schemaVersion: parsed.schemaVersion,
      global: parsed.global,
      upstreams: parsed.upstreams,
      sites: parsed.sites,
    };
    preservedBlocks = parsed.preservedBlocks ?? [];
  } else if (modulesRaw) {
    try {
      const parsed = JSON.parse(modulesRaw);
      const normalized = normalizeModules(parsed);
      const modulePreserved = Array.isArray(parsed?.preservedBlocks)
        ? (parsed.preservedBlocks as PreservedCaddyBlock[])
        : [];
      formModel = normalized;
      preservedBlocks = modulePreserved;
    } catch {
      // modules 损坏时保持空模型，卡片展示为空
    }
  }

  const cards = buildCatalogSitesFromModel(formModel, preservedBlocks);
  attachSiteBlockRaw(cards, configContent);
  return cards;
}

/** 按首域名把站点块原文与能力状态挂到卡片上，供徽章与抽屉使用 */
function attachSiteBlockRaw(cards: CatalogSiteCard[], configContent: string): void {
  if (!configContent.trim()) return;
  const rawByDomain = extractSiteBlocks(configContent);
  for (const card of cards) {
    const key = card.domains.find((domain) => rawByDomain.has(domain));
    const raw = key ? rawByDomain.get(key) : undefined;
    if (!raw) continue;
    card.raw = raw;
    card.capabilities = extractSiteCapabilities(raw);
  }
}

/** 配置工作台深链 query（不触发 /load） */
export function buildConfigWorkbenchQuery(options: {
  mode?: 'blocks' | 'waf' | 'raw' | 'preview';
}): Record<string, string> {
  const query: Record<string, string> = {};
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
