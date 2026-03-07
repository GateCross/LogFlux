import { type Ref, watch } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import type { WafPolicyStatsDimensionItem } from '@/service/api/caddy-observe';

export type PolicyStatsDimensionType = 'host' | 'path' | 'method';

type MessageApi = {
  warning: (content: string) => void;
};

interface UseObserveDrilldownOptions {
  message: MessageApi;
  route: RouteLocationNormalizedLoaded;
  activeTab: Ref<string>;
  observeRouteSyncing: Ref<boolean>;
  policyStatsQuery: {
    policyId: number | '' | null;
    window: '1h' | '6h' | '24h' | '7d';
    intervalSec: number;
    topN: number;
    host: string;
    path: string;
    method: string;
  };
  applyObserveQueryFromRoute: (query: Record<string, unknown>) => boolean;
  syncObserveStateToRouteQuery: () => Promise<void>;
  fetchPolicyStats: () => void | Promise<void>;
}

export function useObserveDrilldown(options: UseObserveDrilldownOptions) {
  const {
    message,
    route,
    activeTab,
    observeRouteSyncing,
    policyStatsQuery,
    applyObserveQueryFromRoute,
    syncObserveStateToRouteQuery,
    fetchPolicyStats
  } = options;

  function normalizePolicyStatsDrillValue(type: PolicyStatsDimensionType, raw: string) {
    const text = String(raw || '').trim();
    if (type === 'host') {
      if (text === '(empty)') return '(empty)';
      return text.toLowerCase();
    }
    if (type === 'method') {
      return text.toUpperCase();
    }
    return text;
  }

  function isPolicyStatsDrillUnlocked(type: PolicyStatsDimensionType) {
    if (type === 'host') return true;
    if (type === 'path') return Boolean(policyStatsQuery.host.trim());
    return Boolean(policyStatsQuery.host.trim() && policyStatsQuery.path.trim());
  }

  function policyStatsDrillStatusLabel(type: PolicyStatsDimensionType) {
    if (type === 'host') return '入口层';
    return isPolicyStatsDrillUnlocked(type) ? '已解锁' : '待解锁';
  }

  function policyStatsDrillHint(type: PolicyStatsDimensionType) {
    if (type === 'host') {
      return '第一层下钻入口：点击 Host 可进入 Host 维度过滤。';
    }
    if (type === 'path') {
      if (!isPolicyStatsDrillUnlocked(type)) {
        return '待解锁：请先在 Top Host 中选择一个 Host。';
      }
      return `已解锁：当前 Host=${policyStatsQuery.host || '-'}，点击 Path 继续下钻。`;
    }
    if (!isPolicyStatsDrillUnlocked(type)) {
      return '待解锁：请先完成 Host + Path 下钻。';
    }
    return `已解锁：当前 Host=${policyStatsQuery.host || '-'}，Path=${policyStatsQuery.path || '-'}。点击 Method 继续下钻。`;
  }

  function canPolicyStatsDrillDimension(type: PolicyStatsDimensionType) {
    if (type === 'host') return true;
    if (type === 'path') return Boolean(policyStatsQuery.host.trim());
    return Boolean(policyStatsQuery.host.trim() && policyStatsQuery.path.trim());
  }

  function isPolicyStatsDimensionSelected(type: PolicyStatsDimensionType, row: WafPolicyStatsDimensionItem) {
    const key = normalizePolicyStatsDrillValue(type, String(row?.key || ''));
    if (!key || key === '-') return false;
    if (type === 'host') {
      return key === normalizePolicyStatsDrillValue('host', policyStatsQuery.host);
    }
    if (type === 'path') {
      return key === normalizePolicyStatsDrillValue('path', policyStatsQuery.path);
    }
    return key === normalizePolicyStatsDrillValue('method', policyStatsQuery.method);
  }

  function handlePolicyStatsDimensionDrill(type: PolicyStatsDimensionType, row: WafPolicyStatsDimensionItem) {
    const key = String(row?.key || '').trim();
    if (!key || key === '-') {
      return;
    }

    if (!canPolicyStatsDrillDimension(type)) {
      if (type === 'path') {
        message.warning('请先从 Top Host 选择一个 Host，再下钻 Path');
      } else if (type === 'method') {
        message.warning('请先完成 Host + Path 下钻，再下钻 Method');
      }
      return;
    }

    if (type === 'host') {
      policyStatsQuery.host = key;
      policyStatsQuery.path = '';
      policyStatsQuery.method = '';
    } else if (type === 'path') {
      policyStatsQuery.path = key;
      policyStatsQuery.method = '';
    } else {
      policyStatsQuery.method = key;
    }

    fetchPolicyStats();
  }

  function buildPolicyStatsDimensionRowProps(type: PolicyStatsDimensionType) {
    return (row: WafPolicyStatsDimensionItem) => {
      const clickable = canPolicyStatsDrillDimension(type);
      const selected = isPolicyStatsDimensionSelected(type, row);
      const styleParts = ['transition: background-color 0.2s ease'];
      if (clickable) {
        styleParts.push('cursor: pointer');
      } else {
        styleParts.push('cursor: not-allowed');
        styleParts.push('opacity: 0.65');
      }
      if (selected) {
        styleParts.push('background: rgba(24, 160, 88, 0.14)');
        styleParts.push('font-weight: 600');
        styleParts.push('box-shadow: inset 3px 0 0 rgba(24, 160, 88, 0.9)');
      }
      const lockedHint = type === 'path' ? '请先从 Top Host 选择一个 Host' : '请先完成 Host + Path 下钻';
      return {
        style: styleParts.join(';'),
        title: clickable ? '点击下钻' : lockedHint,
        onClick: () => {
          if (!clickable) return;
          handlePolicyStatsDimensionDrill(type, row);
        }
      };
    };
  }

  watch(
    () => route.query,
    query => {
      if (observeRouteSyncing.value) {
        return;
      }
      const prevTab = activeTab.value;
      const queryChanged = applyObserveQueryFromRoute(query as Record<string, unknown>);
      if (queryChanged && prevTab === 'observe' && activeTab.value === 'observe') {
        fetchPolicyStats();
      }
    },
    { immediate: true }
  );

  watch(
    () => [
      activeTab.value,
      policyStatsQuery.policyId,
      policyStatsQuery.window,
      policyStatsQuery.intervalSec,
      policyStatsQuery.topN,
      policyStatsQuery.host,
      policyStatsQuery.path,
      policyStatsQuery.method
    ],
    () => {
      if (observeRouteSyncing.value) {
        return;
      }
      syncObserveStateToRouteQuery().catch(() => undefined);
    }
  );

  return {
    policyStatsDrillHint,
    policyStatsDrillStatusLabel,
    isPolicyStatsDrillUnlocked,
    buildPolicyStatsDimensionRowProps
  };
}
