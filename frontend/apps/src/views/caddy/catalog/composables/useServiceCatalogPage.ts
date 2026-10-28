import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { message } from 'antdv-next';

import { getCaddyConfigApi } from '#/api/caddy/server';
import { getCertificateListApi } from '#/api/caddy/certificate';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import {
  buildAccessLogQuery,
  buildCatalogSitesFromConfigPayload,
  buildConfigWorkbenchQuery,
  collectCatalogHosts,
  formatMetricCount,
  siteMetricsForDomains,
  sitePrimaryHost,
  type CatalogSiteCard,
} from '../service-catalog-utils';
import {
  SITE_METRICS_TODAY_MINUTES,
  useSiteMetricsQuery,
} from '../../shared/composables/useSiteMetricsQuery';

export function useServiceCatalogPage() {
  const router = useRouter();

  const keyword = ref('');

  const {
    data: configData,
    loading: loadingConfig,
    errorMessage: configErrorMessage,
    refetch: refetchConfig,
  } = useListDetailQuery({
    queryKey: qk.caddy.catalog(),
    queryFn: () => getCaddyConfigApi(withListDetailErrorMode()),
    errorFallback: '获取站点配置失败',
  });

  const siteCards = computed(() => {
    if (!configData.value) return [] as CatalogSiteCard[];
    return buildCatalogSitesFromConfigPayload(configData.value);
  });

  // 指标统一走共享 query（今日 24h 视角）；hosts 由站点卡片派生
  const metricsHosts = computed(() => collectCatalogHosts(siteCards.value));
  const {
    metricsMap: siteMetricsMap,
    loading: loadingSiteMetrics,
    loaded: siteMetricsLoaded,
    errorMessage: siteMetricsError,
    refetch: refetchSiteMetrics,
  } = useSiteMetricsQuery({
    hosts: metricsHosts,
    windowMinutes: SITE_METRICS_TODAY_MINUTES,
  });

  /** 当前配置原文与哈希，供一步建站弹窗做追加保存与乐观并发 */
  const configRaw = computed(() => configData.value?.config ?? '');
  const configHash = computed(() => configData.value?.hash ?? '');

  /** 证书库存（与证书中心同 queryKey 共享缓存）：卡片「证书天数」chip 数据源 */
  const {
    data: certData,
    loading: loadingCerts,
  } = useListDetailQuery({
    queryKey: qk.caddy.certificates(),
    queryFn: () => getCertificateListApi(withListDetailErrorMode()),
    errorFallback: '获取证书列表失败',
  });
  const certList = computed(() => certData.value ?? []);

  const filteredSites = computed(() => {
    const q = keyword.value.trim().toLowerCase();
    return siteCards.value.filter((site) => {
      if (!q) return true;
      const hay = [site.name, site.primaryHost, ...(site.domains ?? []), site.upstream ?? '']
        .join(' ')
        .toLowerCase();
      return hay.includes(q);
    });
  });

  async function fetchConfigSites() {
    await refetchConfig();
  }

  /** 手动刷新指标：共享 query 自带去重，直接 refetch 即可 */
  async function fetchSiteMetrics() {
    if (metricsHosts.value.length === 0) return;
    await refetchSiteMetrics();
  }

  function handleRefreshSiteMetrics() {
    void fetchSiteMetrics();
  }

  function openConfigWorkbench(mode?: 'blocks' | 'waf' | 'raw' | 'preview') {
    router.push({
      name: 'CaddyConfig',
      query: buildConfigWorkbenchQuery({ mode }),
    });
  }

  function openWaf() {
    openConfigWorkbench('waf');
  }

  function openAccessLogs(domains: string[] | undefined, event?: Event) {
    event?.stopPropagation();
    event?.preventDefault();
    const query = buildAccessLogQuery(domains);
    if (!query.host) {
      message.warning('该站点未配置可用域名，无法过滤访问日志');
      return;
    }
    router.push({
      name: 'LogsAccess',
      query,
    });
  }

  /** 卡片今日统计展示：window=24h 契约字段 total/blocked，后端未上线时按 — 降级 */
  function todayDisplay(site: CatalogSiteCard, kind: 'total' | 'blocked') {
    if (loadingSiteMetrics.value && !siteMetricsLoaded.value) return '…';
    if (siteMetricsError.value) return '—';
    const metrics = siteMetricsForDomains(siteMetricsMap.value, site.domains);
    if (!metrics) return '—';
    return formatMetricCount(kind === 'total' ? metrics.total : metrics.blocked);
  }

  /** 打开站点详情整页（对齐雷池，取代早期的抽屉形态） */
  function openSiteDetail(site: CatalogSiteCard) {
    const host = site.primaryHost || sitePrimaryHost(site.domains);
    if (!host) {
      message.warning('该站点无可识别域名，请用配置工作台编辑');
      return;
    }
    router.push({
      name: 'CaddySiteDetail',
      query: { host },
    });
  }

  /** 工具条「刷新」：站点配置 + 今日指标一起重取 */
  async function handleRefreshSites() {
    await fetchConfigSites();
    handleRefreshSiteMetrics();
  }

  function setKeyword(value: string) {
    keyword.value = value;
  }

  return {
    loadingConfig,
    configErrorMessage,
    siteCards,
    configRaw,
    configHash,
    keyword,
    siteMetricsMap,
    loadingSiteMetrics,
    siteMetricsError,
    siteMetricsLoaded,
    certList,
    loadingCerts,
    filteredSites,
    fetchConfigSites,
    handleRefreshSiteMetrics,
    handleRefreshSites,
    openConfigWorkbench,
    openWaf,
    openAccessLogs,
    openSiteDetail,
    todayDisplay,
    setKeyword,
  };
}
