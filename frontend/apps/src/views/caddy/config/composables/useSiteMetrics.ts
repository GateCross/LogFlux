/**
 * 配置工作台视角的站点指标：近 15 分钟 4xx/5xx（运维视角，编辑配置时看抖动）。
 *
 * 请求、分块、并发去重全部交给 shared/composables/useSiteMetricsQuery，
 * 这里只保留工作台专属的部分：host 收集（简单站点 + 复杂站点）、
 * 深链跳转访问日志。
 */
import { computed, type Ref } from 'vue';
import { useRouter } from 'vue-router';
import { message } from 'antdv-next';

import {
  SITE_METRICS_RECENT_MINUTES,
  useSiteMetricsQuery,
} from '../../shared/composables/useSiteMetricsQuery';
import {
  normalizeSiteHost,
  sitePrimaryHost,
} from '../../catalog/service-catalog-utils';

/** 收集站点域名（简单 + 复杂），去重 */
export function collectHostsFromSites(
  simpleSites: Array<{ domains?: string[] }>,
  complexSites: Array<{ domains?: string[] }>,
): string[] {
  const hosts = new Set<string>();
  for (const site of [...simpleSites, ...complexSites]) {
    for (const domain of site.domains ?? []) {
      const host = normalizeSiteHost(domain);
      if (host) hosts.add(host);
    }
  }
  return [...hosts];
}

export interface UseSiteMetricsOptions {
  /** 简单站点草稿列表（用于收集 host） */
  quickSiteDrafts: Ref<Array<{ domains?: string[] }>>;
  /** 复杂站点摘要列表 */
  complexSiteSummaries: Ref<Array<{ domains?: string[] }>>;
}

/** 工作台指标：近 15 分钟窗口，失败降级为 "—"，不阻断配置编辑 */
export function useSiteMetrics(options: UseSiteMetricsOptions) {
  const { quickSiteDrafts, complexSiteSummaries } = options;
  const router = useRouter();

  const metricsHosts = computed(() =>
    collectHostsFromSites(quickSiteDrafts.value, complexSiteSummaries.value),
  );

  const {
    metricsMap: siteMetricsMap,
    loading: loadingSiteMetrics,
    loaded: siteMetricsLoaded,
    errorMessage: siteMetricsError,
    refetch: refetchSiteMetrics,
  } = useSiteMetricsQuery({
    hosts: metricsHosts,
    windowMinutes: SITE_METRICS_RECENT_MINUTES,
  });

  function siteMetricsForDomains(domains: string[] | undefined) {
    const host = sitePrimaryHost(domains);
    if (!host) return null;
    return siteMetricsMap.value.get(host) ?? null;
  }

  /** 配置重载时由调用方触发；hosts 变化后共享 query 会自行重取 */
  function resetSiteMetrics() {
    void refetchSiteMetrics();
  }

  async function fetchSiteMetrics() {
    if (metricsHosts.value.length === 0) return;
    await refetchSiteMetrics();
  }

  function handleRefreshSiteMetrics() {
    void fetchSiteMetrics();
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
      name: 'LogsAccess',
      query: { host },
    });
  }

  return {
    siteMetricsMap,
    loadingSiteMetrics,
    siteMetricsError,
    siteMetricsLoaded,
    sitePrimaryHost,
    siteMetricsForDomains,
    resetSiteMetrics,
    fetchSiteMetrics,
    handleRefreshSiteMetrics,
    openAccessLogs,
  };
}
