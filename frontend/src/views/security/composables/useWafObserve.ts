import { computed, reactive, ref, type Ref } from 'vue';
import {
  fetchWafPolicyStats,
  type WafPolicyStatsDimensionItem,
  type WafPolicyStatsItem,
  type WafPolicyStatsTrendItem
} from '@/service/api/caddy';

export type PolicyStatsDimensionType = 'host' | 'path' | 'method';

export type PolicyStatsSnapshot = {
  capturedAt: string;
  query: {
    policyId: number | '' | null;
    window: '1h' | '6h' | '24h' | '7d';
    intervalSec: number;
    topN: number;
    host: string;
    path: string;
    method: string;
  };
  range: {
    startTime: string;
    endTime: string;
    intervalSec: number;
  };
  summary: WafPolicyStatsItem;
  list: WafPolicyStatsItem[];
  trend: WafPolicyStatsTrendItem[];
  topHosts: WafPolicyStatsDimensionItem[];
  topPaths: WafPolicyStatsDimensionItem[];
  topMethods: WafPolicyStatsDimensionItem[];
};

interface UseWafObserveOptions {
  crsPolicyOptions: Ref<Array<{ label: string; value: number }>>;
  onAfterStatsLoaded?: () => void;
}

function formatDateTime(date: Date) {
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

export function useWafObserve(options: UseWafObserveOptions) {
  const { crsPolicyOptions, onAfterStatsLoaded } = options;

  const observeWindowOptions = [
    { label: '最近 1 小时', value: '1h' },
    { label: '最近 6 小时', value: '6h' },
    { label: '最近 24 小时', value: '24h' },
    { label: '最近 7 天', value: '7d' }
  ];

  const policyStatsQuery = reactive({
    policyId: '' as number | '' | null,
    window: '24h' as '1h' | '6h' | '24h' | '7d',
    intervalSec: 300,
    topN: 8,
    host: '',
    path: '',
    method: ''
  });

  const policyStatsLoading = ref(false);
  const policyStatsSummary = ref<WafPolicyStatsItem>({
    policyId: 0,
    policyName: '全部策略',
    hitCount: 0,
    blockedCount: 0,
    allowedCount: 0,
    suspectedFalsePositiveCount: 0,
    blockRate: 0
  });
  const policyStatsTable = ref<WafPolicyStatsItem[]>([]);
  const policyStatsTrend = ref<WafPolicyStatsTrendItem[]>([]);
  const policyStatsTopHosts = ref<WafPolicyStatsDimensionItem[]>([]);
  const policyStatsTopPaths = ref<WafPolicyStatsDimensionItem[]>([]);
  const policyStatsTopMethods = ref<WafPolicyStatsDimensionItem[]>([]);
  const policyStatsRange = ref({ startTime: '', endTime: '', intervalSec: 300 });
  const policyStatsPreviousSnapshot = ref<PolicyStatsSnapshot | null>(null);

  const policyStatsPolicyOptions = computed<Array<{ label: string; value: number | '' }>>(() => [
    { label: '全部策略', value: '' },
    ...crsPolicyOptions.value
  ]);

  const hasPolicyStatsDrillFilters = computed(
    () => !!(policyStatsQuery.host.trim() || policyStatsQuery.path.trim() || policyStatsQuery.method.trim())
  );

  function resolvePolicyStatsWindowRange() {
    const end = new Date();
    const start = new Date(end.getTime());
    switch (policyStatsQuery.window) {
      case '1h':
        start.setHours(start.getHours() - 1);
        break;
      case '6h':
        start.setHours(start.getHours() - 6);
        break;
      case '7d':
        start.setDate(start.getDate() - 7);
        break;
      default:
        start.setDate(start.getDate() - 1);
        break;
    }
    return {
      startTime: formatDateTime(start),
      endTime: formatDateTime(end)
    };
  }

  function buildCurrentPolicyStatsSnapshot(): PolicyStatsSnapshot {
    return {
      capturedAt: formatDateTime(new Date()),
      query: {
        policyId: policyStatsQuery.policyId,
        window: policyStatsQuery.window,
        intervalSec: Number(policyStatsQuery.intervalSec || 300),
        topN: Number(policyStatsQuery.topN || 8),
        host: policyStatsQuery.host.trim(),
        path: policyStatsQuery.path.trim(),
        method: policyStatsQuery.method.trim().toUpperCase()
      },
      range: {
        startTime: policyStatsRange.value.startTime || '',
        endTime: policyStatsRange.value.endTime || '',
        intervalSec: Number(policyStatsRange.value.intervalSec || 0)
      },
      summary: { ...policyStatsSummary.value },
      list: policyStatsTable.value.map(item => ({ ...item })),
      trend: policyStatsTrend.value.map(item => ({ ...item })),
      topHosts: policyStatsTopHosts.value.map(item => ({ ...item })),
      topPaths: policyStatsTopPaths.value.map(item => ({ ...item })),
      topMethods: policyStatsTopMethods.value.map(item => ({ ...item }))
    };
  }

  function shouldCapturePolicyStatsSnapshot() {
    if (policyStatsTable.value.length > 0 || policyStatsTrend.value.length > 0) {
      return true;
    }
    if (Number(policyStatsSummary.value.hitCount || 0) > 0) {
      return true;
    }
    return !!(policyStatsRange.value.startTime || policyStatsRange.value.endTime);
  }

  async function fetchPolicyStats() {
    const previousSnapshot = shouldCapturePolicyStatsSnapshot() ? buildCurrentPolicyStatsSnapshot() : null;
    policyStatsLoading.value = true;
    try {
      const { startTime, endTime } = resolvePolicyStatsWindowRange();
      const { data, error } = await fetchWafPolicyStats({
        policyId: policyStatsQuery.policyId ? Number(policyStatsQuery.policyId) : undefined,
        startTime,
        endTime,
        intervalSec: Number(policyStatsQuery.intervalSec || 300),
        topN: Number(policyStatsQuery.topN || 8),
        host: policyStatsQuery.host.trim() || undefined,
        path: policyStatsQuery.path.trim() || undefined,
        method: policyStatsQuery.method.trim() || undefined
      });
      if (!error && data) {
        policyStatsSummary.value = data.summary || {
          policyId: 0,
          policyName: '全部策略',
          hitCount: 0,
          blockedCount: 0,
          allowedCount: 0,
          suspectedFalsePositiveCount: 0,
          blockRate: 0
        };
        policyStatsTable.value = data.list || [];
        policyStatsTrend.value = data.trend || [];
        policyStatsTopHosts.value = data.topHosts || [];
        policyStatsTopPaths.value = data.topPaths || [];
        policyStatsTopMethods.value = data.topMethods || [];
        policyStatsRange.value = data.range || { startTime: '', endTime: '', intervalSec: Number(policyStatsQuery.intervalSec || 300) };
        policyStatsPreviousSnapshot.value = previousSnapshot;
      }
    } finally {
      policyStatsLoading.value = false;
    }
    onAfterStatsLoaded?.();
  }

  function resetPolicyStatsQuery() {
    policyStatsQuery.policyId = '';
    policyStatsQuery.window = '24h';
    policyStatsQuery.intervalSec = 300;
    policyStatsQuery.topN = 8;
    policyStatsQuery.host = '';
    policyStatsQuery.path = '';
    policyStatsQuery.method = '';
    fetchPolicyStats();
  }

  function clearPolicyStatsDrillFilters() {
    policyStatsQuery.host = '';
    policyStatsQuery.path = '';
    policyStatsQuery.method = '';
    fetchPolicyStats();
  }

  function clearPolicyStatsDrillLevel(level: PolicyStatsDimensionType) {
    if (level === 'host') {
      policyStatsQuery.host = '';
      policyStatsQuery.path = '';
      policyStatsQuery.method = '';
    } else if (level === 'path') {
      policyStatsQuery.path = '';
      policyStatsQuery.method = '';
    } else {
      policyStatsQuery.method = '';
    }
    fetchPolicyStats();
  }

  return {
    observeWindowOptions,
    policyStatsQuery,
    policyStatsLoading,
    policyStatsSummary,
    policyStatsTable,
    policyStatsTrend,
    policyStatsTopHosts,
    policyStatsTopPaths,
    policyStatsTopMethods,
    policyStatsRange,
    policyStatsPreviousSnapshot,
    policyStatsPolicyOptions,
    hasPolicyStatsDrillFilters,
    fetchPolicyStats,
    resetPolicyStatsQuery,
    clearPolicyStatsDrillFilters,
    clearPolicyStatsDrillLevel,
    buildCurrentPolicyStatsSnapshot
  };
}
