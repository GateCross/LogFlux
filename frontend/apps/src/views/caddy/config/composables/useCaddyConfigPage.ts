import { computed, onBeforeUnmount, onMounted, watch } from 'vue';
import { Modal } from 'antdv-next';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';

import type { CaddyPageMode } from '../types';
import { useCaddyConfigIO } from './useCaddyConfigIO';
import { useSimpleWaf } from './useSimpleWaf';
import { formatMetricCount } from '../../catalog/service-catalog-utils';
import { useSiteMetrics } from './useSiteMetrics';
import { useConfigHistory } from './useConfigHistory';

/** 深链：/caddy/config?mode= */
function readQueryText(raw: unknown): string {
  return Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '');
}

export function useCaddyConfigPage() {
  const route = useRoute();
  const router = useRouter();

  function queryMode(): CaddyPageMode | undefined {
    const text = readQueryText(route.query.mode);
    if (
      text === 'blocks' ||
      text === 'waf' ||
      text === 'raw' ||
      text === 'preview'
    ) {
      return text;
    }
    return undefined;
  }

  /** metrics 与 configIO 互相回调：可变 holder 避免初始化顺序问题 */
  const metricsHooks = {
    reset: () => {},
    ready: () => {},
  };

  const configApi = useCaddyConfigIO({
    resolveRouteMode: queryMode,
    onConfigReset: () => metricsHooks.reset(),
    onConfigReady: () => metricsHooks.ready(),
  });

  const {
    loadingConfig,
    configErrorMessage,
    saving,
    previewing,
    configContent,
    configHash,
    mode,
    formModel,
    quickSiteDrafts,
    complexSiteSummaries,
    activeQuickSiteId,
    savePreview,
    dockerDiscoveryOpen,
    dockerDiscoveryScanning,
    dockerDiscoveryResult,
    modeOptions,
    quickModeOptions,
    tlsModeOptions,
    lbPolicyOptions,
    upstreamPoolOptions,
    activeQuickSite,
    globalPreservedBlocks,
    preservedSiteBlocks,
    mergedQuickFormModel,
    previewConfig,
    quickValidationErrors,
    applyRouteModeDeepLink,
    parseRawToBlocks,
    fetchConfig,
    addQuickSite,
    openDockerDiscovery,
    handleDockerDiscoveryScan,
    handleDiscoveryCommitDrafts,
    handleDiscoveryPreview,
    handleDiscoveryApply,
    duplicateQuickSite,
    removeQuickSite,
    addUpstream,
    removeUpstream,
    isUpstreamHealthEnabled,
    setUpstreamHealthEnabled,
    ensureUpstreamHealth,
    isSiteHealthEnabled,
    setSiteHealthEnabled,
    ensureSiteHealth,
    applyPreset,
    handleModeChange,
    previewBeforeSave,
    confirmSave,
    blockKindLabel,
    blockKindColor,
  } = configApi;

  const metricsApi = useSiteMetrics({
    quickSiteDrafts,
    complexSiteSummaries,
  });

  const {
    loadingSiteMetrics,
    siteMetricsError,
    siteMetricsLoaded,
    sitePrimaryHost,
    siteMetricsForDomains,
    resetSiteMetrics,
    fetchSiteMetrics,
    handleRefreshSiteMetrics,
    openAccessLogs,
  } = metricsApi;

  metricsHooks.reset = resetSiteMetrics;
  metricsHooks.ready = () => {
    void fetchSiteMetrics();
  };

  const wafApi = useSimpleWaf({
    currentConfig: configContent,
    baseHash: configHash,
    onApplied: () => fetchConfig(),
  });

  const {
    wafLoading,
    wafSaving,
    wafApplying,
    wafPreviewing,
    wafStatus,
    wafPreviewVisible,
    wafPreviewResult,
    wafLastApply,
    wafErrorMessage,
    wafForm,
    wafAvailableSites,
    wafSiteStates,
    wafDirty,
    wafStatusText,
    wafStatusColor,
    wafDraftPending,
    wafPreviewDiff,
    wafValidationError,
    wafModeOptions,
    wafStrengthOptions,
    wafAuditOptions,
    wafLoadState,
    wafActionsDisabled,
    isWafStale,
    fetchWafConfig,
    saveWafConfig,
    previewWafConfig,
    applyWafConfig,
  } = wafApi;

  const historyApi = useConfigHistory({
    previewConfig,
    configHash,
    onRolledBack: () => fetchConfig(),
  });

  const {
    historyDrawerVisible,
    historyLoading,
    historyList,
    historyDetailVisible,
    historyCompareVisible,
    historyDetail,
    historyDiffOnly,
    historyCompareRows,
    historyErrorMessage,
    historyDescription,
    diffSideClass,
    openHistory,
    openHistoryDetail,
    openHistoryCompare,
    rollbackHistory,
  } = historyApi;

  watch(mode, (value) => {
    // fetchConfig 会把 mode 重置成 blocks，所以这里按归属判断而不是「有没有数据」。
    if (value === 'waf' && isWafStale()) {
      void fetchWafConfig();
    }
    // 视角写回 URL：刷新后停在同一个 tab，也让深链可分享。
    const current = readQueryText(route.query.mode);
    if (current !== value) {
      void router.replace({
        query: { ...route.query, mode: value },
      });
    }
  });

  // 站点域名集合变化时无需手动刷新：共享 query 的 key 含 hosts，变化即自动重取

  onMounted(async () => {
    await fetchConfig();
    applyRouteModeDeepLink();
    if (mode.value === 'waf') {
      await fetchWafConfig();
    }
  });

  /** 离开页面或关标签页会丢掉 WAF 草稿，给出提示由模板层确认。 */
  onBeforeRouteLeave(async () => {
    if (!wafDirty.value) return true;
    return confirmBeforeLeave('离开配置页？');
  });

  function handleBeforeUnload(event: BeforeUnloadEvent) {
    if (!wafDirty.value) return;
    event.preventDefault();
    event.returnValue = '';
  }

  window.addEventListener('beforeunload', handleBeforeUnload);
  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', handleBeforeUnload);
  });

  // 打包 props

  const serverToolbarProps = computed(() => ({
    mode: mode.value,
    previewing: previewing.value,
    configContent: configContent.value,
  }));

  const blocksWorkbenchProps = computed(() => ({
    formModel: formModel.value,
    globalPreservedBlocks: globalPreservedBlocks.value,
    preservedSiteBlocks: preservedSiteBlocks.value,
    quickSiteDrafts: quickSiteDrafts.value,
    complexSiteSummaries: complexSiteSummaries.value,
    activeQuickSiteId: activeQuickSiteId.value,
    activeQuickSite: activeQuickSite.value,
    lbPolicyOptions,
    quickModeOptions,
    tlsModeOptions,
    upstreamPoolOptions: upstreamPoolOptions.value,
    loadingSiteMetrics: loadingSiteMetrics.value,
    siteMetricsLoaded: siteMetricsLoaded.value,
    siteMetricsError: siteMetricsError.value,
    siteMetricsForDomains,
    formatMetricCount,
    sitePrimaryHost,
    blockKindLabel,
    blockKindColor,
    isUpstreamHealthEnabled,
    setUpstreamHealthEnabled,
    ensureUpstreamHealth,
    isSiteHealthEnabled,
    setSiteHealthEnabled,
    ensureSiteHealth,
  }));

  const wafPanelProps = computed(() => ({
    wafLoading: wafLoading.value,
    wafSaving: wafSaving.value,
    wafApplying: wafApplying.value,
    wafPreviewing: wafPreviewing.value,
    wafStatus: wafStatus.value,
    wafStatusText: wafStatusText.value,
    wafStatusColor: wafStatusColor.value,
    wafDraftPending: wafDraftPending.value,
    wafPreviewDiff: wafPreviewDiff.value,
    wafValidationError: wafValidationError.value,
    wafErrorMessage: wafErrorMessage.value,
    wafLoadState: wafLoadState.value,
    wafActionsDisabled: wafActionsDisabled.value,
    wafDirty: wafDirty.value,
    wafForm,
    wafAvailableSites: wafAvailableSites.value,
    wafSiteStates: wafSiteStates.value,
    wafModeOptions,
    wafStrengthOptions,
    wafAuditOptions,
    wafPreviewVisible: wafPreviewVisible.value,
    wafPreviewResult: wafPreviewResult.value,
    wafLastApply: wafLastApply.value,
  }));

  const savePreviewProps = computed(() => ({
    open: savePreview.open,
    saving: saving.value,
    kind: savePreview.kind,
    actions: savePreview.actions,
    errors: savePreview.errors,
    config: savePreview.config,
  }));

  const historyDrawerProps = computed(() => ({
    open: historyDrawerVisible.value,
    historyLoading: historyLoading.value,
    historyList: historyList.value,
    historyDescription,
    historyErrorMessage: historyErrorMessage.value,
    historyDetailVisible: historyDetailVisible.value,
    historyCompareVisible: historyCompareVisible.value,
    historyDetail: historyDetail.value,
    historyDiffOnly: historyDiffOnly.value,
    historyCompareRows: historyCompareRows.value,
    diffSideClass,
  }));

  return {
    configErrorMessage,
    loadingConfig,
    mode,
    modeOptions,
    quickValidationErrors,
    configContent,
    previewConfig,
    dockerDiscoveryOpen,
    dockerDiscoveryScanning,
    dockerDiscoveryResult,
    mergedQuickFormModel,
    previewing,
    upstreamPoolOptions,

    serverToolbarProps,
    blocksWorkbenchProps,
    wafPanelProps,
    savePreviewProps,
    historyDrawerProps,

    openHistory,
    applyPreset,
    openDockerDiscovery,
    parseRawToBlocks,
    previewBeforeSave,
    handleModeChange,
    setActiveQuickSiteId: (id: string) => {
      activeQuickSiteId.value = id;
    },
    setWafPreviewVisible: (v: boolean) => {
      wafPreviewVisible.value = v;
    },
    setSavePreviewOpen: (v: boolean) => {
      savePreview.open = v;
    },
    setHistoryDrawerVisible: (v: boolean) => {
      historyDrawerVisible.value = v;
    },
    setHistoryDetailVisible: (v: boolean) => {
      historyDetailVisible.value = v;
    },
    setHistoryCompareVisible: (v: boolean) => {
      historyCompareVisible.value = v;
    },
    setHistoryDiffOnly: (v: boolean) => {
      historyDiffOnly.value = v;
    },

    addUpstream,
    removeUpstream,
    handleRefreshSiteMetrics,
    addQuickSite,
    openAccessLogs,
    duplicateQuickSite,
    removeQuickSite,
    saveWafConfig,
    previewWafConfig,
    applyWafConfig,
    fetchWafConfig,
    confirmSave,
    openHistoryDetail,
    openHistoryCompare,
    rollbackHistory,
    handleDockerDiscoveryScan,
    handleDiscoveryCommitDrafts,
    handleDiscoveryPreview,
    handleDiscoveryApply,
  };
}

/** 丢弃 WAF 草稿前确认；返回是否继续。 */
function confirmBeforeLeave(title: string): Promise<boolean> {
  return new Promise((resolve) => {
    Modal.confirm({
      title,
      content: '防火墙设置还有未保存的修改，继续将丢失这些改动。',
      okText: '放弃修改',
      cancelText: '返回编辑',
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    });
  });
}
