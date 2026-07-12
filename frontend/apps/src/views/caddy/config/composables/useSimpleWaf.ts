import { computed, reactive, ref, type Ref } from 'vue';
import { message } from 'antdv-next';
import { useQueryClient } from '@tanstack/vue-query';

import {
  applySimpleWafConfigApi,
  getSimpleWafConfigApi,
  previewSimpleWafConfigApi,
  updateSimpleWafConfigApi,
  type CaddySimpleWafApi,
} from '#/api/caddy/simple-waf';
import { withListDetailErrorMode } from '#/api/list-detail';
import {
  invalidateListDetailQueries,
  invalidateListDetailQueryKeys,
} from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { apiErrorMessage } from '#/utils/api-error-message';

type SimpleWafMode = CaddySimpleWafApi.SimpleWafMode;
type SimpleWafStrength = CaddySimpleWafApi.SimpleWafStrength;
type SimpleWafAudit = CaddySimpleWafApi.SimpleWafAudit;

/** MB → bytes（至少 1MB） */
export function mbToBytes(value: number) {
  return Math.max(1, Math.round(Number(value || 0))) * 1024 * 1024;
}

/** bytes → MB；非法值回退 fallback */
export function bytesToMB(value: number, fallback: number) {
  if (!value || value <= 0) return fallback;
  return Math.max(1, Math.round(value / 1024 / 1024));
}

export const wafModeOptions: Array<{ label: string; value: SimpleWafMode }> = [
  { label: '仅检测', value: 'detectiononly' },
  { label: '阻断', value: 'on' },
  { label: '关闭', value: 'off' },
];

export const wafStrengthOptions: Array<{ label: string; value: SimpleWafStrength }> = [
  { label: '低误报', value: 'low_fp' },
  { label: '平衡', value: 'balanced' },
  { label: '严格', value: 'high_blocking' },
];

export const wafAuditOptions: Array<{ label: string; value: SimpleWafAudit }> = [
  { label: '相关请求', value: 'relevantonly' },
  { label: '全量', value: 'on' },
  { label: '关闭', value: 'off' },
];

export interface UseSimpleWafOptions {
  selectedServerId: Ref<number | undefined>;
  /** 应用 WAF 成功后刷新配置（与既有行为一致） */
  onApplied?: () => Promise<void> | void;
}

