import { ref, type Ref } from 'vue';
import type { Router, RouteLocationNormalizedLoaded } from 'vue-router';
import type {
  WafPolicyStatsDimensionItem,
  WafPolicyStatsItem
} from '@/service/api/caddy';
import type { PolicyStatsSnapshot } from './useWafObserve';

type MessageApi = {
  success: (content: string) => void;
  warning: (content: string) => void;
};

type PolicyStatsQuery = {
  policyId: number | '' | null;
  window: '1h' | '6h' | '24h' | '7d';
  intervalSec: number;
  topN: number;
  host: string;
  path: string;
  method: string;
};

interface UseWafObserveExportOptions {
  message: MessageApi;
  route: RouteLocationNormalizedLoaded;
  router: Router;
  activeTab: Ref<string>;
  observeWindowOptions: Array<{ label: string; value: string }>;
  policyStatsQuery: PolicyStatsQuery;
  policyStatsRange: Ref<{ startTime: string; endTime: string; intervalSec: number }>;
  policyStatsSummary: Ref<WafPolicyStatsItem>;
  policyStatsTable: Ref<WafPolicyStatsItem[]>;
  policyStatsTrend: Ref<Array<{ time: string; hitCount: number; blockedCount: number; allowedCount: number }>>;
  policyStatsTopHosts: Ref<WafPolicyStatsDimensionItem[]>;
  policyStatsTopPaths: Ref<WafPolicyStatsDimensionItem[]>;
  policyStatsTopMethods: Ref<WafPolicyStatsDimensionItem[]>;
  policyStatsPreviousSnapshot: Ref<PolicyStatsSnapshot | null>;
  buildCurrentPolicyStatsSnapshot: () => PolicyStatsSnapshot;
  formatRatePercent: (value: number) => string;
  formatDateTime: (date: Date) => string;
}

const observeQueryKeys = ['policyId', 'window', 'intervalSec', 'topN', 'host', 'path', 'method'];

function pickRouteQueryValue(value: unknown) {
  if (Array.isArray(value)) {
    return String(value[0] ?? '').trim();
  }
  return String(value ?? '').trim();
}

function parseRangedInteger(value: string, fallback: number, min: number, max: number) {
  const parsed = Number.parseInt(String(value || '').trim(), 10);
  if (!Number.isFinite(parsed)) {
    return fallback;
  }
  return Math.min(max, Math.max(min, parsed));
}

function buildQuerySignature(query: Record<string, unknown>) {
  const pairs = Object.entries(query)
    .map(([key, value]) => [key, pickRouteQueryValue(value)] as const)
    .filter(([, value]) => value !== '')
    .sort(([a], [b]) => a.localeCompare(b));
  return pairs.map(([key, value]) => `${key}:${value}`).join('|');
}

function escapeCsvCell(value: unknown) {
  const text = String(value ?? '');
  if (text.includes('"') || text.includes(',') || text.includes('\n')) {
    return `"${text.replace(/"/g, '""')}"`;
  }
  return text;
}

function buildDimensionCsvRows(section: string, rows: WafPolicyStatsDimensionItem[], formatRatePercent: (value: number) => string) {
  const lines: string[] = [escapeCsvCell(section), '维度值,命中,拦截,放行,拦截率'];
  rows.forEach(row => {
    lines.push([
      escapeCsvCell(row.key || '-'),
      row.hitCount,
      row.blockedCount,
      row.allowedCount,
      escapeCsvCell(formatRatePercent(row.blockRate))
    ].join(','));
  });
  lines.push('');
  return lines;
}

