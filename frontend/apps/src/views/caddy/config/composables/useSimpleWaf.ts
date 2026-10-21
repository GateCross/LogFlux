import { computed, reactive, ref, type Ref } from 'vue';
import { message, Modal } from 'antdv-next';
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

import { buildLineDiff, formatCaddyfile } from '../caddy-config-utils';

type SimpleWafMode = CaddySimpleWafApi.SimpleWafMode;
type SimpleWafStrength = CaddySimpleWafApi.SimpleWafStrength;
type SimpleWafAudit = CaddySimpleWafApi.SimpleWafAudit;

export type WafFormState = {
  audit: SimpleWafAudit;
  enabled: boolean;
  mode: SimpleWafMode;
  requestBodyAccess: boolean;
  requestBodyLimitMB: number;
  requestBodyNoFilesLimitMB: number;
  siteAddresses: string[];
  strength: SimpleWafStrength;
};

/** 读请求的显式状态：只有 ready 才允许保存/预览/应用。 */
export type WafLoadState = 'error' | 'idle' | 'loading' | 'ready';

export function createDefaultWafForm(): WafFormState {
  return {
    audit: 'relevantonly',
    enabled: false,
    mode: 'detectiononly',
    requestBodyAccess: true,
    requestBodyLimitMB: 10,
    requestBodyNoFilesLimitMB: 1,
    siteAddresses: [],
    strength: 'balanced',
  };
}

/** 与后端 validatePolicyRequestBodyLimit 对齐：1B ~ 1GB。 */
export const WAF_REQUEST_BODY_MAX_MB = 1024;

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
  /** 当前服务器 Caddyfile 原文；预览用它算 diff */
  currentConfig?: Ref<string>;
  /** 当前配置哈希；apply 时回传做乐观并发 */
  baseHash?: Ref<string>;
  /** 应用 WAF 成功后刷新配置（与既有行为一致） */
  onApplied?: () => Promise<void> | void;
}

