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
  siteMetricsForHost,
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
  const kindFilter = ref<'all' | 'simple' | 'complex'>('all');

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
    onlineCount,
    offlineCount,
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
      if (kindFilter.value !== 'all' && site.kind !== kindFilter.value) return false;
      if (!q) return true;
      const hay = [site.name, site.primaryHost, ...(site.domains ?? []), site.upstream ?? '']
        .join(' ')
        .toLowerCase();
      return hay.includes(q);
    });
  });

  const summary = computed(() => {
    const total = siteCards.value.length;
    const simple = siteCards.value.filter((s) => s.kind === 'simple').length;
    const complex = total - simple;
    return {
      total,
      simple,
      complex,
      online: onlineCount.value,
      offline: offlineCount.value,
    };
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
        const list = await queryClient.fetchQuery({
          queryKey: qk.caddy.metrics(selectedServerId.value ?? 0, chunk.join(',')),
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
      name: 'CaddyLog',
      query,
    });
  }

  function openSiteInWorkbench(site: CatalogSiteCard) {
    openConfigWorkbench('blocks');
    void site;
  }

  function metricDisplay(site: CatalogSiteCard, kind: '4xx' | '5xx') {
    if (loadingSiteMetrics.value && !siteMetricsLoaded.value) return '…';
    if (siteMetricsError.value) return '—';
    const metrics = siteMetricsForHost(siteMetricsMap.value, site.domains);
    return formatMetricCount(kind === '5xx' ? metrics?.count5xx : metrics?.count4xx);
  }

  function setKeyword(value: string) {
    keyword.value = value;
  }

  function setKindFilter(value: 'all' | 'simple' | 'complex') {
    kindFilter.value = value;
  }

  return {
    SITE_METRICS_WINDOW_MINUTES,
    servers,
    selectedServerId,
    loadingServers,
    loadingConfig,
    serversErrorMessage,
    configErrorMessage,
    siteCards,
    keyword,
    kindFilter,
    siteMetricsMap,
    loadingSiteMetrics,
    siteMetricsError,
    siteMetricsLoaded,
    serverOptions,
    loadingServerStatus,
    serverStatusError,
    serverStatusLoaded,
    lastProbedAt,
    selectedServerStatus,
    serverStatusRows,
    filteredSites,
    summary,
    fetchConfigSites,
    fetchServers,
    handleRefreshServerStatus,
    handleRefreshSiteMetrics,
    handleServerChange,
    selectServerFromStatus,
    openConfigWorkbench,
    openWaf,
    openAccessLogs,
    openSiteInWorkbench,
    metricDisplay,
    setKeyword,
    setKindFilter,
  };
}
