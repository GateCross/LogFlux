<script lang="ts" setup>
import { computed, onBeforeUnmount, watch } from 'vue';
import { onBeforeRouteLeave, useRouter } from 'vue-router';
import { Modal } from 'antdv-next';

import { Alert, Button, Select, Space } from 'antdv-next';

import {
  getCaddyConfigApi,
  getCaddyServerListApi,
} from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

import { serverLabel } from '../../config/composables/useCaddyServers';
import { useSimpleWaf } from '../../config/composables/useSimpleWaf';
import WafPanel from '../../shared/components/WafPanel.vue';

defineOptions({ name: 'SettingsWafCard' });

const props = defineProps<{
  /** 由父页持有（深链与「高级配置」跳转共用同一节点） */
  serverId?: number;
}>();

const emit = defineEmits<{
  'update:serverId': [value: number | undefined];
}>();

const router = useRouter();

/** 父页持有节点 id，这里只做双向透传 */
const selectedServerId = computed({
  get: () => props.serverId,
  set: (value) => emit('update:serverId', value),
});

// ---------- 节点列表（与配置管理/服务目录共享 qk.caddy.servers 缓存） ----------
const {
  data: serversData,
  loading: serversLoading,
  errorMessage: serversError,
  refetch: refetchServers,
} = useListDetailQuery({
  queryKey: qk.caddy.servers(),
  queryFn: () => getCaddyServerListApi(withListDetailErrorMode()),
  errorFallback: '获取服务器列表失败',
});

const servers = computed(() => serversData.value ?? []);

const serverOptions = computed(() =>
  servers.value.map((server) => ({
    label: serverLabel(server),
    value: Number(server.id),
  })),
);

/** 列表到达后解析目标节点：深链 serverId 优先，否则取第一个 */
watch(
  servers,
  (list) => {
    const current = selectedServerId.value;
    if (current && list.some((server) => Number(server.id) === current)) return;
    if (list.length > 0) {
      selectedServerId.value = Number(list[0]?.id);
    }
  },
  { immediate: true },
);

// ---------- 当前节点配置（为 WAF 预览 diff 与 baseHash 提供数据源） ----------
const {
  data: configData,
  loading: configLoading,
  errorMessage: configError,
  refetch: refetchConfig,
} = useListDetailQuery({
  queryKey: computed(() =>
    selectedServerId.value
      ? qk.caddy.config(selectedServerId.value)
      : (['caddy', 'config', 'none'] as const),
  ),
  queryFn: () => getCaddyConfigApi(selectedServerId.value!, withListDetailErrorMode()),
  errorFallback: '获取 Caddy 配置失败',
  enabled: () => Boolean(selectedServerId.value),
});

/** 仅在选中节点时才算加载中（禁用查询时 isPending 恒真） */
const configLoadingActive = computed(
  () => Boolean(selectedServerId.value) && configLoading.value,
);

// ---------- WAF 状态层（与配置管理 waf 视角同一套 useSimpleWaf） ----------
const wafApi = useSimpleWaf({
  selectedServerId,
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
  resetWafState,
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

// ---------- 节点切换与草稿保护（沿用配置管理页同款确认） ----------
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

/** 回退节点选择会再次触发 watcher，用标记跳过这一轮 */
let revertingServerSwitch = false;

watch(selectedServerId, async (next, prev) => {
  if (revertingServerSwitch) {
    revertingServerSwitch = false;
    return;
  }
  if (next === prev) return;
  const shouldConfirm = prev !== undefined && next !== undefined && wafDirty.value;
  if (shouldConfirm && !(await confirmDiscardWafDraft('切换节点？'))) {
    revertingServerSwitch = true;
    selectedServerId.value = prev;
    return;
  }
  // WAF 状态属于具体节点：切换前先作废，避免旧数据冒充新节点状态
  resetWafState();
  if (next) {
    await fetchWafConfig();
  }
});

function handleServerChange(value: unknown) {
  const next = Number(value);
  if (!Number.isFinite(next)) return;
  selectedServerId.value = next;
}

/** 跳配置管理页：waf 视角带 mode，普通入口直接进 */
function goAdvancedConfig(withWafMode = false) {
  void router.push({
    name: 'CaddyConfig',
    query: {
      ...(selectedServerId.value
        ? { serverId: String(selectedServerId.value) }
        : {}),
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
    <Alert
      v-if="serversError"
      class="mb-3"
      type="error"
      show-icon
      :message="serversError"
    >
      <template #description>
        <Space>
          <span>节点列表读取失败，WAF 写操作已禁用。</span>
          <Button size="small" @click="refetchServers">重试</Button>
        </Space>
      </template>
    </Alert>

    <Alert
      v-else-if="!serversLoading && servers.length === 0"
      class="mb-3"
      type="info"
      show-icon
      message="尚无 Caddy 节点"
      description="WAF 防护按节点管理，请先在服务目录或配置管理添加节点。"
    />

    <template v-else>
      <!-- 节点选择 + 高级配置入口：WafPanel 自带「WAF 防护配置」卡片 -->
      <div class="waf-toolbar">
        <div class="waf-toolbar-left">
          <!-- 单节点时选择器是空操作，连标签一起隐藏 -->
          <template v-if="serverOptions.length > 1">
            <span class="waf-toolbar-label">节点</span>
            <Select
              :value="selectedServerId"
              :options="serverOptions"
              :loading="serversLoading"
              style="width: 220px"
              @change="handleServerChange"
            />
          </template>
        </div>
        <Button
          type="link"
          size="small"
          :disabled="!selectedServerId"
          @click="goAdvancedConfig(true)"
        >
          误报排除等高级操作 →
        </Button>
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
    </template>
  </div>
</template>

<style scoped>
.mb-3 {
  margin-bottom: 12px;
}

.waf-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.waf-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.waf-toolbar-label {
  color: #8c8c8c;
  font-size: 13px;
}
</style>
