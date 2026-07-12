<script lang="ts" setup>
/** 目录页工具栏 / 辅助导航（跳转配置工作台，无页内表单） */
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
    section?: 'toolbar' | 'secondary';
    selectedServerId?: number;
    serverOptions?: Array<{ label: string; value: number }>;
    loadingServers?: boolean;
    selectedServerStatus?: ServerStatusItem | null;
    loadingServerStatus?: boolean;
    loadingConfig?: boolean;
    loadingSiteMetrics?: boolean;
  }>(),
  {
    section: 'toolbar',
    serverOptions: () => [],
    loadingServers: false,
    loadingServerStatus: false,
    loadingConfig: false,
    loadingSiteMetrics: false,
  },
);

const emit = defineEmits<{
  'update:selectedServerId': [value: number];
  refreshStatus: [];
  refreshSites: [];
  refreshMetrics: [];
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
    <Button :loading="loadingConfig" @click="emit('refreshSites')">刷新站点</Button>
    <Button :loading="loadingSiteMetrics" @click="emit('refreshMetrics')">
      刷新指标
    </Button>
    <Button type="primary" :disabled="!selectedServerId" @click="emit('openWorkbench')">
      配置工作台
    </Button>
    <Button :disabled="!selectedServerId" @click="emit('openWaf')">WAF</Button>
  </Space>

  <div v-else class="catalog-secondary">
    <Space wrap>
      <span class="text-muted">辅助入口（仍走既有 Apply_Path）：</span>
      <Button size="small" @click="emit('openWorkbench', 'blocks')">
        站点向导 / 编辑
      </Button>
      <Button size="small" @click="emit('openWorkbench')">
        Docker 发现（工作台内）
      </Button>
      <Button size="small" @click="emit('openWaf')">防火墙 WAF</Button>
    </Space>
  </div>
</template>

<style scoped>
.server-select {
  min-width: 220px;
}

.catalog-secondary {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.text-muted {
  color: #8c8c8c;
}
</style>
