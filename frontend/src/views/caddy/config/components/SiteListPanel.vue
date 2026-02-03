<template>
  <n-card
    size="small"
    class="h-full"
    :bordered="false"
    :content-style="{ display: 'flex', flexDirection: 'column', minHeight: 0 }"
  >
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">站点列表</span>
        <n-button size="tiny" type="primary" @click="$emit('add')">新增</n-button>
      </div>
    </template>
    <n-empty v-if="sites.length === 0" description="暂无站点" />
    <div v-else class="flex-1 min-h-0 overflow-auto flex flex-col gap-2">
      <div
        v-for="site in sites"
        :key="site.id"
        class="border rounded-md p-2 cursor-pointer"
        :class="activeId === site.id ? 'border-primary-500 bg-primary-50' : 'border-gray-200'"
        @click="$emit('select', site.id)"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <div class="font-medium truncate">{{ site.name }}</div>
            <div class="text-xs text-gray-500 truncate">{{ site.domains.join(', ') || '-' }}</div>
          </div>
          <n-tag size="small" :type="site.enabled ? 'success' : 'default'">{{ site.enabled ? '启用' : '停用' }}</n-tag>
        </div>
        <div class="mt-2 flex items-center gap-1">
          <n-button size="tiny" @click.stop="$emit('duplicate', site.id)">复制</n-button>
          <n-button size="tiny" @click.stop="$emit('move', site.id, 'up')">上移</n-button>
          <n-button size="tiny" @click.stop="$emit('move', site.id, 'down')">下移</n-button>
          <n-button size="tiny" type="error" @click.stop="$emit('remove', site.id)">删除</n-button>
        </div>
      </div>
    </div>
  </n-card>
</template>

<script setup lang="ts">
import type { Site } from '../types';

defineProps<{
  sites: Site[];
  activeId: string | null;
}>();

defineEmits<{
  (e: 'select', id: string): void;
  (e: 'add'): void;
  (e: 'duplicate', id: string): void;
  (e: 'remove', id: string): void;
  (e: 'move', id: string, direction: 'up' | 'down'): void;
}>();
</script>
