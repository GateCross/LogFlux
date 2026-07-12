/** 站点近窗指标；失败降级为 "—"，不阻断配置编辑 */

import { ref, type Ref } from 'vue';
import { useRouter } from 'vue-router';
import { useDebounceFn } from '@vueuse/core';
import { message } from 'antdv-next';

import { getSiteMetricsApi } from '#/api/caddy/server';
import type { SiteMetricsItem } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';
import { useQueryClient } from '@tanstack/vue-query';

/** 站点卡片近窗指标（host → 计数） */
export type SiteMetricsMap = Map<string, { count4xx: number; count5xx: number }>;

export const SITE_METRICS_WINDOW_MINUTES = 15;
export const SITE_METRICS_HOST_CHUNK = 50;

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
  // 去掉 path / query
  const withoutPath = raw.split('/')[0] ?? raw;
  // IPv4/域名去掉端口；罕见 IPv6 host:port 无解析时保留方括号
  const hostPart = withoutPath.includes(']')
    ? withoutPath
    : (withoutPath.split(':')[0] ?? withoutPath);
  return hostPart.trim().toLowerCase();
}

/** 站点卡片用：取第一个有效 host 对应的指标 */
export function sitePrimaryHost(domains: string[] | undefined): string {
  for (const domain of domains ?? []) {
    const host = normalizeSiteHost(domain);
    if (host) return host;
  }
  return '';
}

/** 收集站点域名（简单 + 复杂），去重 */
export function collectHostsFromSites(
  simpleSites: Array<{ domains?: string[] }>,
  complexSites: Array<{ domains?: string[] }>,
): string[] {
  const hosts = new Set<string>();
  for (const site of simpleSites) {
    for (const domain of site.domains ?? []) {
      const host = normalizeSiteHost(domain);
      if (host) hosts.add(host);
    }
  }
  for (const site of complexSites) {
    for (const domain of site.domains ?? []) {
      const host = normalizeSiteHost(domain);
      if (host) hosts.add(host);
    }
  }
  return [...hosts];
}

export function formatMetricCount(value: number | undefined | null): string {
  if (value === undefined || value === null || Number.isNaN(Number(value))) return '—';
  return String(Math.max(0, Math.round(Number(value))));
}

export function metricCountColor(count: number | undefined | null, kind: '4xx' | '5xx') {
  if (count === undefined || count === null) return 'default';
  if (Number(count) <= 0) return 'default';
  return kind === '5xx' ? 'error' : 'warning';
}

export interface UseSiteMetricsOptions {
  /** 简单站点草稿列表（用于收集 host） */
  quickSiteDrafts: Ref<Array<{ domains?: string[] }>>;
  /** 复杂站点摘要列表 */
  complexSiteSummaries: Ref<Array<{ domains?: string[] }>>;
}

export function useSiteMetrics(options: UseSiteMetricsOptions) {
  const { quickSiteDrafts, complexSiteSummaries } = options;
  const router = useRouter();
  const queryClient = useQueryClient();

  const siteMetricsMap = ref<SiteMetricsMap>(new Map());
  const loadingSiteMetrics = ref(false);
  const siteMetricsError = ref('');
  const siteMetricsLoaded = ref(false);

  function collectSiteHosts(): string[] {
    return collectHostsFromSites(quickSiteDrafts.value, complexSiteSummaries.value);
  }

  function siteMetricsForDomains(domains: string[] | undefined) {
    const host = sitePrimaryHost(domains);
    if (!host) return null;
    return siteMetricsMap.value.get(host) ?? null;
  }

  function resetSiteMetrics() {
    siteMetricsMap.value = new Map();
    siteMetricsError.value = '';
    siteMetricsLoaded.value = false;
  }

  /**
   * 批量拉取站点近窗 4xx/5xx。
   * 失败时仅记录错误并清空/保留上次数据展示 "—"，绝不阻断配置编辑。
   */
  async function fetchSiteMetrics() {
    if (loadingSiteMetrics.value) return;
    const hosts = collectSiteHosts();
    if (hosts.length === 0) {
      siteMetricsMap.value = new Map();
      siteMetricsError.value = '';
      siteMetricsLoaded.value = true;
      return;
    }

    loadingSiteMetrics.value = true;
    siteMetricsError.value = '';
    try {
      const next = new Map<string, { count4xx: number; count5xx: number }>();
      // 后端单次最多 50 hosts，分块请求
      for (let i = 0; i < hosts.length; i += SITE_METRICS_HOST_CHUNK) {
        const chunk = hosts.slice(i, i + SITE_METRICS_HOST_CHUNK);
        const list = await queryClient.fetchQuery({
          queryKey: qk.caddy.metrics(0, chunk.join(',')),
          queryFn: () =>
            getSiteMetricsApi(
              {
                hosts: chunk,
                windowMinutes: SITE_METRICS_WINDOW_MINUTES,
              },
              withListDetailErrorMode(),
            ),
        });
        for (const item of list as SiteMetricsItem[]) {
          const host = normalizeSiteHost(item.host) || String(item.host ?? '').trim().toLowerCase();
          if (!host) continue;
          next.set(host, {
            count4xx: Number(item.count4xx) || 0,
            count5xx: Number(item.count5xx) || 0,
          });
        }
      }
      // 无日志 host 后端可能省略，补 0
      for (const host of hosts) {
        if (!next.has(host)) {
          next.set(host, { count4xx: 0, count5xx: 0 });
        }
      }
      siteMetricsMap.value = next;
      siteMetricsLoaded.value = true;
    } catch (error) {
      // 降级：不阻塞编辑；展示 "—" 与可选中文提示
      siteMetricsError.value = apiErrorMessage(error, '获取站点近窗指标失败');
      siteMetricsMap.value = new Map();
      siteMetricsLoaded.value = true;
    } finally {
      loadingSiteMetrics.value = false;
    }
  }

  /** 配置/站点变更后 debounce 拉取，避免每个键入 thrash */
  const debouncedFetchSiteMetrics = useDebounceFn(fetchSiteMetrics, 600);

  function handleRefreshSiteMetrics() {
    debouncedFetchSiteMetrics();
  }

  /** 深链到访问日志并带 host 过滤 */
  function openAccessLogs(domains: string[] | undefined, event?: Event) {
    event?.stopPropagation();
    event?.preventDefault();
    const host = sitePrimaryHost(domains);
    if (!host) {
      message.warning('该站点未配置可用域名，无法过滤访问日志');
      return;
    }
    router.push({
      name: 'CaddyLog',
      query: { host },
    });
  }

  return {
    siteMetricsMap,
    loadingSiteMetrics,
    siteMetricsError,
    siteMetricsLoaded,
    SITE_METRICS_WINDOW_MINUTES,
    collectSiteHosts,
    sitePrimaryHost,
    siteMetricsForDomains,
    formatMetricCount,
    metricCountColor,
    resetSiteMetrics,
    fetchSiteMetrics,
    debouncedFetchSiteMetrics,
    handleRefreshSiteMetrics,
    openAccessLogs,
  };
}
