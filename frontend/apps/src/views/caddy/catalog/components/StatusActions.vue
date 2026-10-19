<script lang="ts" setup>
import { Alert, Button, Card, Empty, Tag } from 'antdv-next';

import type { ServerStatusItem } from '../../shared/caddy-server-status';
import {
  formatLatency,
  statusErrorSummary,
  statusLabel,
  statusTagColor,
} from '../../shared/caddy-server-status';

defineProps<{
  serversCount: number;
  lastProbedAt: string;
  serverStatusError: string;
  serverStatusLoaded: boolean;
  selectedServerId?: number;
  serverStatusRows: Array<{
    id: number;
    name: string;
    url?: string;
    status?: ServerStatusItem;
  }>;
}>();

const emit = defineEmits<{
  selectServer: [id: number];
  openWorkbench: [];
}>();
</script>

<template>
  <Card size="small" class="server-status-card" variant="outlined">
    <template #title>
      <div class="server-status-title">
        <span>节点状态</span>
        <span v-if="lastProbedAt" class="text-muted">最近探测：{{ lastProbedAt }}</span>
        <span v-else class="text-muted">
          点击「探测状态」检查节点在线与延迟（不会修改运行配置）
        </span>
      </div>
    </template>
    <Alert
      v-if="serverStatusError"
      class="mb-3"
      type="warning"
      show-icon
      :message="serverStatusError"
    />
    <div v-if="serversCount === 0" class="sidebar-empty">
      <Empty description="暂无已登记 Caddy 节点">
        <Button type="primary" @click="emit('openWorkbench')">前往配置管理添加</Button>
      </Empty>
    </div>
    <div v-else class="server-status-list">
      <button
        v-for="row in serverStatusRows"
        :key="row.id"
        type="button"
        class="server-status-item"
        :class="{
          active: row.id === selectedServerId,
          online: row.status?.reachable,
          offline: row.status && !row.status.reachable,
        }"
        @click="emit('selectServer', row.id)"
      >
        <div class="server-status-main">
          <div class="server-status-name-line">
            <span class="server-status-name">{{ row.name }}</span>
            <Tag :color="statusTagColor(row.status)">{{ statusLabel(row.status) }}</Tag>
          </div>
          <div class="server-status-url text-muted">{{ row.url || '—' }}</div>
        </div>
        <div class="server-status-meta">
          <div class="server-status-latency">
            <span class="meta-label">延迟</span>
            <strong>{{
              serverStatusLoaded || row.status
                ? formatLatency(row.status?.latencyMs)
                : '未探测'
            }}</strong>
          </div>
          <div
            v-if="row.status && !row.status.reachable"
            class="server-status-error text-error"
          >
            {{ statusErrorSummary(row.status) || '探测失败' }}
          </div>
          <div v-else-if="row.status?.probedAt" class="server-status-probed text-muted">
            {{ row.status.probedAt }}
          </div>
        </div>
      </button>
    </div>
  </Card>
</template>

<style scoped>
.server-status-card {
  margin-bottom: 24px;
  border-color: hsl(var(--border));
  border-radius: 8px;
}

.server-status-title {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: baseline;
}

.server-status-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}

.server-status-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 96px;
  padding: 12px 14px;
  text-align: left;
  cursor: pointer;
  background: hsl(var(--muted) / 40%);
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    background 0.16s ease;
}

.server-status-item:hover {
  border-color: hsl(var(--primary) / 60%);
}

.server-status-item.active {
  background: hsl(var(--primary) / 10%);
  border-color: hsl(var(--primary));
  box-shadow: 0 0 0 2px hsl(var(--primary) / 8%);
}

.server-status-item.online {
  border-left: 3px solid hsl(var(--success));
}

.server-status-item.offline {
  border-left: 3px solid hsl(var(--destructive));
}

.server-status-name-line {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.server-status-name {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.server-status-url {
  margin-top: 4px;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.server-status-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.server-status-latency {
  display: flex;
  gap: 8px;
  align-items: baseline;
}

.server-status-error {
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta-label {
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.sidebar-empty {
  padding: 24px 0;
}

.text-muted {
  color: hsl(var(--muted-foreground));
}

.text-error {
  color: hsl(var(--destructive));
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
