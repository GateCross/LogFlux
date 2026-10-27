import { computed, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useDebounceFn } from '@vueuse/core';
import { useQueryClient } from '@tanstack/vue-query';
import { message } from 'antdv-next';

import {
  getCaddyConfigApi,
  getSiteMetricsApi,
  type SiteMetricsItem,
} from '#/api/caddy/server';
import { getSimpleWafConfigApi } from '#/api/caddy/simple-waf';
import { getCertificateListApi } from '#/api/caddy/certificate';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { apiErrorMessage } from '#/utils/api-error-message';

import {
  SITE_METRICS_HOST_CHUNK,
  SITE_METRICS_WINDOW_MINUTES,
  buildAccessLogQuery,
  buildCatalogSitesFromConfigPayload,
  buildConfigWorkbenchQuery,
  buildSiteMetricsMap,
  collectCatalogHosts,
  formatMetricCount,
  siteMetricsForDomains,
  sitePrimaryHost,
  type CatalogSiteCard,
  type SiteMetricsMap,
} from '../service-catalog-utils';

export function useServiceCatalogPage() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const keyword = ref('');

  /** 近窗指标：失败降级，不阻塞目录浏览 */
  const siteMetricsMap = ref<SiteMetricsMap>(new Map());
  const loadingSiteMetrics = ref(false);
  const siteMetricsError = ref('');
  const siteMetricsLoaded = ref(false);

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

  /** 全局 WAF 状态（key 与 useSimpleWaf 一致，共享缓存）：卡片防护模式徽章数据源 */
  const { data: wafStatusData } = useListDetailQuery({
    queryKey: [...qk.caddy.config(), 'simple-waf'],
    queryFn: () => getSimpleWafConfigApi(withListDetailErrorMode()),
    errorFallback: '获取 WAF 状态失败',
  });
  const wafStatus = computed(() => wafStatusData.value ?? null);

  watch(siteCards, () => {
    siteMetricsMap.value = new Map();
    siteMetricsError.value = '';
    siteMetricsLoaded.value = false;
    debouncedFetchSiteMetrics();
  });

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

  async function fetchSiteMetrics() {
    if (loadingSiteMetrics.value) return;
    const hosts = collectCatalogHosts(siteCards.value);
    if (hosts.length === 0) {
      siteMetricsMap.value = new Map();
      siteMetricsError.value = '';
      siteMetricsLoaded.value = true;
      return;
    }

    loadingSiteMetrics.value = true;
    siteMetricsError.value = '';
    try {
      const items: SiteMetricsItem[] = [];
      for (let i = 0; i < hosts.length; i += SITE_METRICS_HOST_CHUNK) {
        const chunk = hosts.slice(i, i + SITE_METRICS_HOST_CHUNK);
        // window=24h 契约：windowMinutes=1440 进 query key，与旧 15 分钟窗缓存隔离
        const list = await queryClient.fetchQuery({
          queryKey: qk.caddy.metrics(
            chunk.join(','),
            SITE_METRICS_WINDOW_MINUTES,
          ),
          queryFn: () =>
            getSiteMetricsApi(
              {
                hosts: chunk,
                windowMinutes: SITE_METRICS_WINDOW_MINUTES,
              },
              withListDetailErrorMode(),
            ),
        });
        items.push(...((list as SiteMetricsItem[]) ?? []));
      }
      siteMetricsMap.value = buildSiteMetricsMap(items, hosts);
      siteMetricsLoaded.value = true;
    } catch (error) {
      siteMetricsError.value = apiErrorMessage(error, '获取站点近窗指标失败');
      siteMetricsMap.value = new Map();
      siteMetricsLoaded.value = true;
    } finally {
      loadingSiteMetrics.value = false;
    }
  }

  const debouncedFetchSiteMetrics = useDebounceFn(fetchSiteMetrics, 600);

  function handleRefreshSiteMetrics() {
    debouncedFetchSiteMetrics();
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
    wafStatus,
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
