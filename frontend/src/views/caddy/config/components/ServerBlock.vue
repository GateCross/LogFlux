<script setup lang="ts">
import type { CaddyServer } from '../composables/useCaddyServers';

defineProps<{
  servers: CaddyServer[];
  serverOptions: Array<{ label: string; value: number }>;
  currentServerId: number | null;
}>();

const emit = defineEmits<{
  (e: 'update:currentServerId', id: number | null): void;
  (e: 'add'): void;
  (e: 'edit'): void;
  (e: 'delete'): void;
}>();
</script>

<template>
  <div class="flex min-w-0 flex-1 items-center gap-2">
    <NSelect
      :value="currentServerId"
      :options="serverOptions"
      placeholder="选择服务器"
      class="max-w-72 w-full"
      size="small"
      @update:value="emit('update:currentServerId', $event)"
    />
    <NDropdown
      :options="[
        { label: '添加服务器', key: 'add' },
        { label: '编辑当前服务器', key: 'edit', disabled: !currentServerId },
        { label: '删除当前服务器', key: 'delete', disabled: !currentServerId }
      ]"
      @select="(key: string) => { if (key === 'add') emit('add'); else if (key === 'edit') emit('edit'); else if (key === 'delete') emit('delete'); }"
    >
      <NButton size="small" secondary>管理</NButton>
    </NDropdown>
  </div>
</template>
