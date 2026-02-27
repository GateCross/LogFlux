import { computed, reactive, ref, type Ref } from 'vue';
import type { PaginationProps } from 'naive-ui';
import {
  fetchWafPolicyFalsePositiveFeedbackList,
  fetchWafPolicyStats,
  type WafPolicyFalsePositiveFeedbackItem,
  type WafPolicyStatsDimensionItem,
  type WafPolicyStatsItem,
  type WafPolicyStatsTrendItem
} from '@/service/api/caddy-observe';
import { mergePolicyFeedbackCheckedRowKeys } from '../policy-feedback-draft';

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
  ensureUserNamesByIds?: (values: unknown[]) => Promise<void>;
}

function formatDateTime(date: Date) {
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

export function useWafObserve(options: UseWafObserveOptions) {
  const { crsPolicyOptions, ensureUserNamesByIds } = options;

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

  const policyFeedbackLoading = ref(false);
  const policyFeedbackTable = ref<WafPolicyFalsePositiveFeedbackItem[]>([]);
  const policyFeedbackCheckedRowKeys = ref<number[]>([]);
  const policyFeedbackPagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 10,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50]
  });
  const policyFeedbackStatusFilter = ref<'' | 'pending' | 'confirmed' | 'resolved'>('');
  const policyFeedbackAssigneeFilter = ref('');
  const policyFeedbackSLAStatusFilter = ref<'all' | 'normal' | 'overdue' | 'resolved'>('all');

  const policyStatsPolicyOptions = computed<Array<{ label: string; value: number | '' }>>(() => [
    { label: '全部策略', value: '' },
    ...crsPolicyOptions.value
  ]);

  const hasPolicyStatsDrillFilters = computed(
    () => !!(policyStatsQuery.host.trim() || policyStatsQuery.path.trim() || policyStatsQuery.method.trim())
  );
  const hasPolicyFeedbackSelection = computed(() => policyFeedbackCheckedRowKeys.value.length > 0);
  const policyFeedbackCheckedRowKeysInPage = computed(() => {
    const selectedKeySet = new Set(policyFeedbackCheckedRowKeys.value);
    return policyFeedbackTable.value.map(item => Number(item.id || 0)).filter(id => id > 0 && selectedKeySet.has(id));
  });

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

  function buildPolicyFeedbackListParams() {
    return {
      page: Number(policyFeedbackPagination.page || 1),
      pageSize: Number(policyFeedbackPagination.pageSize || 10),
      policyId: policyStatsQuery.policyId ? Number(policyStatsQuery.policyId) : undefined,
      host: policyStatsQuery.host.trim() || undefined,
      path: policyStatsQuery.path.trim() || undefined,
      method: policyStatsQuery.method.trim().toUpperCase() || undefined,
      feedbackStatus: policyFeedbackStatusFilter.value || undefined,
      assignee: policyFeedbackAssigneeFilter.value.trim() || undefined,
      slaStatus: policyFeedbackSLAStatusFilter.value || undefined
    };
  }

  async function fetchPolicyFalsePositiveFeedbacks() {
    policyFeedbackLoading.value = true;
    try {
      const { data, error } = await fetchWafPolicyFalsePositiveFeedbackList(buildPolicyFeedbackListParams());
      if (!error && data) {
        const list = data.list || [];
        await ensureUserNamesByIds?.(list.flatMap(item => [item.operator, item.processedBy, item.assignee]));
        policyFeedbackTable.value = list;
        policyFeedbackPagination.itemCount = data.total || 0;
      }
    } finally {
      policyFeedbackLoading.value = false;
    }
  }

  function resetPolicyFeedbackSelection() {
    policyFeedbackCheckedRowKeys.value = [];
  }

  function handlePolicyFeedbackPageChange(page: number) {
    policyFeedbackPagination.page = page;
    fetchPolicyFalsePositiveFeedbacks();
  }

  function handlePolicyFeedbackPageSizeChange(pageSize: number) {
    policyFeedbackPagination.pageSize = pageSize;
    policyFeedbackPagination.page = 1;
    fetchPolicyFalsePositiveFeedbacks();
  }

  function handlePolicyFeedbackStatusFilterChange() {
    policyFeedbackPagination.page = 1;
    resetPolicyFeedbackSelection();
    fetchPolicyFalsePositiveFeedbacks();
  }

  function handlePolicyFeedbackCheckedRowKeysChange(keys: Array<string | number>) {
    const currentPageIDs = policyFeedbackTable.value.map(item => Number(item.id || 0)).filter(id => id > 0);
    policyFeedbackCheckedRowKeys.value = mergePolicyFeedbackCheckedRowKeys(policyFeedbackCheckedRowKeys.value, currentPageIDs, keys);
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

    resetPolicyFeedbackSelection();
    fetchPolicyFalsePositiveFeedbacks();
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
    policyFeedbackLoading,
    policyFeedbackTable,
    policyFeedbackCheckedRowKeys,
    policyFeedbackPagination,
    policyFeedbackStatusFilter,
    policyFeedbackAssigneeFilter,
    policyFeedbackSLAStatusFilter,
    policyStatsPolicyOptions,
    hasPolicyStatsDrillFilters,
    hasPolicyFeedbackSelection,
    policyFeedbackCheckedRowKeysInPage,
    fetchPolicyStats,
    resetPolicyStatsQuery,
    clearPolicyStatsDrillFilters,
    clearPolicyStatsDrillLevel,
    fetchPolicyFalsePositiveFeedbacks,
    resetPolicyFeedbackSelection,
    handlePolicyFeedbackPageChange,
    handlePolicyFeedbackPageSizeChange,
    handlePolicyFeedbackStatusFilterChange,
    handlePolicyFeedbackCheckedRowKeysChange,
    buildCurrentPolicyStatsSnapshot
  };
}