function buildDimensionCompareCsvRows(
  section: string,
  currentRows: WafPolicyStatsDimensionItem[],
  previousRows: WafPolicyStatsDimensionItem[],
  formatRatePercent: (value: number) => string
) {
  const lines: string[] = [escapeCsvCell(section), '维度值,当前命中,基线命中,命中变化,当前拦截,基线拦截,拦截变化,当前放行,基线放行,放行变化,当前拦截率,基线拦截率,拦截率变化(pp)'];
  const currentMap = new Map<string, WafPolicyStatsDimensionItem>();
  const previousMap = new Map<string, WafPolicyStatsDimensionItem>();
  currentRows.forEach(item => currentMap.set(String(item.key || '-'), item));
  previousRows.forEach(item => previousMap.set(String(item.key || '-'), item));
  const allKeys = Array.from(new Set([...currentMap.keys(), ...previousMap.keys()])).sort((a, b) => a.localeCompare(b));
  allKeys.forEach(key => {
    const current = currentMap.get(key);
    const previous = previousMap.get(key);
    const currentHit = Number(current?.hitCount || 0);
    const previousHit = Number(previous?.hitCount || 0);
    const currentBlocked = Number(current?.blockedCount || 0);
    const previousBlocked = Number(previous?.blockedCount || 0);
    const currentAllowed = Number(current?.allowedCount || 0);
    const previousAllowed = Number(previous?.allowedCount || 0);
    const currentRate = Number(current?.blockRate || 0);
    const previousRate = Number(previous?.blockRate || 0);
    lines.push([
      escapeCsvCell(key || '-'),
      currentHit,
      previousHit,
      currentHit - previousHit,
      currentBlocked,
      previousBlocked,
      currentBlocked - previousBlocked,
      currentAllowed,
      previousAllowed,
      currentAllowed - previousAllowed,
      escapeCsvCell(formatRatePercent(currentRate)),
      escapeCsvCell(formatRatePercent(previousRate)),
      `${((currentRate - previousRate) * 100).toFixed(2)}pp`
    ].join(','));
  });
  lines.push('');
  return lines;
}

