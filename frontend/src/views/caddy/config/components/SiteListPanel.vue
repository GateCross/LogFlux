<script setup lang="ts">
import { computed, ref, toRefs } from 'vue';
import type { Site } from '../types';

const props = defineProps<{
  sites: Site[];
  activeId: string | null;
}>();
const { sites } = toRefs(props);

defineEmits<{
  (e: 'select', id: string): void;
  (e: 'add'): void;
  (e: 'duplicate', id: string): void;
  (e: 'remove', id: string): void;
  (e: 'move', id: string, direction: 'up' | 'down'): void;
}>();

const search = ref('');
const domainCountLabel = (domains: string[]) => {
  const values = domains.filter(Boolean);
  if (values.length > 0 && values.every(v => /^:\d+$/.test(v))) return '端口';
  return '域名';
};
const filteredSites = computed(() => {
  const keyword = search.value.trim().toLowerCase();
  if (!keyword) return sites.value;
  return sites.value.filter(site => {
    const nameMatch = site.name?.toLowerCase().includes(keyword);
    const domainMatch = site.domains.some(domain => domain.toLowerCase().includes(keyword));
    return nameMatch || domainMatch;
  });
});
</script>

<template>
  <NCard
    size="small"
    class="h-full"
    :bordered="false"
    :content-style="{ display: 'flex', flexDirection: 'column', minHeight: 0 }"
  >
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">站点列表</span>
        <NButton size="tiny" type="primary" @click="$emit('add')">新增</NButton>
      </div>
    </template>
    <NInput v-model:value="search" size="small" placeholder="搜索站点/域名" class="mb-3" clearable />
    <NEmpty v-if="filteredSites.length === 0" :description="sites.length === 0 ? '暂无站点' : '无匹配结果'" />
    <div v-else class="min-h-0 flex flex-col flex-1 gap-2 overflow-auto">
      <div
        v-for="site in filteredSites"
        :key="site.id"
        class="cursor-pointer border rounded-md p-2"
        :class="activeId === site.id ? 'border-primary-500 bg-primary-50' : 'border-gray-200'"
        @click="$emit('select', site.id)"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <div class="truncate font-medium">{{ site.name }}</div>
            <div class="truncate text-xs text-gray-500">{{ site.domains.join(', ') || '-' }}</div>
            <div class="mt-1 text-xs text-gray-400">
              {{ domainCountLabel(site.domains) }} {{ site.domains.length }} · 路由 {{ site.routes.length }}
            </div>
          </div>
          <NTag size="small" :type="site.enabled ? 'success' : 'default'">{{ site.enabled ? '启用' : '停用' }}</NTag>
        </div>
        <div class="mt-2 flex items-center gap-1">
          <NButton size="tiny" @click.stop="$emit('duplicate', site.id)">复制</NButton>
          <NButton size="tiny" @click.stop="$emit('move', site.id, 'up')">上移</NButton>
          <NButton size="tiny" @click.stop="$emit('move', site.id, 'down')">下移</NButton>
          <NButton size="tiny" type="error" @click.stop="$emit('remove', site.id)">删除</NButton>
        </div>
      </div>
    </div>
  </NCard>
</template>
