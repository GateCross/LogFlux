<script setup lang="ts">
import { Button, Popconfirm, Space } from 'antdv-next';

defineProps<{
  row: { id: number; status: number };
}>();

const emit = defineEmits<{
  edit: [];
  delete: [id: number];
  toggleStatus: [];
}>();
</script>

<template>
  <Space :size="6">
    <Button
      size="small"
      class="table-action-btn table-action-btn--secondary"
      @click="emit('edit')"
    >
      编辑
    </Button>
    <Button
      v-if="row.status === 1"
      size="small"
      class="table-action-btn table-action-btn--warning"
      @click="emit('toggleStatus')"
    >
      冻结
    </Button>
    <Button
      v-else
      size="small"
      class="table-action-btn table-action-btn--primary"
      @click="emit('toggleStatus')"
    >
      解冻
    </Button>
    <Popconfirm
      title="确认永久删除该用户吗？此操作无法恢复！"
      ok-text="确认"
      cancel-text="取消"
      @confirm="emit('delete', row.id)"
    >
      <Button
        size="small"
        danger
        class="table-action-btn table-action-btn--danger"
      >
        删除
      </Button>
    </Popconfirm>
  </Space>
</template>
