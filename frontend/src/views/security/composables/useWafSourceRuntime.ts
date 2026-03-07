import { ref } from 'vue';
import { type WafEngineStatusResp, checkWafEngine, fetchWafEngineStatus } from '@/service/api/caddy-source';
import {
  type WafIntegrationStatusResp,
  applyWafIntegration,
  fetchWafIntegrationStatus
} from '@/service/api/caddy-integration';

type MessageApi = {
  success: (content: string) => void;
  warning: (content: string) => void;
};

interface UseWafSourceRuntimeOptions {
  message: MessageApi;
  onEngineChecked?: () => void;
}

export function useWafSourceRuntime(options: UseWafSourceRuntimeOptions) {
  const { message, onEngineChecked } = options;

  const engineLoading = ref(false);
  const engineChecking = ref(false);
  const engineUnavailable = ref(false);
  const engineStatus = ref<WafEngineStatusResp | null>(null);

  const integrationLoading = ref(false);
  const integrationSubmitting = ref(false);
  const integrationPreviewing = ref(false);
  const integrationUnavailable = ref(false);
  const integrationStatus = ref<WafIntegrationStatusResp | null>(null);
  const selectedIntegrationSites = ref<string[]>([]);
  const integrationPreviewActions = ref<string[]>([]);

  function displayEngineValue(value: unknown) {
    if (value === undefined || value === null || value === '') {
      return '-';
    }
    return String(value);
  }

  function syncSelectedIntegrationSites(status: WafIntegrationStatusResp | null) {
    const available = Array.isArray(status?.availableSites) ? status.availableSites : [];
    const imported = Array.isArray(status?.importedSites) ? status.importedSites : [];

    if (imported.length > 0) {
      selectedIntegrationSites.value = imported.filter(item => available.includes(item));
      return;
    }

    const preserved = selectedIntegrationSites.value.filter(item => available.includes(item));
    if (preserved.length > 0) {
      selectedIntegrationSites.value = preserved;
      return;
    }

    selectedIntegrationSites.value = [...available];
  }

  async function fetchIntegrationStatus() {
    if (integrationUnavailable.value) {
      return;
    }

    integrationLoading.value = true;
    try {
      const { data, error } = await fetchWafIntegrationStatus();
      if (!error && data) {
        integrationStatus.value = data;
        integrationUnavailable.value = false;
        syncSelectedIntegrationSites(data);
        return;
      }

      if (error) {
        const status = Number((error as any)?.response?.status || 0);
        if (status === 404 || status === 405) {
          integrationUnavailable.value = true;
        }
      }
    } finally {
      integrationLoading.value = false;
    }
  }

  function handleRefreshIntegrationStatus() {
    fetchIntegrationStatus();
  }

  function handleIntegrationSiteChange(value: Array<string | number>) {
    selectedIntegrationSites.value = value.map(item => String(item));
  }

  async function submitIntegration(enabled: boolean, dryRun: boolean) {
    if (integrationUnavailable.value) {
      message.warning('后端接入开关接口尚未开放，当前仅展示占位状态');
      return;
    }

    const siteAddresses = selectedIntegrationSites.value.filter(item => item.trim());
    if (siteAddresses.length === 0) {
      message.warning('请至少选择一个站点');
      return;
    }

    if (dryRun) {
      integrationPreviewing.value = true;
    } else {
      integrationSubmitting.value = true;
    }

    try {
      const { data, error } = await applyWafIntegration({
        enabled,
        siteAddresses,
        dryRun
      });

      if (error || !data) {
        const status = Number((error as any)?.response?.status || 0);
        if (status === 404 || status === 405) {
          integrationUnavailable.value = true;
          message.warning('后端接入开关接口尚未开放，已切换占位模式');
        }
        return;
      }

      integrationPreviewActions.value = data.actions || [];
      if (dryRun) {
        message.success(data.message || '已生成接入变更预览');
        return;
      }

      message.success(data.message || (enabled ? 'WAF 接入已启用' : 'WAF 接入已取消'));
      await fetchIntegrationStatus();
    } finally {
      integrationPreviewing.value = false;
      integrationSubmitting.value = false;
    }
  }

  function handlePreviewIntegration() {
    return submitIntegration(true, true);
  }

  function handleEnableIntegration() {
    return submitIntegration(true, false);
  }

  function handleDisableIntegration() {
    return submitIntegration(false, false);
  }

  async function fetchEngineStatus() {
    if (engineUnavailable.value) {
      return;
    }

    engineLoading.value = true;
    try {
      const { data, error } = await fetchWafEngineStatus();
      if (!error && data) {
        engineStatus.value = data;
        engineUnavailable.value = false;
        return;
      }

      if (error) {
        const status = Number((error as any)?.response?.status || 0);
        if (status === 404 || status === 405) {
          engineUnavailable.value = true;
        }
      }
    } finally {
      engineLoading.value = false;
    }
  }

  function handleRefreshEngineStatus() {
    fetchEngineStatus();
  }

  async function handleCheckEngine() {
    if (engineUnavailable.value) {
      message.warning('后端接口尚未开放，当前仅展示占位状态');
      return;
    }

    engineChecking.value = true;
    try {
      const { error } = await checkWafEngine();
      if (!error) {
        message.success('引擎检查任务已提交');
        fetchEngineStatus();
        onEngineChecked?.();
        return;
      }

      const status = Number((error as any)?.response?.status || 0);
      if (status === 404 || status === 405) {
        engineUnavailable.value = true;
        message.warning('后端接口尚未开放，已切换占位模式');
      }
    } finally {
      engineChecking.value = false;
    }
  }

  return {
    engineLoading,
    engineChecking,
    engineUnavailable,
    engineStatus,
    integrationLoading,
    integrationSubmitting,
    integrationPreviewing,
    integrationUnavailable,
    integrationStatus,
    selectedIntegrationSites,
    integrationPreviewActions,
    displayEngineValue,
    fetchIntegrationStatus,
    handleRefreshIntegrationStatus,
    handleIntegrationSiteChange,
    handlePreviewIntegration,
    handleEnableIntegration,
    handleDisableIntegration,
    fetchEngineStatus,
    handleRefreshEngineStatus,
    handleCheckEngine
  };
}