export function useWafObserveExport(options: UseWafObserveExportOptions) {
  const {
    message,
    route,
    router,
    activeTab,
    observeWindowOptions,
    policyStatsQuery,
    policyStatsRange,
    policyStatsSummary,
    policyStatsTable,
    policyStatsTrend,
    policyStatsTopHosts,
    policyStatsTopPaths,
    policyStatsTopMethods,
    policyStatsPreviousSnapshot,
    buildCurrentPolicyStatsSnapshot,
    formatRatePercent,
    formatDateTime
  } = options;

  const observeWindowValueSet = new Set(observeWindowOptions.map(item => item.value));
  const observeRouteSyncing = ref(false);

  function applyObserveQueryFromRoute(query: Record<string, unknown>) {
    if (activeTab.value !== 'observe') {
      return false;
    }

    const nextPolicyIdRaw = pickRouteQueryValue(query.policyId);
    const nextPolicyIdParsed = Number.parseInt(nextPolicyIdRaw, 10);
    const nextPolicyId = Number.isInteger(nextPolicyIdParsed) && nextPolicyIdParsed > 0 ? nextPolicyIdParsed : '';
    const nextWindowRaw = pickRouteQueryValue(query.window);
    const nextWindow = observeWindowValueSet.has(nextWindowRaw)
      ? (nextWindowRaw as (typeof policyStatsQuery)['window'])
      : '24h';
    const nextIntervalSec = parseRangedInteger(pickRouteQueryValue(query.intervalSec), 300, 60, 86400);
    const nextTopN = parseRangedInteger(pickRouteQueryValue(query.topN), 8, 1, 50);
    const nextHost = pickRouteQueryValue(query.host);
    const nextPath = pickRouteQueryValue(query.path);
    const nextMethod = pickRouteQueryValue(query.method).toUpperCase();

    const changed =
      policyStatsQuery.policyId !== nextPolicyId ||
      policyStatsQuery.window !== nextWindow ||
      Number(policyStatsQuery.intervalSec) !== nextIntervalSec ||
      Number(policyStatsQuery.topN) !== nextTopN ||
      policyStatsQuery.host !== nextHost ||
      policyStatsQuery.path !== nextPath ||
      policyStatsQuery.method !== nextMethod;

    if (changed) {
      policyStatsQuery.policyId = nextPolicyId;
      policyStatsQuery.window = nextWindow;
      policyStatsQuery.intervalSec = nextIntervalSec;
      policyStatsQuery.topN = nextTopN;
      policyStatsQuery.host = nextHost;
      policyStatsQuery.path = nextPath;
      policyStatsQuery.method = nextMethod;
    }

    return changed;
  }

  async function syncObserveStateToRouteQuery() {
    if (activeTab.value !== 'observe') {
      return;
    }

    const nextQuery: Record<string, string> = {};
    observeQueryKeys.forEach(key => {
      const value = route.query[key];
      const resolved = pickRouteQueryValue(value);
      if (resolved) {
        nextQuery[key] = resolved;
      }
    });

    if (policyStatsQuery.policyId) {
      nextQuery.policyId = String(policyStatsQuery.policyId);
    } else {
      delete nextQuery.policyId;
    }

    nextQuery.window = policyStatsQuery.window;
    nextQuery.intervalSec = String(parseRangedInteger(String(policyStatsQuery.intervalSec), 300, 60, 86400));
    nextQuery.topN = String(parseRangedInteger(String(policyStatsQuery.topN), 8, 1, 50));

    const host = policyStatsQuery.host.trim();
    const path = policyStatsQuery.path.trim();
    const method = policyStatsQuery.method.trim().toUpperCase();
    if (host) {
      nextQuery.host = host;
    } else {
      delete nextQuery.host;
    }
    if (path) {
      nextQuery.path = path;
    } else {
      delete nextQuery.path;
    }
    if (method) {
      nextQuery.method = method;
    } else {
      delete nextQuery.method;
    }

    if (buildQuerySignature(route.query as Record<string, unknown>) === buildQuerySignature(nextQuery)) {
      return;
    }

    observeRouteSyncing.value = true;
    try {
      await router.replace({ query: nextQuery });
    } finally {
      observeRouteSyncing.value = false;
    }
  }

  async function handleCopyPolicyStatsLink() {
    await syncObserveStateToRouteQuery();
    const currentUrl = window.location.href;

    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(currentUrl);
        message.success('已复制当前筛选链接');
        return;
      } catch {
        // fallback
      }
    }

    const textArea = document.createElement('textarea');
    textArea.value = currentUrl;
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    const copied = document.execCommand('copy');
    document.body.removeChild(textArea);

    if (copied) {
      message.success('已复制当前筛选链接');
    } else {
      message.warning('复制失败，请手动复制浏览器地址栏链接');
    }
  }

  function handleExportPolicyStatsCsv() {
    const lines: string[] = [
      'LogFlux WAF Policy Stats Export',
      `导出时间,${escapeCsvCell(formatDateTime(new Date()))}`,
      `统计区间开始,${escapeCsvCell(policyStatsRange.value.startTime || '-')}`,
      `统计区间结束,${escapeCsvCell(policyStatsRange.value.endTime || '-')}`,
      `趋势粒度秒,${policyStatsRange.value.intervalSec || 0}`,
      `下钻Host,${escapeCsvCell(policyStatsQuery.host || '-')}`,
      `下钻Path,${escapeCsvCell(policyStatsQuery.path || '-')}`,
      `下钻Method,${escapeCsvCell(policyStatsQuery.method || '-')}`,
      ''
    ];

    lines.push('总览');
    lines.push('策略,命中,拦截,放行,疑似误报,拦截率');
    lines.push([
      escapeCsvCell(policyStatsSummary.value.policyName || '-'),
      policyStatsSummary.value.hitCount,
      policyStatsSummary.value.blockedCount,
      policyStatsSummary.value.allowedCount,
      policyStatsSummary.value.suspectedFalsePositiveCount,
      escapeCsvCell(formatRatePercent(policyStatsSummary.value.blockRate))
    ].join(','));
    lines.push('');

    lines.push('策略统计');
    lines.push('策略,命中,拦截,放行,疑似误报,拦截率');
    policyStatsTable.value.forEach(row => {
      lines.push([
        escapeCsvCell(row.policyName || `#${row.policyId}`),
        row.hitCount,
        row.blockedCount,
        row.allowedCount,
        row.suspectedFalsePositiveCount,
        escapeCsvCell(formatRatePercent(row.blockRate))
      ].join(','));
    });
    lines.push('');

    lines.push('趋势');
    lines.push('时间,命中,拦截,放行');
    policyStatsTrend.value.forEach(row => {
      lines.push([escapeCsvCell(row.time), row.hitCount, row.blockedCount, row.allowedCount].join(','));
    });
    lines.push('');

    lines.push(...buildDimensionCsvRows('Top Host', policyStatsTopHosts.value, formatRatePercent));
    lines.push(...buildDimensionCsvRows('Top Path', policyStatsTopPaths.value, formatRatePercent));
    lines.push(...buildDimensionCsvRows('Top Method', policyStatsTopMethods.value, formatRatePercent));

    const content = `\ufeff${lines.join('\n')}`;
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `logflux-waf-policy-stats-${Date.now()}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    message.success('策略观测统计已导出');
  }

  function buildPolicyStatsSnapshotScopeText(snapshot: PolicyStatsSnapshot) {
    const policyLabel = snapshot.query.policyId ? `#${snapshot.query.policyId}` : '全部策略';
    return `策略=${policyLabel}, window=${snapshot.query.window}, interval=${snapshot.query.intervalSec}s, topN=${snapshot.query.topN}, host=${snapshot.query.host || '-'}, path=${snapshot.query.path || '-'}, method=${snapshot.query.method || '-'}`;
  }

  function handleExportPolicyStatsCompareCsv() {
    const previous = policyStatsPreviousSnapshot.value;
    if (!previous) {
      message.warning('暂无可对比的历史快照');
      return;
    }

    const current = buildCurrentPolicyStatsSnapshot();
    const delta = (next: number, prev: number) => Number(next || 0) - Number(prev || 0);
    const lines: string[] = [
      'LogFlux WAF Policy Stats Compare Export',
      `导出时间,${escapeCsvCell(formatDateTime(new Date()))}`,
      `当前快照时间,${escapeCsvCell(current.capturedAt)}`,
      `对比基线时间,${escapeCsvCell(previous.capturedAt)}`,
      `当前筛选,${escapeCsvCell(buildPolicyStatsSnapshotScopeText(current))}`,
      `基线筛选,${escapeCsvCell(buildPolicyStatsSnapshotScopeText(previous))}`,
      ''
    ];

    lines.push('总览对比');
    lines.push('指标,当前,基线,变化');
    lines.push(['命中', current.summary.hitCount, previous.summary.hitCount, delta(current.summary.hitCount, previous.summary.hitCount)].join(','));
    lines.push(['拦截', current.summary.blockedCount, previous.summary.blockedCount, delta(current.summary.blockedCount, previous.summary.blockedCount)].join(','));
    lines.push(['放行', current.summary.allowedCount, previous.summary.allowedCount, delta(current.summary.allowedCount, previous.summary.allowedCount)].join(','));
    lines.push([
      '疑似误报',
      current.summary.suspectedFalsePositiveCount,
      previous.summary.suspectedFalsePositiveCount,
      delta(current.summary.suspectedFalsePositiveCount, previous.summary.suspectedFalsePositiveCount)
    ].join(','));
    lines.push([
      '拦截率',
      escapeCsvCell(formatRatePercent(current.summary.blockRate)),
      escapeCsvCell(formatRatePercent(previous.summary.blockRate)),
      `${(delta(current.summary.blockRate, previous.summary.blockRate) * 100).toFixed(2)}pp`
    ].join(','));
    lines.push('');

    lines.push('策略维度对比');
    lines.push('策略,当前命中,基线命中,命中变化,当前拦截,基线拦截,拦截变化');
    const currentMap = new Map<number, WafPolicyStatsItem>();
    const previousMap = new Map<number, WafPolicyStatsItem>();
    current.list.forEach(item => currentMap.set(Number(item.policyId || 0), item));
    previous.list.forEach(item => previousMap.set(Number(item.policyId || 0), item));
    const allPolicyIds = Array.from(new Set([...currentMap.keys(), ...previousMap.keys()])).sort((a, b) => a - b);
    allPolicyIds.forEach(policyId => {
      const currentItem = currentMap.get(policyId);
      const previousItem = previousMap.get(policyId);
      const policyName = currentItem?.policyName || previousItem?.policyName || `#${policyId}`;
      const currentHit = Number(currentItem?.hitCount || 0);
      const previousHit = Number(previousItem?.hitCount || 0);
      const currentBlocked = Number(currentItem?.blockedCount || 0);
      const previousBlocked = Number(previousItem?.blockedCount || 0);
      lines.push([
        escapeCsvCell(policyName),
        currentHit,
        previousHit,
        delta(currentHit, previousHit),
        currentBlocked,
        previousBlocked,
        delta(currentBlocked, previousBlocked)
      ].join(','));
    });
    lines.push('');

    lines.push(...buildDimensionCompareCsvRows('Top Host 对比', current.topHosts, previous.topHosts, formatRatePercent));
    lines.push(...buildDimensionCompareCsvRows('Top Path 对比', current.topPaths, previous.topPaths, formatRatePercent));
    lines.push(...buildDimensionCompareCsvRows('Top Method 对比', current.topMethods, previous.topMethods, formatRatePercent));

    const content = `\ufeff${lines.join('\n')}`;
    const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `logflux-waf-policy-stats-compare-${Date.now()}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    message.success('策略观测对比统计已导出');
  }

  return {
    observeRouteSyncing,
    applyObserveQueryFromRoute,
    syncObserveStateToRouteQuery,
    handleCopyPolicyStatsLink,
    handleExportPolicyStatsCsv,
    handleExportPolicyStatsCompareCsv
  };
}
