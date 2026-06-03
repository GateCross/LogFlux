<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { onMounted, reactive, ref } from 'vue';

import {
  Button,
  Card,
  Popconfirm,
  Space,
  Table,
  Tag,
  message,
} from 'ant-design-vue';

import {
  batchDeleteNotificationLogsApi,
  clearNotificationLogsApi,
  deleteNotificationLogApi,
  getNotificationLogsApi,
} from '#/api/notification';

defineOptions({ name: 'NotificationLog' });

// ── List state ──────────────────────────────────────────────

const loading = ref(false);
const logs = ref<NotificationApi.NotificationLog[]>([]);
const selectedRowKeys = ref<string[]>([]);

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
});

async function fetchLogs() {
  loading.value = true;
  try {
    const data = await getNotificationLogsApi({
      current: pagination.current,
      pageSize: pagination.pageSize,
    });
    logs.value = Array.isArray(data) ? data : [];
    pagination.total = Array.isArray(data) ? data.length : 0;
  } finally {
    loading.value = false;
  }
}

// ── Table columns ───────────────────────────────────────────

const columns = [
  { dataIndex: 'channelName', key: 'channelName', title: '渠道' },
  { dataIndex: 'ruleName', key: 'ruleName', title: 'Recipient' },
  { dataIndex: 'content', key: 'content', title: '主题', ellipsis: true },
  { dataIndex: 'status', key: 'status', title: '状态' },
  { dataIndex: 'createdAt', key: 'createdAt', title: '发送时间' },
  { key: 'actions', title: '操作', width: 100 },
];

// ── Row selection ───────────────────────────────────────────

const rowSelection = {
  onChange: (keys: string[]) => {
    selectedRowKeys.value = keys;
  },
  selectedRowKeys,
};

// ── Pagination change ───────────────────────────────────────

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchLogs();
}

// ── 删除 single log ───────────────────────────────────────

async function handle删除(id: string) {
  try {
    await deleteNotificationLogApi(id);
    message.success('Log deleted');
    await fetchLogs();
  } catch {
    message.error('删除 failed');
  }
}

// ── Batch delete ────────────────────────────────────────────

async function handleBatch删除() {
  if (selectedRowKeys.value.length === 0) {
    message.warning('Please select at least one log');
    return;
  }
  try {
    await batchDeleteNotificationLogsApi({ ids: selectedRowKeys.value });
    message.success(`${selectedRowKeys.value.length} logs deleted`);
    selectedRowKeys.value = [];
    await fetchLogs();
  } catch {
    message.error('Batch delete failed');
  }
}

// ── Clear all ───────────────────────────────────────────────

async function handleClearAll() {
  try {
    await clearNotificationLogsApi();
    message.success('All logs cleared');
    selectedRowKeys.value = [];
    await fetchLogs();
  } catch {
    message.error('Clear failed');
  }
}

// ── Init ────────────────────────────────────────────────────

onMounted(() => {
  fetchLogs();
});
</script>

<template>
  <div class="p-5">
    <Card title="Notification Logs">
      <template #extra>
        <Space>
          <Popconfirm
            title="Are you sure to delete the selected logs?"
            :disabled="selectedRowKeys.length === 0"
            @confirm="handleBatch删除"
          >
            <Button danger :disabled="selectedRowKeys.length === 0">
              Batch 删除{{ selectedRowKeys.length > 0 ? ` (${selectedRowKeys.length})` : '' }}
            </Button>
          </Popconfirm>
          <Popconfirm
            title="Are you sure to clear all notification logs? This cannot be undone."
            @confirm="handleClearAll"
          >
            <Button danger>
              Clear All
            </Button>
          </Popconfirm>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="logs"
        :loading="loading"
        :pagination="{
          current: pagination.current,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (t: number) => `Total ${t} records`,
        }"
        :row-selection="rowSelection"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <Tag
              :color="
                record.status === 'sent'
                  ? 'green'
                  : record.status === 'failed'
                    ? 'red'
                    : 'blue'
              "
            >
              {{ record.status }}
            </Tag>
          </template>

          <template v-if="column.key === 'createdAt'">
            {{ record.createdAt }}
          </template>

          <template v-if="column.key === 'content'">
            {{ record.content?.substring(0, 80) }}{{ record.content?.length > 80 ? '...' : '' }}
          </template>

          <template v-if="column.key === 'actions'">
            <Popconfirm
              title="Are you sure to delete this log?"
              @confirm="handle删除(record.id)"
            >
              <Button size="small" type="link" danger>
                删除
              </Button>
            </Popconfirm>
          </template>
        </template>
      </Table>
    </Card>
  </div>
</template>
