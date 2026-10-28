/**
 * 站点指标统一入口（目录页 / 配置工作台 / 站点详情共用）。
 *
 * 三处的请求实体完全一致：同一支 POST /caddy/logs/site-metrics、同样按 50 个 host
 * 分块、同样用 buildSiteMetricsMap 转 map。此前各自手写一份，于是并发吞请求、
 * queryKey 缺 windowMinutes 之类的问题要在三个地方各修一遍。这里收成一处：
 *
 * - 声明式 useQuery：按 key 天然去重、共享缓存，无需手写「请求序号」防并发
 * - windowMinutes 既进请求参数也进 queryKey，15 分钟（运维视角）与 1440
 *   分钟（今日视角）互不串味
 * - staleTime: 0：指标是"当下"快照，命中全局 30s 缓存会让刷新按钮刷不出新数字
 */
import { computed, type MaybeRefOrGetter, toValue } from 'vue';
import { useQuery } from '@tanstack/vue-query';

import { getSiteMetricsApi, type SiteMetricsItem } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';

import {
  buildSiteMetricsMap,
  SITE_METRICS_HOST_CHUNK,
  type SiteMetricsMap,
} from '../../catalog/service-catalog-utils';

/** 站点指标时间窗：今日（24h）业务视角 */
export const SITE_METRICS_TODAY_MINUTES = 1440;
/** 站点指标时间窗：近窗（15min）运维视角，配置编辑时看抖动 */
export const SITE_METRICS_RECENT_MINUTES = 15;

/** 按后端上限切块：单次最多 50 host */
export function chunkHosts(hosts: string[]): string[][] {
  const chunks: string[][] = [];
  for (let i = 0; i < hosts.length; i += SITE_METRICS_HOST_CHUNK) {
    chunks.push(hosts.slice(i, i + SITE_METRICS_HOST_CHUNK));
  }
  return chunks;
}

/** queryKey 用的 hosts 签名：排序后拼接，顺序不同不产生第二份缓存 */
export function hostsSignature(hosts: string[]): string {
  return hosts.toSorted().join(',');
}

export interface UseSiteMetricsQueryOptions {
  /** 待统计的 host 列表；空数组时不发请求 */
  hosts: MaybeRefOrGetter<string[]>;
  /** 时间窗分钟数，同时作为 queryKey 的隔离维度 */
  windowMinutes: number;
}

/**
 * 批量拉取站点指标。
 * @returns metricsMap host → 计数；字符串展示与聚合走 service-catalog-utils 的纯函数
 */
export function useSiteMetricsQuery(options: UseSiteMetricsQueryOptions) {
  const hostList = computed(() => toValue(options.hosts) ?? []);
  const windowMinutes = options.windowMinutes;

  const query = useQuery({
    // hosts 排序后拼 key：同一批 host 顺序不同不应产生两份缓存
    queryKey: computed(() =>
      qk.caddy.metrics(hostsSignature(hostList.value), windowMinutes),
    ),
    queryFn: async () => {
      const hosts = hostList.value;
      const items: SiteMetricsItem[] = [];
      for (const chunk of chunkHosts(hosts)) {
        const list = await getSiteMetricsApi(
          { hosts: chunk, windowMinutes },
          withListDetailErrorMode(),
        );
        items.push(...((list as SiteMetricsItem[]) ?? []));
      }
      return buildSiteMetricsMap(items, hosts);
    },
    // 指标是"当下"快照，不能用全局 30s staleTime
    staleTime: 0,
    enabled: computed(() => hostList.value.length > 0),
  });

  const metricsMap = computed<SiteMetricsMap>(() => query.data.value ?? new Map());

  return {
    metricsMap,
    /** 首次加载中（有 hosts 且还没有结果） */
    loading: computed(
      () => hostList.value.length > 0 && query.isPending.value,
    ),
    /** 是否已有结果落地，用于区分 "…"（加载中）与 "—"（失败/无数据） */
    loaded: computed(() => query.data.value !== undefined),
    errorMessage: computed(() =>
      toValue(query.error) == null
        ? ''
        : apiErrorMessage(toValue(query.error), '获取站点近窗指标失败'),
    ),
    refetch: () => query.refetch(),
  };
}
