<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import { computed, onMounted, reactive, ref } from 'vue';

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
  getNotificationChannelsApi,
  getNotificationLogsApi,
  getNotificationRulesApi,
} from '#/api/notification';

defineOptions({ name: 'NotificationLog' });

type NotificationLogRecord = Partial<NotificationApi.NotificationLog> &
  Record<string, any>;

// ── List state ──────────────────────────────────────────────

const loading = ref(false);
const logs = ref<NotificationApi.NotificationLog[]>([]);
const channelNameMap = ref<Record<string, string>>({});
const ruleNameMap = ref<Record<string, string>>({});
const selectedRowKeys = ref<Array<number | string>>([]);

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
});

async function fetchLogs() {
  loading.value = true;
  try {
    const data = await getNotificationLogsApi({
      page: pagination.current,
      pageSize: pagination.pageSize,
    });
    logs.value = data.list ?? [];
    pagination.total = data.total ?? 0;
  } finally {
    loading.value = false;
  }
}

async function fetchReferences() {
  const [channels, rules] = await Promise.all([
    getNotificationChannelsApi(),
    getNotificationRulesApi(),
  ]);

  channelNameMap.value = Object.fromEntries(
    channels.map((item) => [String(item.id), item.name]),
  );
  ruleNameMap.value = Object.fromEntries(
    rules.map((item) => [String(item.id), item.name]),
  );
}

// ── Table columns ───────────────────────────────────────────

const columns = [
  { dataIndex: 'channelId', key: 'channel', title: '渠道' },
  { dataIndex: 'ruleId', key: 'rule', title: '规则' },
  { dataIndex: 'title', key: 'title', title: '主题', ellipsis: true },
  { dataIndex: 'status', key: 'status', title: '状态' },
  { dataIndex: 'createdAt', key: 'createdAt', title: '发送时间' },
  { key: 'actions', title: '操作', width: 100 },
];

// ── Row selection ───────────────────────────────────────────

const rowSelection = computed(() => ({
  onChange: (keys: Array<number | string>) => {
    selectedRowKeys.value = keys;
  },
  selectedRowKeys: selectedRowKeys.value,
}));

// ── Pagination change ───────────────────────────────────────

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  void fetchLogs();
}

// ── 删除 single log ───────────────────────────────────────

async function handleDelete(id: number | string) {
  try {
    await deleteNotificationLogApi(String(id));
    message.success('发送日志已删除');
    await fetchLogs();
  } catch {
    message.error('删除失败');
  }
}

// ── Batch delete ────────────────────────────────────────────

async function handleBatchDelete() {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请至少选择一条日志');
    return;
  }
  try {
    await batchDeleteNotificationLogsApi({ ids: selectedRowKeys.value });
    message.success(`已删除 ${selectedRowKeys.value.length} 条日志`);
    selectedRowKeys.value = [];
    await fetchLogs();
  } catch {
    message.error('批量删除失败');
  }
}

// ── Clear all ───────────────────────────────────────────────

async function handleClearAll() {
  try {
    await clearNotificationLogsApi();
    message.success('发送日志已清空');
    selectedRowKeys.value = [];
    await fetchLogs();
  } catch {
    message.error('清空失败');
  }
}

function channelLabel(record: NotificationLogRecord) {
  const id = record.channelId;
  if (id === undefined || id === null || id === '' || Number(id) === 0) return '-';
  return record.channelName || channelNameMap.value[String(id)] || `#${id}`;
}

function ruleLabel(record: NotificationLogRecord) {
  const id = record.ruleId;
  if (id === undefined || id === null || id === '' || Number(id) === 0) return '-';
  return record.ruleName || ruleNameMap.value[String(id)] || `#${id}`;
}

function subjectText(record: NotificationLogRecord) {
  return record.title || record.content || record.message || '-';
}

function statusColor(status: NotificationApi.NotificationLog['status']) {
  if (status === 2 || status === 'sent' || status === 'success') return 'green';
  if (status === 3 || status === 'failed') return 'red';
  if (status === 1 || status === 'sending') return 'blue';
  return 'default';
}

function statusLabel(status: NotificationApi.NotificationLog['status']) {
  if (status === 0 || status === 'pending') return '等待中';
  if (status === 1 || status === 'sending') return '发送中';
  if (status === 2 || status === 'sent' || status === 'success') return '已发送';
  if (status === 3 || status === 'failed') return '失败';
  return status || '-';
}

// ── Init ────────────────────────────────────────────────────

onMounted(() => {
  void fetchReferences();
  void fetchLogs();
});
</script>

<template>
  <div class="p-5">
    <Card title="发送日志">
      <template #extra>
        <Space>
          <Popconfirm
            title="确认删除选中的发送日志？"
            :disabled="selectedRowKeys.length === 0"
            @confirm="handleBatchDelete"
          >
            <Button danger :disabled="selectedRowKeys.length === 0">
              批量删除{{ selectedRowKeys.length > 0 ? ` (${selectedRowKeys.length})` : '' }}
            </Button>
          </Popconfirm>
          <Popconfirm
            title="确认清空全部发送日志？此操作不可恢复。"
            @confirm="handleClearAll"
          >
            <Button danger>
              清空全部
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
          showTotal: (t: number) => `共 ${t} 条`,
        }"
        :row-selection="rowSelection"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'channel'">
            {{ channelLabel(record) }}
          </template>

          <template v-if="column.key === 'rule'">
            {{ ruleLabel(record) }}
          </template>

          <template v-if="column.key === 'title'">
            {{ subjectText(record).substring(0, 80) }}{{ subjectText(record).length > 80 ? '...' : '' }}
          </template>

          <template v-if="column.key === 'status'">
            <Tag :color="statusColor(record.status)">
              {{ statusLabel(record.status) }}
            </Tag>
          </template>

          <template v-if="column.key === 'createdAt'">
            {{ record.createdAt }}
          </template>

          <template v-if="column.key === 'actions'">
            <Popconfirm
              title="确认删除该发送日志？"
              @confirm="handleDelete(record.id)"
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
