import { computed, reactive, ref, watch, type Ref } from 'vue';
import { useMessage } from 'naive-ui';
import {
  type WafIntegrationStatusResp,
  applyWafIntegration,
  fetchWafIntegrationStatus
} from '@/service/api/caddy-integration';
import {
  type SimpleWafAudit,
  type SimpleWafConfigPayload,
  type SimpleWafConfigResp,
  type SimpleWafMode,
  type SimpleWafStrength,
  applySimpleWafConfig,
  fetchSimpleWafConfig,
  previewSimpleWafConfig,
  updateSimpleWafConfig
} from '@/service/api/caddy-simple-waf';

export function useSimpleWafBlock(opts: {
  currentServerId: Ref<number | null>;
  loadConfig: () => Promise<void>;
}) {
  const message = useMessage();

  // ========== Simple WAF 配置 ==========

  const simpleWafLoading = ref(false);
  const simpleWafSubmitting = ref(false);
  const simpleWafPreviewing = ref(false);
  const simpleWafSaving = ref(false);
  const simpleWafStatus = ref<SimpleWafConfigResp | null>(null);
  const simpleWafPreviewResult = ref<SimpleWafConfigResp | null>(null);
  const showSimpleWafPreview = ref(false);

  const simpleWafForm = reactive({
    enabled: false,
    mode: 'detectiononly' as SimpleWafMode,
    strength: 'low_fp' as SimpleWafStrength,
    audit: 'relevantonly' as SimpleWafAudit,
    requestBodyAccess: true,
    requestBodyLimitMB: 10,
    requestBodyNoFilesLimitMB: 1,
    siteAddresses: [] as string[]
  });

  const simpleWafModeOptions = [
    { label: '仅检测', value: 'detectiononly' },
    { label: '阻断', value: 'on' },
    { label: '关闭', value: 'off' }
  ];

  const simpleWafStrengthOptions = [
    { label: '低误报', value: 'low_fp' },
    { label: '平衡', value: 'balanced' },
    { label: '严格', value: 'high_blocking' }
  ];

  const simpleWafAuditOptions = [
    { label: '相关请求', value: 'relevantonly' },
    { label: '全量', value: 'on' },
    { label: '关闭', value: 'off' }
  ];

  const simpleWafStatusType = computed(() => {
    if (!simpleWafStatus.value) return 'default';
    if (simpleWafStatus.value.enabled && simpleWafStatus.value.mode === 'on') return 'success';
    if (simpleWafStatus.value.enabled && simpleWafStatus.value.mode === 'detectiononly') return 'warning';
    return 'default';
  });

  const simpleWafStatusText = computed(() => {
    if (!simpleWafStatus.value) return '未加载';
    if (!simpleWafStatus.value.enabled) return '关闭';
    if (simpleWafStatus.value.mode === 'on') return '阻断';
    if (simpleWafStatus.value.mode === 'detectiononly') return '仅检测';
    return '关闭';
  });

  const simpleWafSiteOptions = computed(() =>
    (simpleWafStatus.value?.availableSites || []).map(item => ({ label: item, value: item }))
  );

  function mbToBytes(value: number) {
    return Math.max(1, Math.round(Number(value || 0))) * 1024 * 1024;
  }

  function bytesToMB(value: number, fallback: number) {
    if (!value || value <= 0) return fallback;
    return Math.max(1, Math.round(value / 1024 / 1024));
  }

  function formatVersion(value?: string) {
    const trimmed = String(value || '').trim();
    return trimmed || '未检测到';
  }

  function syncSimpleWafForm(data: SimpleWafConfigResp) {
    simpleWafStatus.value = data;
    simpleWafForm.enabled = Boolean(data.enabled);
    simpleWafForm.mode = data.mode === 'off' ? 'detectiononly' : data.mode || 'detectiononly';
    simpleWafForm.strength = data.strength || 'low_fp';
    simpleWafForm.audit = data.audit || 'relevantonly';
    simpleWafForm.requestBodyAccess = data.requestBodyAccess;
    simpleWafForm.requestBodyLimitMB = bytesToMB(data.requestBodyLimit, 10);
    simpleWafForm.requestBodyNoFilesLimitMB = bytesToMB(data.requestBodyNoFilesLimit, 1);
    simpleWafForm.siteAddresses = data.siteAddresses?.length ? [...data.siteAddresses] : [...(data.availableSites || [])];
  }

  function buildSimpleWafPayload(): SimpleWafConfigPayload {
    const enabled = Boolean(simpleWafForm.enabled);
    return {
      serverId: opts.currentServerId.value || undefined,
      enabled,
      mode: enabled ? simpleWafForm.mode : 'off',
      strength: simpleWafForm.strength,
      audit: simpleWafForm.audit,
      requestBodyAccess: simpleWafForm.requestBodyAccess,
      requestBodyLimit: mbToBytes(simpleWafForm.requestBodyLimitMB),
      requestBodyNoFilesLimit: mbToBytes(simpleWafForm.requestBodyNoFilesLimitMB),
      siteAddresses: [...simpleWafForm.siteAddresses]
    };
  }

  async function fetchSimpleWafData() {
    if (!opts.currentServerId.value) return;
    simpleWafLoading.value = true;
    try {
      const { data, error } = await fetchSimpleWafConfig(opts.currentServerId.value);
      if (error || !data) {
        message.error('获取防火墙配置失败');
        return;
      }
      syncSimpleWafForm(data);
    } finally {
      simpleWafLoading.value = false;
    }
  }

  async function saveSimpleWafConfig() {
    if (!opts.currentServerId.value) return;
    simpleWafSaving.value = true;
    try {
      const { error } = await updateSimpleWafConfig(buildSimpleWafPayload());
      if (error) {
        message.error('保存防火墙设置失败');
        return;
      }
      message.success('防火墙设置已保存');
      await fetchSimpleWafData();
    } finally {
      simpleWafSaving.value = false;
    }
  }

  async function previewSimpleWaf() {
    if (!opts.currentServerId.value) return;
    simpleWafPreviewing.value = true;
    try {
      const { data, error } = await previewSimpleWafConfig(buildSimpleWafPayload());
      if (error || !data) {
        message.error('生成防火墙预览失败');
        return;
      }
      simpleWafPreviewResult.value = data;
      showSimpleWafPreview.value = true;
    } finally {
      simpleWafPreviewing.value = false;
    }
  }

  async function applySimpleWaf() {
    if (!opts.currentServerId.value) return;
    simpleWafSubmitting.value = true;
    try {
      const { data, error } = await applySimpleWafConfig(buildSimpleWafPayload());
      if (error || !data) {
        message.error('应用防火墙配置失败');
        return;
      }
      message.success(data.message || '防火墙配置已应用');
      syncSimpleWafForm(data);
      await opts.loadConfig();
      await fetchWafIntegrationState();
    } finally {
      simpleWafSubmitting.value = false;
    }
  }

  function handleSimpleWafSiteChange(value: Array<string | number>) {
    simpleWafForm.siteAddresses = value.map(item => String(item));
  }

  // enable 时自动切到 detectiononly
  watch(
    () => simpleWafForm.enabled,
    enabled => {
      if (enabled && simpleWafForm.mode === 'off') {
        simpleWafForm.mode = 'detectiononly';
      }
    }
  );

  // ========== WAF Integration（Coraza 接入开关）==========

  const wafIntegrationLoading = ref(false);
  const wafIntegrationSubmitting = ref(false);
  const wafIntegrationPreviewing = ref(false);
  const wafIntegrationUnavailable = ref(false);
  const wafIntegrationStatus = ref<WafIntegrationStatusResp | null>(null);
  const selectedWafIntegrationSites = ref<string[]>([]);
  const wafIntegrationPreviewActions = ref<string[]>([]);

  function syncSelectedWafIntegrationSites(status: WafIntegrationStatusResp | null) {
    const available = Array.isArray(status?.availableSites) ? status.availableSites : [];
    const imported = Array.isArray(status?.importedSites) ? status.importedSites : [];

    if (imported.length > 0) {
      selectedWafIntegrationSites.value = imported.filter(item => available.includes(item));
      return;
    }

    const preserved = selectedWafIntegrationSites.value.filter(item => available.includes(item));
    if (preserved.length > 0) {
      selectedWafIntegrationSites.value = preserved;
      return;
    }

    selectedWafIntegrationSites.value = [...available];
  }

  async function fetchWafIntegrationState() {
    if (!opts.currentServerId.value || wafIntegrationUnavailable.value) return;

    wafIntegrationLoading.value = true;
    try {
      const { data, error } = await fetchWafIntegrationStatus();
      if (!error && data) {
        if (data.serverId && data.serverId !== opts.currentServerId.value) {
          wafIntegrationStatus.value = null;
          selectedWafIntegrationSites.value = [];
          return;
        }
        wafIntegrationStatus.value = data;
        wafIntegrationUnavailable.value = false;
        syncSelectedWafIntegrationSites(data);
        return;
      }

      if (error) {
        const status = Number((error as any)?.response?.status || 0);
        if (status === 404 || status === 405) {
          wafIntegrationUnavailable.value = true;
        }
      }
    } finally {
      wafIntegrationLoading.value = false;
    }
  }

  function handleRefreshWafIntegrationState() {
    fetchWafIntegrationState();
  }

  function handleWafIntegrationSiteChange(value: Array<string | number>) {
    selectedWafIntegrationSites.value = value.map(item => String(item));
  }

  async function submitWafIntegration(enabled: boolean, dryRun: boolean) {
    if (!opts.currentServerId.value) {
      message.warning('请先选择 Caddy 服务器');
      return;
    }
    if (wafIntegrationUnavailable.value) {
      message.warning('当前接入开关接口暂不可用');
      return;
    }

    const siteAddresses = selectedWafIntegrationSites.value.filter(item => item.trim());
    if (siteAddresses.length === 0) {
      message.warning('请至少选择一个站点');
      return;
    }

    if (dryRun) {
      wafIntegrationPreviewing.value = true;
    } else {
      wafIntegrationSubmitting.value = true;
    }

    try {
      const { data, error } = await applyWafIntegration({
        serverId: opts.currentServerId.value,
        enabled,
        siteAddresses,
        dryRun
      });
      if (error || !data) {
        const status = Number((error as any)?.response?.status || 0);
        if (status === 404 || status === 405) {
          wafIntegrationUnavailable.value = true;
          message.warning('当前接入开关接口暂不可用');
        }
        return;
      }

      wafIntegrationPreviewActions.value = data.actions || [];
      if (dryRun) {
        message.success(data.message || '已生成 WAF 接入预览');
        return;
      }

      message.success(data.message || (enabled ? 'WAF 接入已应用' : 'WAF 接入已取消'));
      await opts.loadConfig();
      await fetchWafIntegrationState();
    } finally {
      wafIntegrationPreviewing.value = false;
      wafIntegrationSubmitting.value = false;
    }
  }

  function handlePreviewWafIntegration() {
    return submitWafIntegration(true, true);
  }

  function handleEnableWafIntegration() {
    return submitWafIntegration(true, false);
  }

  function handleDisableWafIntegration() {
    return submitWafIntegration(false, false);
  }

  // ========== 通用 ==========

  async function handleWafApplied() {
    await opts.loadConfig();
    await fetchWafIntegrationState();
  }

  // 切换服务器时重置并重新加载
  watch(
    () => opts.currentServerId.value,
    () => {
      if (opts.currentServerId.value) {
        wafIntegrationUnavailable.value = false;
        fetchSimpleWafData();
        fetchWafIntegrationState();
      }
    }
  );

  return {
    // Simple WAF
    simpleWafLoading,
    simpleWafSubmitting,
    simpleWafPreviewing,
    simpleWafSaving,
    simpleWafStatus,
    simpleWafPreviewResult,
    showSimpleWafPreview,
    simpleWafForm,
    simpleWafModeOptions,
    simpleWafStrengthOptions,
    simpleWafAuditOptions,
    simpleWafStatusType,
    simpleWafStatusText,
    simpleWafSiteOptions,
    formatVersion,
    fetchSimpleWafData,
    saveSimpleWafConfig,
    previewSimpleWaf,
    applySimpleWaf,
    handleSimpleWafSiteChange,
    // WAF Integration
    wafIntegrationLoading,
    wafIntegrationSubmitting,
    wafIntegrationPreviewing,
    wafIntegrationUnavailable,
    wafIntegrationStatus,
    selectedWafIntegrationSites,
    wafIntegrationPreviewActions,
    fetchWafIntegrationState,
    handleRefreshWafIntegrationState,
    handleWafIntegrationSiteChange,
    handlePreviewWafIntegration,
    handleEnableWafIntegration,
    handleDisableWafIntegration,
    handleWafApplied
  };
}
