<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import {
  Button,
  Popconfirm,
  Space,
  Table,
  Tag,
} from 'antdv-next';

import { channelEndpoint, channelTypeLabel } from '../channel-utils';

defineProps<{
  channels: NotificationApi.Channel[];
  loading: boolean;
  testing: boolean;
  testTargetId: string;
}>();

const emit = defineEmits<{
  test: [record: NotificationApi.Channel];
  edit: [record: NotificationApi.Channel];
  delete: [id: string];
}>();

const columns = [
  { dataIndex: 'name', key: 'name', title: '名称' },
  { dataIndex: 'type', key: 'type', title: '类型', width: 150 },
  { dataIndex: 'enabled', key: 'enabled', title: '状态', width: 100 },
  { key: 'endpoint', title: '地址', ellipsis: true },
  {
    dataIndex: 'description',
    key: 'description',
    title: '描述',
    ellipsis: true,
  },
  { key: 'actions', title: '操作', width: 280 },
];
</script>

<template>
  <Table
    :columns="columns"
    :data-source="channels"
    :loading="loading"
    row-key="id"
    size="middle"
    :pagination="false"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.key === 'type'">
        <Tag color="blue">{{ channelTypeLabel(record.type) }}</Tag>
      </template>

      <template v-if="column.key === 'enabled'">
        <Tag :color="record.enabled ? 'green' : 'default'">
          {{ record.enabled ? '启用' : '停用' }}
        </Tag>
      </template>

      <template v-if="column.key === 'endpoint'">
        {{ channelEndpoint(record) }}
      </template>

      <template v-if="column.key === 'actions'">
        <Space :size="6">
          <Button
            size="small"
            class="table-action-btn table-action-btn--primary"
            :loading="testing && testTargetId === String(record.id)"
            @click="emit('test', record as NotificationApi.Channel)"
          >
            测试
          </Button>
          <Button
            size="small"
            class="table-action-btn"
            @click="emit('edit', record as NotificationApi.Channel)"
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除该通知渠道？"
            @confirm="emit('delete', String(record.id))"
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
    </template>
  </Table>
</template>
