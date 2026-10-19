<script lang="ts" setup>
import type { ServerStatusItem } from '../../shared/caddy-server-status';
import type { ServerFormState } from '../composables/useCaddyServers';

import { computed } from 'vue';

import { Button, Dropdown, Select, Tag, Tooltip } from 'antdv-next';

import {
  formatLatency,
  statusLabel,
  statusTagColor,
  statusTooltip,
} from '../../shared/caddy-server-status';
import ServerFormModal from './ServerFormModal.vue';

const props = defineProps<{
  configContent: string;
  loadingServers: boolean;
  loadingServerStatus: boolean;
  mode: string;
  previewing: boolean;
  selectedServerId?: number;
  selectedServerStatus: null | ServerStatusItem | undefined;
  serverForm: ServerFormState;
  serverModalType: 'add' | 'edit';
  /** 服务器表单 modal 状态（由工具栏托管，不进入页面模板） */
  serverModalVisible: boolean;
  serverOptions: Array<{ label: string; value: number }>;
  serversCount: number;
  serverStatusLoaded: boolean;
}>();

const emit = defineEmits<{
  applyPreset: [];
  deleteServer: [];
  openAdd: [];
  openEdit: [];
  openHistory: [];
  parseRaw: [];
  refreshList: [];
  refreshStatus: [];
  saveBlocks: [];
  saveRaw: [];
  saveServer: [];
  'update:selectedServerId': [value: number];
  'update:serverModalVisible': [value: boolean];
}>();

/** 节点运维动作低频，收进下拉菜单，避免与配置主流程争抢视线 */
const nodeMenuItems = computed(() => [
  {
    key: 'probe',
    label: props.loadingServerStatus ? '探测中…' : '探测当前节点',
    disabled: props.serversCount === 0 || props.loadingServerStatus,
  },
  { type: 'divider' as const, key: 'd1' },
  { key: 'add', label: '添加节点' },
  { key: 'edit', label: '编辑节点', disabled: !props.selectedServerId },
  { key: 'delete', label: '删除节点', danger: true, disabled: !props.selectedServerId },
]);

function handleServerChange(value: unknown) {
  const next = Number(value);
  if (!Number.isFinite(next)) return;
  emit('update:selectedServerId', next);
}

function handleNodeMenuClick(info: { key: number | string }) {
  const actions: Record<string, () => void> = {
    probe: () => emit('refreshStatus'),
    add: () => emit('openAdd'),
    edit: () => emit('openEdit'),
    delete: () => emit('deleteServer'),
  };
  actions[String(info.key)]?.();
}

/** 保存按钮文案随编辑视角变化；防火墙有独立保存动作 */
const saveLabel = computed(() => (props.mode === 'raw' ? '保存原始配置' : '保存分块配置'));
</script>

<template>
  <div class="caddy-toolbar">
    <div class="toolbar-group primary">
      <Select
        :value="props.selectedServerId"
        :loading="props.loadingServers"
        :options="props.serverOptions"
        placeholder="选择服务器"
        class="server-select"
        @change="handleServerChange"
      />
      <Tooltip v-if="props.selectedServerStatus" :title="statusTooltip(props.selectedServerStatus)">
        <Tag :color="statusTagColor(props.selectedServerStatus)">
          {{ statusLabel(props.selectedServerStatus) }}
          <template v-if="props.selectedServerStatus.reachable">
            · {{ formatLatency(props.selectedServerStatus.latencyMs) }}
          </template>
        </Tag>
      </Tooltip>
      <Tag v-else-if="props.serverStatusLoaded" color="default">未探测</Tag>

      <Tooltip title="刷新节点列表">
        <Button :loading="props.loadingServers" @click="emit('refreshList')">刷新</Button>
      </Tooltip>

      <Dropdown
        :menu="{ items: nodeMenuItems, onClick: handleNodeMenuClick }"
        :trigger="['click']"
        placement="bottomLeft"
      >
        <Button :loading="props.loadingServerStatus">节点管理</Button>
      </Dropdown>
    </div>

    <div class="toolbar-group actions">
      <Button :disabled="!props.selectedServerId" @click="emit('openHistory')">历史</Button>
      <Button v-if="props.mode === 'blocks'" @click="emit('applyPreset')">默认模板</Button>
      <Button
        v-if="props.mode === 'raw'"
        :disabled="!props.configContent.trim()"
        @click="emit('parseRaw')"
      >
        解析为分块配置
      </Button>
      <!-- 防火墙有自己的保存动作，这里不显示 Caddyfile 保存按钮 -->
      <Button
        v-if="props.mode !== 'waf'"
        type="primary"
        :loading="props.previewing"
        :disabled="!props.selectedServerId"
        @click="props.mode === 'raw' ? emit('saveRaw') : emit('saveBlocks')"
      >
        {{ saveLabel }}
      </Button>
    </div>
  </div>

  <!-- 由工具栏打开；表单字段主体不在 页面 -->
  <ServerFormModal
    :open="props.serverModalVisible"
    :type="props.serverModalType"
    :form="props.serverForm"
    @update:open="emit('update:serverModalVisible', $event)"
    @save="emit('saveServer')"
  />
</template>

<style scoped>
.caddy-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px solid hsl(var(--border));
}

.toolbar-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.server-select {
  min-width: 220px;
}

@media (max-width: 960px) {
  .toolbar-group.actions {
    justify-content: flex-start;
  }
}
</style>
