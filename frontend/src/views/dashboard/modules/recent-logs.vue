<script setup lang="ts">
import type { DashboardRecentItem } from '@/service/api/dashboard';

interface Props {
  logs: DashboardRecentItem[];
}

defineProps<Props>();

function methodClass(method: string) {
  const key = method.toUpperCase();
  if (key === 'GET') return 'bg-green-100 text-green-600';
  if (key === 'POST') return 'bg-blue-100 text-blue-600';
  if (key === 'PUT' || key === 'PATCH') return 'bg-amber-100 text-amber-700';
  if (key === 'DELETE') return 'bg-red-100 text-red-600';
  return 'bg-gray-100 text-gray-600';
}

function statusClass(status: number) {
  if (status >= 500) return 'text-red-500';
  if (status >= 400) return 'text-orange-500';
  if (status >= 300) return 'text-blue-500';
  return 'text-green-600';
}

function formatMeta(item: DashboardRecentItem) {
  const time = item.logTime?.slice(11) || '';
  const ip = item.remoteIp ? ` · ${item.remoteIp}` : '';
  return `${time}${ip}`;
}
</script>

<template>
  <NCard title="实时日志" class="flex-1 rounded-2xl shadow-sm">
    <div v-if="logs.length === 0" class="text-xs text-gray-400">暂无日志</div>
    <div v-else class="flex flex-col gap-3 text-xs">
      <div
        v-for="item in logs"
        :key="item.id"
        class="log-row border-b border-gray-100 pb-2"
      >
        <div class="log-method">
          <span :class="['px-1 rounded', methodClass(item.method)]">{{ item.method || 'N/A' }}</span>
        </div>
        <NTooltip placement="top-start" :show-arrow="false">
          <template #trigger>
            <div class="log-path text-gray-600 truncate">{{ item.uri || '-' }}</div>
          </template>
          {{ item.uri || '-' }}
        </NTooltip>
        <div class="log-status" :class="statusClass(item.status)">{{ item.status }}</div>
        <div class="log-meta text-gray-400">{{ formatMeta(item) }}</div>
      </div>
    </div>
  </NCard>
</template>

<style scoped>
.log-row {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr) 60px 160px;
  align-items: center;
  column-gap: 12px;
}

.log-method,
.log-status,
.log-meta {
  white-space: nowrap;
}

.log-path {
  min-width: 0;
}

@media (max-width: 768px) {
  .log-row {
    grid-template-columns: 48px minmax(0, 1fr) 48px 120px;
    column-gap: 8px;
  }
}

@media (max-width: 480px) {
  .log-row {
    grid-template-columns: 44px minmax(0, 1fr) 44px 96px;
    column-gap: 6px;
  }
}
</style>
