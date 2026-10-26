import { computed, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { useDebounceFn } from '@vueuse/core';
import { useQueryClient } from '@tanstack/vue-query';
import { message } from 'antdv-next';

import {
  getCaddyConfigApi,
  getCaddyServerListApi,
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
import { useCaddyServerStatus } from '../../shared/use-caddy-server-status';

export function serverLabel(server: Record<string, any>) {
  return server.name || server.url || `Server #${server.id}`;
}

export function useServiceCatalogPage() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const selectedServerId = ref<number>();
  const keyword = ref('');

  /** 近窗指标：失败降级，不阻塞目录浏览 */
  const siteMetricsMap = ref<SiteMetricsMap>(new Map());
  const loadingSiteMetrics = ref(false);
  const siteMetricsError = ref('');
  const siteMetricsLoaded = ref(false);

  const {
    data: serversData,
    loading: loadingServers,
    errorMessage: serversErrorMessage,
    refetch: refetchServers,
  } = useListDetailQuery({
    queryKey: qk.caddy.servers(),
    queryFn: () => getCaddyServerListApi(withListDetailErrorMode()),
    errorFallback: '获取服务器列表失败',
  });

  const servers = computed(() => serversData.value ?? []);

  const {
    loadingServerStatus,
    serverStatusError,
    serverStatusLoaded,
    lastProbedAt,
    selectedServerStatus,
    serverStatusRows,
    pruneByServers,
    handleRefreshServerStatus,
  } = useCaddyServerStatus({
    servers,
    selectedServerId,
    labelOf: serverLabel,
  });

  watch(
    servers,
    (list) => {
      if (list.length > 0 && !selectedServerId.value) {
        selectedServerId.value = Number(list[0]?.id);
      }
      pruneByServers(list);
      if (list.length === 0) {
        siteMetricsMap.value = new Map();
      }
    },
    { immediate: true },
  );

  const {
    data: configData,
    loading: loadingConfig,
    errorMessage: configErrorMessage,
    refetch: refetchConfig,
  } = useListDetailQuery({
    queryKey: computed(() =>
      qk.caddy.catalog(selectedServerId.value ?? 0),
    ),
    queryFn: () =>
      getCaddyConfigApi(selectedServerId.value!, withListDetailErrorMode()),
    errorFallback: '获取站点配置失败',
    enabled: computed(() => Boolean(selectedServerId.value)),
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
    queryKey: computed(() => qk.caddy.certificates(selectedServerId.value)),
    queryFn: () =>
      getCertificateListApi(selectedServerId.value, withListDetailErrorMode()),
    errorFallback: '获取证书列表失败',
    enabled: computed(() => Boolean(selectedServerId.value)),
  });
  const certList = computed(() => certData.value ?? []);

  /** 全局 WAF 状态（key 与 useSimpleWaf 一致，共享缓存）：卡片防护模式徽章数据源 */
  const { data: wafStatusData } = useListDetailQuery({
    queryKey: computed(() => [...qk.caddy.config(selectedServerId.value ?? 0), 'simple-waf']),
    queryFn: () =>
      getSimpleWafConfigApi(selectedServerId.value, withListDetailErrorMode()),
    errorFallback: '获取 WAF 状态失败',
    enabled: computed(() => Boolean(selectedServerId.value)),
  });
  const wafStatus = computed(() => wafStatusData.value ?? null);

  watch(siteCards, () => {
    siteMetricsMap.value = new Map();
    siteMetricsError.value = '';
    siteMetricsLoaded.value = false;
    debouncedFetchSiteMetrics();
  });

  const serverOptions = computed(() =>
    servers.value.map((server) => ({
      label: serverLabel(server),
      value: Number(server.id),
    })),
  );

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

  async function fetchServers() {
    await refetchServers();
  }

  async function fetchConfigSites() {
    if (!selectedServerId.value) return;
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
            selectedServerId.value ?? 0,
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

  function handleServerChange(value: unknown) {
    const next = Number(value);
    if (!Number.isFinite(next)) return;
    selectedServerId.value = next;
  }

  function selectServerFromStatus(id: number) {
    if (!Number.isFinite(id)) return;
    selectedServerId.value = id;
  }

  function openConfigWorkbench(mode?: 'blocks' | 'waf' | 'raw' | 'preview') {
    router.push({
      name: 'CaddyConfig',
      query: buildConfigWorkbenchQuery({
        serverId: selectedServerId.value,
        mode,
      }),
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
    if (!selectedServerId.value) return;
    const host = site.primaryHost || sitePrimaryHost(site.domains);
    if (!host) {
      message.warning('该站点无可识别域名，请用配置工作台编辑');
      return;
    }
    router.push({
      name: 'CaddySiteDetail',
      query: { serverId: String(selectedServerId.value), host },
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
    servers,
    selectedServerId,
    loadingServers,
    loadingConfig,
    serversErrorMessage,
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
    serverOptions,
    loadingServerStatus,
    serverStatusError,
    serverStatusLoaded,
    lastProbedAt,
    selectedServerStatus,
    serverStatusRows,
    filteredSites,
    fetchConfigSites,
    fetchServers,
    handleRefreshServerStatus,
    handleRefreshSiteMetrics,
    handleRefreshSites,
    handleServerChange,
    selectServerFromStatus,
    openConfigWorkbench,
    openWaf,
    openAccessLogs,
    openSiteDetail,
    todayDisplay,
    setKeyword,
  };
}
