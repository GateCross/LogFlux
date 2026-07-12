<script lang="ts" setup>
import { useRouter } from 'vue-router';
import { Button, Select, Tag, Tooltip } from 'antdv-next';

import type { ServerStatusItem } from '../../shared/caddy-server-status';
import {
  formatLatency,
  statusLabel,
  statusTagColor,
  statusTooltip,
} from '../../shared/caddy-server-status';
import type { ServerFormState } from '../composables/useCaddyServers';
import ServerFormModal from './ServerFormModal.vue';

const props = defineProps<{
  selectedServerId?: number;
  loadingServers: boolean;
  serverOptions: Array<{ label: string; value: number }>;
  selectedServerStatus: ServerStatusItem | null | undefined;
  serverStatusLoaded: boolean;
  loadingServerStatus: boolean;
  serversCount: number;
  mode: string;
  previewing: boolean;
  configContent: string;
  /** 服务器表单 modal 状态（由工具栏托管，不进入页面模板） */
  serverModalVisible: boolean;
  serverModalType: 'add' | 'edit';
  serverForm: ServerFormState;
}>();

const emit = defineEmits<{
  'update:selectedServerId': [value: number];
  'update:serverModalVisible': [value: boolean];
  refreshList: [];
  refreshStatus: [];
  openAdd: [];
  openEdit: [];
  deleteServer: [];
  openHistory: [];
  applyPreset: [];
  openWizard: [];
  openDocker: [];
  parseRaw: [];
  saveRaw: [];
  saveBlocks: [];
  saveServer: [];
}>();

const router = useRouter();

function handleServerChange(value: unknown) {
  const next = Number(value);
  if (!Number.isFinite(next)) return;
  emit('update:selectedServerId', next);
}
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
      <Button :loading="props.loadingServers" @click="emit('refreshList')">刷新列表</Button>
      <Button
        :loading="props.loadingServerStatus"
        :disabled="props.serversCount === 0"
        @click="emit('refreshStatus')"
      >
        探测当前节点
      </Button>
      <Button type="link" @click="router.push({ name: 'CaddyCatalog' })">
        服务目录总览
      </Button>
      <Button type="primary" ghost @click="emit('openAdd')">添加</Button>
      <Button :disabled="!props.selectedServerId" @click="emit('openEdit')">编辑</Button>
      <Button danger :disabled="!props.selectedServerId" @click="emit('deleteServer')">删除</Button>
    </div>
    <div class="toolbar-group actions">
      <Button :disabled="!props.selectedServerId" @click="emit('openHistory')">历史</Button>
      <Button @click="emit('applyPreset')">默认模板</Button>
      <Button type="primary" ghost @click="emit('openWizard')">站点向导</Button>
      <Button type="primary" ghost @click="emit('openDocker')">Docker 发现</Button>
      <Button :disabled="!props.configContent.trim()" @click="emit('parseRaw')">从原始配置解析</Button>
      <Button
        v-if="props.mode === 'raw'"
        type="primary"
        :loading="props.previewing"
        :disabled="!props.selectedServerId"
        @click="emit('saveRaw')"
      >
        保存原始配置
      </Button>
      <Button
        v-else
        type="primary"
        :loading="props.previewing"
        :disabled="!props.selectedServerId"
        @click="emit('saveBlocks')"
      >
        保存分块配置
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
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  padding-bottom: 16px;
  margin-bottom: 16px;
  border-bottom: 1px solid #eef0f4;
}

.toolbar-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.toolbar-group.actions {
  justify-content: flex-end;
}

.server-select {
  min-width: 260px;
}

@media (max-width: 960px) {
  .caddy-toolbar {
    grid-template-columns: 1fr;
  }

  .toolbar-group.actions {
    justify-content: flex-start;
  }
}
</style>
