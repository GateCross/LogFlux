<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted } from 'vue';
import { onBeforeRouteLeave, useRouter } from 'vue-router';
import { Modal } from 'antdv-next';

import { Alert, Button, Space } from 'antdv-next';

import { getCaddyConfigApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import { useSimpleWaf } from '../../config/composables/useSimpleWaf';
import WafPanel from '../../shared/components/WafPanel.vue';

defineOptions({ name: 'SettingsWafCard' });

const router = useRouter();

// ---------- 当前配置（为 WAF 预览 diff 与 baseHash 提供数据源） ----------
const {
  data: configData,
  loading: configLoading,
  errorMessage: configError,
  refetch: refetchConfig,
} = useListDetailQuery({
  queryKey: qk.caddy.config(),
  queryFn: () => getCaddyConfigApi(withListDetailErrorMode()),
  errorFallback: '获取 Caddy 配置失败',
});

/** 仅在配置读取中才算加载（禁用查询时 isPending 恒真） */
const configLoadingActive = computed(() => configLoading.value);

// ---------- WAF 状态层（与配置管理 waf 视角同一套 useSimpleWaf） ----------
const wafApi = useSimpleWaf({
  currentConfig: computed(() => configData.value?.config ?? ''),
  baseHash: computed(() => configData.value?.hash ?? ''),
  onApplied: async () => {
    await refetchConfig();
  },
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
  fetchWafConfig,
  saveWafConfig,
  previewWafConfig,
  applyWafConfig,
} = wafApi;

/** 组装 WafPanel 全量 props（加载态合并配置读取） */
const wafPanelProps = computed(() => ({
  wafLoading: wafLoading.value || configLoadingActive.value,
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

function setWafPreviewVisible(value: boolean) {
  wafPreviewVisible.value = value;
}

/** 刷新 = 配置与 WAF 状态一起重读，保证 baseHash 不落后 */
async function handleRefreshWaf() {
  await Promise.all([refetchConfig(), fetchWafConfig()]);
}

// 进页即读一次 WAF 状态：CRS/Coraza/Caddy 版本与已挂载规则数不必等「刷新」按钮。
// 配置查询由 useListDetailQuery 自行发起，这里只补 WAF 状态。
onMounted(() => {
  void fetchWafConfig();
});

// ---------- 草稿保护 ----------
function confirmDiscardWafDraft(title: string): Promise<boolean> {
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

/** 跳配置管理页：waf 视角带 mode，普通入口直接进 */
function goAdvancedConfig(withWafMode = false) {
  void router.push({
    name: 'CaddyConfig',
    query: {
      ...(withWafMode ? { mode: 'waf' } : {}),
    },
  });
}

// ---------- 离开页面草稿保护（与配置管理页一致） ----------
onBeforeRouteLeave(async () => {
  if (!wafDirty.value) return true;
  return confirmDiscardWafDraft('离开全局设置？');
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
</script>

<template>
  <div class="settings-waf-card">
    <div class="waf-toolbar">
      <div class="waf-toolbar-left">
        <Button type="link" size="small" @click="goAdvancedConfig(true)">
          误报排除等高级操作 →
        </Button>
      </div>
    </div>

    <Alert
      v-if="configError"
      class="mb-3"
      type="error"
      show-icon
      :message="configError"
    >
      <template #description>
        <Space>
          <span>配置读取失败时无法生成变更预览与并发保护。</span>
          <Button size="small" @click="handleRefreshWaf">重试</Button>
        </Space>
      </template>
    </Alert>

    <WafPanel
      v-bind="wafPanelProps"
      @update:waf-preview-visible="setWafPreviewVisible"
      @save="saveWafConfig"
      @preview="previewWafConfig"
      @apply="applyWafConfig"
      @refresh="handleRefreshWaf"
    />
  </div>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 12px;
}

.waf-toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-bottom: 12px;
}

.waf-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
