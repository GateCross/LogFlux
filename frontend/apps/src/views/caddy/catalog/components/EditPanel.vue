<script lang="ts" setup>
/** 目录页节点工具栏（节点选择/探测/工作台深链）；站点刷新与添加在卡片网格工具条 */
import { Button, Select, Space, Tag, Tooltip } from 'antdv-next';

import type { ServerStatusItem } from '../../shared/caddy-server-status';
import {
  formatLatency,
  statusErrorSummary,
  statusLabel,
  statusTagColor,
} from '../../shared/caddy-server-status';

withDefaults(
  defineProps<{
    section?: 'toolbar';
    selectedServerId?: number;
    serverOptions?: Array<{ label: string; value: number }>;
    loadingServers?: boolean;
    selectedServerStatus?: ServerStatusItem | null;
    loadingServerStatus?: boolean;
    loadingConfig?: boolean;
  }>(),
  {
    section: 'toolbar',
    serverOptions: () => [],
    loadingServers: false,
    loadingServerStatus: false,
    loadingConfig: false,
  },
);

const emit = defineEmits<{
  'update:selectedServerId': [value: number];
  refreshStatus: [];
  openWorkbench: [mode?: 'blocks' | 'waf' | 'raw' | 'preview'];
  openWaf: [];
}>();

function handleServerChange(value: unknown) {
  const next = Number(value);
  if (!Number.isFinite(next)) return;
  emit('update:selectedServerId', next);
}
</script>

<template>
  <Space v-if="section === 'toolbar'" wrap>
    <Select
      :value="selectedServerId"
      :options="serverOptions"
      :loading="loadingServers"
      class="server-select"
      placeholder="选择 Caddy 节点"
      @change="handleServerChange"
    />
    <Tooltip
      v-if="selectedServerStatus"
      :title="
        statusErrorSummary(selectedServerStatus) ||
        `延迟 ${formatLatency(selectedServerStatus.latencyMs)} · ${selectedServerStatus.probedAt || ''}`
      "
    >
      <Tag :color="statusTagColor(selectedServerStatus)">
        {{ statusLabel(selectedServerStatus) }}
      </Tag>
    </Tooltip>
    <Button :loading="loadingServerStatus" @click="emit('refreshStatus')">
      探测状态
    </Button>
    <Button type="primary" :disabled="!selectedServerId" @click="emit('openWorkbench')">
      配置工作台
    </Button>
    <Button :disabled="!selectedServerId" @click="emit('openWaf')">WAF</Button>
  </Space>
</template>

<style scoped>
.server-select {
  min-width: 220px;
}
</style>