export function useSimpleWaf(options: UseSimpleWafOptions) {
  const { selectedServerId, onApplied } = options;
  const queryClient = useQueryClient();

  const wafLoading = ref(false);
  /** 每次加载递增，避免旧服务器的慢响应覆盖当前 WAF 草稿。 */
  let wafRequestVersion = 0;
  const wafSaving = ref(false);
  const wafApplying = ref(false);
  const wafPreviewing = ref(false);
  const wafStatus = ref<CaddySimpleWafApi.SimpleWafConfig | null>(null);
  const wafPreviewVisible = ref(false);
  const wafPreviewResult = ref<CaddySimpleWafApi.SimpleWafConfig | null>(null);
  /** WAF 读失败页内展示（suppress 全局 toast 后不再 message.error） */
  const wafErrorMessage = ref<string | null>(null);
  const wafForm = reactive({
    audit: 'relevantonly' as SimpleWafAudit,
    enabled: false,
    mode: 'detectiononly' as SimpleWafMode,
    requestBodyAccess: true,
    requestBodyLimitMB: 10,
    requestBodyNoFilesLimitMB: 1,
    siteAddresses: [] as string[],
    strength: 'balanced' as SimpleWafStrength,
  });

  const wafAvailableSites = computed(() =>
    Array.isArray(wafStatus.value?.availableSites) ? wafStatus.value.availableSites : [],
  );

  const wafStatusText = computed(() => {
    if (!wafStatus.value) return '未加载';
    if (!wafStatus.value.enabled) return '关闭';
    if (wafStatus.value.mode === 'on') return '阻断';
    if (wafStatus.value.mode === 'detectiononly') return '仅检测';
    return '关闭';
  });

  const wafStatusColor = computed(() => {
    if (!wafStatus.value?.enabled) return 'default';
    return wafStatus.value.mode === 'on' ? 'green' : 'orange';
  });

  function syncWafForm(data: CaddySimpleWafApi.SimpleWafConfig) {
    wafStatus.value = data;
    wafForm.enabled = Boolean(data.enabled);
    wafForm.mode =
      data.mode === 'off'
        ? 'detectiononly'
        : ((data.mode || 'detectiononly') as SimpleWafMode);
    wafForm.strength = (data.strength || 'balanced') as SimpleWafStrength;
    wafForm.audit = (data.audit || 'relevantonly') as SimpleWafAudit;
    wafForm.requestBodyAccess = data.requestBodyAccess ?? true;
    wafForm.requestBodyLimitMB = bytesToMB(data.requestBodyLimit, 10);
    wafForm.requestBodyNoFilesLimitMB = bytesToMB(data.requestBodyNoFilesLimit, 1);
    wafForm.siteAddresses = data.siteAddresses?.length
      ? [...data.siteAddresses]
      : [...(data.availableSites ?? [])];
  }

  function buildWafPayload(): CaddySimpleWafApi.SimpleWafConfigPayload {
    const enabled = Boolean(wafForm.enabled);
    return {
      audit: wafForm.audit,
      enabled,
      mode: enabled ? wafForm.mode : 'off',
      requestBodyAccess: wafForm.requestBodyAccess,
      requestBodyLimit: mbToBytes(wafForm.requestBodyLimitMB),
      requestBodyNoFilesLimit: mbToBytes(wafForm.requestBodyNoFilesLimitMB),
      serverId: selectedServerId.value,
      siteAddresses: [...wafForm.siteAddresses],
      strength: wafForm.strength,
    };
  }

  /** config 前缀匹配：含 `qk.caddy.config(id)` 与 `[...config, 'simple-waf']` */
  async function invalidateWafRelatedQueries(opts?: { applied?: boolean }) {
    if (!selectedServerId.value) return;
    const serverId = selectedServerId.value;
    if (opts?.applied) {
      // apply 会改写运行中配置：连带 history / catalog / metrics
      await invalidateListDetailQueryKeys(queryClient, [
        qk.caddy.config(serverId),
        ['caddy', 'history', serverId],
        qk.caddy.catalog(serverId),
        ['caddy', 'metrics', serverId],
      ]);
      return;
    }
    await invalidateListDetailQueries(queryClient, qk.caddy.config(serverId));
  }

  async function fetchWafConfig() {
    const serverId = selectedServerId.value;
    if (!serverId) return;
    const requestVersion = ++wafRequestVersion;
    wafLoading.value = true;
    wafErrorMessage.value = null;
    try {
      // WAF 远程状态走 query cache；wafForm 仍是本地草稿
      const data = await queryClient.fetchQuery({
        queryKey: [...qk.caddy.config(serverId), 'simple-waf'],
        queryFn: () =>
          getSimpleWafConfigApi(serverId, withListDetailErrorMode()),
      });
      // 服务器已经切换或有更新的加载请求时，丢弃过期响应。
      if (
        requestVersion !== wafRequestVersion ||
        selectedServerId.value !== serverId
      ) {
        return;
      }
      if (data) {
        syncWafForm(data);
      }
    } catch (error) {
      if (
        requestVersion === wafRequestVersion &&
        selectedServerId.value === serverId
      ) {
        wafErrorMessage.value = apiErrorMessage(error, '获取防火墙配置失败');
      }
    } finally {
      if (requestVersion === wafRequestVersion) {
        wafLoading.value = false;
      }
    }
  }

  async function saveWafConfig() {
    if (!selectedServerId.value) return;
    wafSaving.value = true;
    try {
      await updateSimpleWafConfigApi(buildWafPayload());
      message.success('防火墙设置已保存');
      // 必须先 invalidate，否则 fetchQuery 可能命中 30s staleTime 旧缓存
      await invalidateWafRelatedQueries();
      await fetchWafConfig();
    } catch (error) {
      message.error(apiErrorMessage(error, '保存防火墙设置失败'));
    } finally {
      wafSaving.value = false;
    }
  }

  async function previewWafConfig() {
    if (!selectedServerId.value) return;
    wafPreviewing.value = true;
    try {
      wafPreviewResult.value = await previewSimpleWafConfigApi(buildWafPayload());
      wafPreviewVisible.value = true;
    } catch (error) {
      message.error(apiErrorMessage(error, '生成防火墙预览失败'));
    } finally {
      wafPreviewing.value = false;
    }
  }

  async function applyWafConfig() {
    if (!selectedServerId.value) return;
    wafApplying.value = true;
    try {
      const data = await applySimpleWafConfigApi(buildWafPayload());
      message.success(data?.message || '防火墙配置已应用');
      // 应用会改服务端配置/WAF 状态：失效 config + 关联 list，再 seed 本地草稿
      await invalidateWafRelatedQueries({ applied: true });
      if (data) {
        syncWafForm(data);
      } else {
        await fetchWafConfig();
      }
      await onApplied?.();
    } catch (error) {
      message.error(apiErrorMessage(error, '应用防火墙配置失败'));
    } finally {
      wafApplying.value = false;
    }
  }

  return {
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
    syncWafForm,
    buildWafPayload,
    fetchWafConfig,
    saveWafConfig,
    previewWafConfig,
    applyWafConfig,
  };
}
