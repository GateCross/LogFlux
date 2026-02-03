<template>
  <n-card size="small" :bordered="false">
    <template #header>
      <div class="flex items-center justify-between">
        <span class="font-semibold">上游池</span>
        <n-button size="tiny" type="primary" @click="addUpstream">新增</n-button>
      </div>
    </template>
    <n-empty v-if="upstreams.length === 0" description="暂无上游" />
    <div v-else class="flex flex-col gap-2">
      <div v-for="up in upstreams" :key="up.name" class="rounded border p-2">
        <div class="flex items-center gap-2">
          <n-input v-model:value="up.name" placeholder="名称" class="flex-1" />
          <n-select v-model:value="up.lbPolicy" :options="lbOptions" size="small" class="w-32" />
          <n-button size="tiny" type="error" @click="removeUpstream(up.name)">删除</n-button>
        </div>
        <div class="mt-2">
          <n-dynamic-tags v-model:value="up.targets" />
        </div>
        <div v-if="!up.name" class="text-xs text-red-500 mt-1">名称不能为空</div>
        <div v-else-if="isDuplicateName(up.name)" class="text-xs text-red-500 mt-1">名称重复</div>
        <div v-if="up.targets.length === 0" class="text-xs text-red-500 mt-1">至少配置一个目标</div>
      </div>
    </div>
  </n-card>
</template>

<script setup lang="ts">
import type { Upstream } from '../types';

const props = defineProps<{
  upstreams: Upstream[];
}>();

const lbOptions = [
  { label: 'round_robin', value: 'round_robin' },
  { label: 'least_conn', value: 'least_conn' },
  { label: 'ip_hash', value: 'ip_hash' }
];

function isDuplicateName(name: string) {
  return props.upstreams.filter(u => u.name === name).length > 1;
}

function addUpstream() {
  props.upstreams.push({
    name: `upstream-${props.upstreams.length + 1}`,
    targets: ['localhost:8080'],
    lbPolicy: 'round_robin'
  });
}

function removeUpstream(name: string) {
  const idx = props.upstreams.findIndex(u => u.name === name);
  if (idx >= 0) props.upstreams.splice(idx, 1);
}
</script>
