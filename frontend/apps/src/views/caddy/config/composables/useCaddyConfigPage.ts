import { computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';

import type { CaddyPageMode } from '../types';
import { useCaddyServers } from './useCaddyServers';
import { useCaddyConfigIO } from './useCaddyConfigIO';
import { useSimpleWaf } from './useSimpleWaf';
import { useSiteMetrics } from './useSiteMetrics';
import { useConfigHistory } from './useConfigHistory';

/** 服务目录等入口深链：/caddy/config?serverId=&mode= */
function readQueryText(raw: unknown): string {
  return Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '');
}

export function useCaddyConfigPage() {
  const route = useRoute();

  function queryServerId(): number | undefined {
    const id = Number(readQueryText(route.query.serverId));
    return Number.isFinite(id) && id > 0 ? id : undefined;
  }

  function queryMode(): CaddyPageMode | undefined {
    const text = readQueryText(route.query.mode);
    if (text === 'blocks' || text === 'waf' || text === 'raw' || text === 'preview') {
      return text;
    }
    return undefined;
  }

  const serversApi = useCaddyServers({
    resolvePreferredServerId: queryServerId,
  });

  const {
    servers,
    selectedServerId,
    loadingServers,
    serversErrorMessage,
    serverModalVisible,
    serverModalType,
    serverForm,
    serverOptions,
    loadingServerStatus,
    serverStatusLoaded,
    selectedServerStatus,
    handleRefreshServerStatus,
    fetchServers,
    handleServerChange,
    openAddServerModal,
    openEditServerModal,
    saveServer,
    deleteCurrentServer,
  } = serversApi;

  /** metrics 与 configIO 互相回调：可变 holder 避免初始化顺序问题 */
  const metricsHooks = {
    reset: () => {},
    ready: () => {},
  };

  const configApi = useCaddyConfigIO({
    selectedServerId,
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
    mode,
    formModel,
    quickSiteDrafts,
    complexSiteSummaries,
    activeQuickSiteId,
    savePreview,
    siteWizardOpen,
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
    openSiteWizard,
    openDockerDiscovery,
    handleDockerDiscoveryScan,
    handleDiscoveryCommitDrafts,
    handleDiscoveryPreview,
    handleDiscoveryApply,
    handleWizardCommitDraft,
    handleWizardPreview,
    handleWizardApply,
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
    formatMetricCount,
    metricCountColor,
    resetSiteMetrics,
    debouncedFetchSiteMetrics,
    handleRefreshSiteMetrics,
    openAccessLogs,
    collectSiteHosts,
  } = metricsApi;

  metricsHooks.reset = resetSiteMetrics;
  metricsHooks.ready = () => {
    debouncedFetchSiteMetrics();
  };

  const wafApi = useSimpleWaf({
    selectedServerId,
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
    wafErrorMessage,
    wafForm,
    wafAvailableSites,
    wafStatusText,
    wafStatusColor,
    wafModeOptions,
    wafStrengthOptions,
    wafAuditOptions,
    fetchWafConfig,
    saveWafConfig,
    previewWafConfig,
    applyWafConfig,
  } = wafApi;

  const historyApi = useConfigHistory({
    selectedServerId,
    previewConfig,
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

  watch(selectedServerId, async () => {
    await fetchConfig();
    if (mode.value === 'waf') {
      await fetchWafConfig();
    }
  });

  watch(mode, (value) => {
    if (value === 'waf' && !wafStatus.value) {
      fetchWafConfig();
    }
  });

  /** 站点域名集合变化时 debounce 刷新近窗指标 */
  const siteHostsKey = computed(() => collectSiteHosts().sort().join('|'));
  watch(siteHostsKey, () => {
    debouncedFetchSiteMetrics();
  });

  onMounted(async () => {
    await fetchServers();
    if (selectedServerId.value) {
      await fetchConfig();
    }
    applyRouteModeDeepLink();
    if (mode.value === 'waf') {
      await fetchWafConfig();
    }
  });

  // 打包 props

  const serverToolbarProps = computed(() => ({
    selectedServerId: selectedServerId.value,
    loadingServers: loadingServers.value,
    serverOptions: serverOptions.value,
    selectedServerStatus: selectedServerStatus.value,
    serverStatusLoaded: serverStatusLoaded.value,
    loadingServerStatus: loadingServerStatus.value,
    serversCount: servers.value.length,
    mode: mode.value,
    previewing: previewing.value,
    configContent: configContent.value,
    serverModalVisible: serverModalVisible.value,
    serverModalType: serverModalType.value,
    serverForm,
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
    metricCountColor,
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
    wafStatusText: wafStatusText.value,
    wafStatusColor: wafStatusColor.value,
    wafErrorMessage: wafErrorMessage.value,
    wafForm,
    wafAvailableSites: wafAvailableSites.value,
    wafModeOptions,
    wafStrengthOptions,
    wafAuditOptions,
    wafPreviewVisible: wafPreviewVisible.value,
    wafPreviewResult: wafPreviewResult.value,
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
    servers,
    selectedServerId,
    serversErrorMessage,
    configErrorMessage,
    loadingConfig,
    mode,
    modeOptions,
    quickValidationErrors,
    configContent,
    previewConfig,
    siteWizardOpen,
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

    handleServerChange,
    openAddServerModal,
    openEditServerModal,
    deleteCurrentServer,
    saveServer,
    fetchServers,
    handleRefreshServerStatus,
    openHistory,
    applyPreset,
    openSiteWizard,
    openDockerDiscovery,
    parseRawToBlocks,
    previewBeforeSave,
    handleModeChange,
    setServerModalVisible: (v: boolean) => {
      serverModalVisible.value = v;
    },
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
    handleWizardCommitDraft,
    handleWizardPreview,
    handleWizardApply,
  };
}