export function useSimpleWaf(options: UseSimpleWafOptions) {
  const { selectedServerId, currentConfig, baseHash, onApplied } = options;
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
  /** 最近一次成功应用的结果，用于页内回显（toast 会消失，这里留痕）。 */
  const wafLastApply = ref<{ actions: string[]; message: string; at: string } | null>(null);
  /** WAF 读失败页内展示（suppress 全局 toast 后不再 message.error） */
  const wafErrorMessage = ref<null | string>(null);
  /** 显式加载态：读失败后禁止保存/预览/应用，避免拿默认值覆盖线上策略。 */
  const wafLoadState = ref<WafLoadState>('idle');
  /** 当前 wafStatus 归属的服务器；与 selectedServerId 不一致说明数据已过期。 */
  const wafLoadedServerId = ref<null | number>(null);
  const wafForm = reactive(createDefaultWafForm());

  /** 只有成功读到当前服务器状态才允许写操作。 */
  const wafActionsDisabled = computed(
    () => wafLoadState.value !== 'ready' || wafLoadedServerId.value !== selectedServerId.value,
  );

  const wafAvailableSites = computed(() =>
    Array.isArray(wafStatus.value?.availableSites) ? wafStatus.value.availableSites : [],
  );

  /** Caddyfile 里已挂载 waf_protect 的站点（后端快照，非本地草稿）。 */
  const wafImportedSites = computed(() =>
    Array.isArray(wafStatus.value?.importedSites) ? wafStatus.value.importedSites : [],
  );

  /** 当前生效的站点集合：表单为空表示"跟随全部可用站点"。 */
  const wafEffectiveSites = computed(() =>
    wafForm.siteAddresses.length > 0 ? wafForm.siteAddresses : wafAvailableSites.value,
  );

  /** 站点在"已生效 / 待应用 / 将移除"三态中的归属。 */
  const wafSiteStates = computed(() => {
    const imported = new Set(wafImportedSites.value);
    const effective = new Set(wafEffectiveSites.value);
    const enabled = wafForm.enabled;
    const all = [...new Set([...wafAvailableSites.value, ...imported])];

    return all.map((site) => {
      if (!enabled) {
        return { site, state: imported.has(site) ? ('remove' as const) : ('pending' as const) };
      }
      if (!effective.has(site)) return { site, state: 'pending' as const };
      return { site, state: imported.has(site) ? ('active' as const) : ('pending' as const) };
    });
  });

  /** 表单相对最近一次读库快照是否有未保存改动。 */
  const wafDirty = computed(() => {
    if (wafLoadState.value !== 'ready' || !wafStatus.value) return false;
    const baseline = formFromConfig(wafStatus.value);
    return (Object.keys(baseline) as Array<keyof WafFormState>).some(
      (key) => !isFormFieldEqual(key, baseline[key], wafForm[key]),
    );
  });

  /** 作废在途请求并回到未知态；切服务器时调用。 */
  function resetWafState() {
    wafRequestVersion += 1;
    wafStatus.value = null;
    wafPreviewResult.value = null;
    wafPreviewVisible.value = false;
    wafLastApply.value = null;
    wafErrorMessage.value = null;
    wafLoadState.value = 'idle';
    wafLoadedServerId.value = null;
    Object.assign(wafForm, createDefaultWafForm());
  }

  /** 当前 WAF 状态是否不属于选中服务器（含从未加载）。 */
  function isWafStale() {
    return wafLoadedServerId.value !== selectedServerId.value;
  }

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

  /** 策略已开启但 Caddyfile 里还没挂载：属于草稿态，需要「应用到 Caddy」才生效。 */
  const wafDraftPending = computed(
    () => Boolean(wafStatus.value?.enabled) && !wafStatus.value?.integrated,
  );

  /** 预览候选 Caddyfile 相对当前配置的逐行差异。 */
  const wafPreviewDiff = computed(() => {
    const next = wafPreviewResult.value?.config;
    if (!next) return [];
    return buildLineDiff(formatCaddyfile(currentConfig?.value ?? ''), formatCaddyfile(next));
  });

  /** 把后端配置归一成表单形状；读库回填与脏标记基线共用。 */
  function formFromConfig(data: CaddySimpleWafApi.SimpleWafConfig): WafFormState {
    return {
      audit: (data.audit || 'relevantonly') as SimpleWafAudit,
      enabled: Boolean(data.enabled),
      mode:
        data.mode === 'off'
          ? 'detectiononly'
          : ((data.mode || 'detectiononly') as SimpleWafMode),
      requestBodyAccess: data.requestBodyAccess ?? true,
      requestBodyLimitMB: bytesToMB(data.requestBodyLimit, 10),
      requestBodyNoFilesLimitMB: bytesToMB(data.requestBodyNoFilesLimit, 1),
      // 空选择表示"跟随全部站点"，不要固化成当前站点快照。
      siteAddresses: Array.isArray(data.siteAddresses) ? [...data.siteAddresses] : [],
      strength: (data.strength || 'balanced') as SimpleWafStrength,
    };
  }

  /** 数组按集合比较，其余字段直接比较。 */
  function isFormFieldEqual(key: keyof WafFormState, a: unknown, b: unknown) {
    if (key === 'siteAddresses') {
      const left = Array.isArray(a) ? a : [];
      const right = Array.isArray(b) ? b : [];
      if (left.length !== right.length) return false;
      const rightSet = new Set(right);
      return left.every((item) => rightSet.has(item));
    }
    return a === b;
  }

  function syncWafForm(data: CaddySimpleWafApi.SimpleWafConfig) {
    wafStatus.value = data;
    Object.assign(wafForm, formFromConfig(data));
  }

  function buildWafPayload(): CaddySimpleWafApi.SimpleWafConfigPayload {
    const enabled = Boolean(wafForm.enabled);
    return {
      audit: wafForm.audit,
      baseHash: baseHash?.value || undefined,
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
    wafLoadState.value = 'loading';
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
      syncWafForm(data);
      wafLoadedServerId.value = serverId;
      wafLoadState.value = 'ready';
    } catch (error) {
      if (
        requestVersion === wafRequestVersion &&
        selectedServerId.value === serverId
      ) {
        wafErrorMessage.value = apiErrorMessage(error, '获取防火墙配置失败');
        // 读失败必须回到未知态：保留旧表单会让用户拿着过期/默认值继续提交。
        wafStatus.value = null;
        wafLoadedServerId.value = null;
        Object.assign(wafForm, createDefaultWafForm());
        wafLoadState.value = 'error';
      }
    } finally {
      if (requestVersion === wafRequestVersion) {
        wafLoading.value = false;
      }
    }
  }

  /** 提交前本地校验：后端也会拒绝，但没必要等一次往返。 */
  function validateWafForm(): string | null {
    const limit = Number(wafForm.requestBodyLimitMB || 0);
    const noFiles = Number(wafForm.requestBodyNoFilesLimitMB || 0);
    if (limit < 1 || limit > WAF_REQUEST_BODY_MAX_MB) {
      return `请求体上限需在 1~${WAF_REQUEST_BODY_MAX_MB} MB 之间`;
    }
    if (noFiles < 1 || noFiles > WAF_REQUEST_BODY_MAX_MB) {
      return `无文件请求体上限需在 1~${WAF_REQUEST_BODY_MAX_MB} MB 之间`;
    }
    if (noFiles > limit) {
      return '无文件请求体上限不能大于请求体总上限';
    }
    return null;
  }

  /** 表单校验结果；面板据此就地提示，提交路径复用同一判据。 */
  const wafValidationError = computed(() => validateWafForm());

  async function saveWafConfig() {
    if (!selectedServerId.value || wafActionsDisabled.value) return;
    const invalid = validateWafForm();
    if (invalid) {
      message.error(invalid);
      return;
    }
    wafSaving.value = true;
    try {
      await updateSimpleWafConfigApi(buildWafPayload());
      message.success('防火墙草稿已保存，需「应用到 Caddy」才会生效');
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
    if (!selectedServerId.value || wafActionsDisabled.value) return;
    const invalid = validateWafForm();
    if (invalid) {
      message.error(invalid);
      return;
    }
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

  /** 应用会立即改变线上拦截行为，先汇总影响面再确认。 */
  function describeApplyImpact() {
    const enabled = Boolean(wafForm.enabled);
    const sites = wafEffectiveSites.value;
    const modeLabel =
      wafModeOptions.find((item) => item.value === (enabled ? wafForm.mode : 'off'))?.label ??
      '关闭';
    const strengthLabel =
      wafStrengthOptions.find((item) => item.value === wafForm.strength)?.label ?? '-';
    const siteText = sites.length > 0 ? `${sites.length} 个：${sites.join('、')}` : '无';
    const lines = [
      '该操作会立即改写运行中的 Caddyfile 并触发重载。',
      `引擎模式：${modeLabel}`,
      `规则强度：${strengthLabel}`,
      `目标站点：${siteText}`,
    ];
    if (!enabled) {
      lines.push('注意：将取消这些站点的 WAF 挂载，防护立即失效。');
    }
    return lines.join('\n');
  }

  function applyWafConfig() {
    if (!selectedServerId.value || wafActionsDisabled.value) return;
    Modal.confirm({
      title: '确认应用到 Caddy',
      content: describeApplyImpact(),
      okText: '确认应用',
      cancelText: '取消',
      async onOk() {
        wafApplying.value = true;
        try {
          const data = await applySimpleWafConfigApi(buildWafPayload());
          message.success(data?.message || '防火墙配置已应用');
          wafLastApply.value = {
            actions: data?.actions ?? [],
            at: new Date().toLocaleString(),
            message: data?.message || '防火墙配置已应用',
          };
          // 应用会改服务端配置/WAF 状态：失效 config + 关联 list，再 seed 本地草稿
          await invalidateWafRelatedQueries({ applied: true });
          if (data) {
            syncWafForm(data);
            wafLoadedServerId.value = selectedServerId.value ?? null;
            wafLoadState.value = 'ready';
          } else {
            await fetchWafConfig();
          }
          await onApplied?.();
        } catch (error) {
          message.error(apiErrorMessage(error, '应用防火墙配置失败'));
        } finally {
          wafApplying.value = false;
        }
      },
    });
  }

  return {
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
    wafLoadedServerId,
    isWafStale,
    resetWafState,
    validateWafForm,
    syncWafForm,
    buildWafPayload,
    fetchWafConfig,
    saveWafConfig,
    previewWafConfig,
    applyWafConfig,
  };
}
