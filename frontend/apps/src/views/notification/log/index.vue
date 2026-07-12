<script lang="ts" setup>
import type { NotificationApi } from '#/api/notification';

import {
  computed,
  nextTick,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  watch,
} from 'vue';
import { useQueryClient } from '@tanstack/vue-query';

import { Page } from '@vben/common-ui';
import {
  Alert,
  Button,
  Card,
  Popconfirm,
  Space,
  Table,
  Tag,
  message,
} from 'antdv-next';

import {
  batchDeleteNotificationLogsApi,
  clearNotificationLogsApi,
  deleteNotificationLogApi,
  getNotificationChannelsApi,
  getNotificationLogsApi,
  getNotificationRulesApi,
} from '#/api/notification';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

defineOptions({ name: 'NotificationLog' });

type NotificationLogRecord = Partial<NotificationApi.NotificationLog> &
  Record<string, any>;

const queryClient = useQueryClient();
const selectedRowKeys = ref<Array<number | string>>([]);
const tableWrapRef = ref<HTMLElement | null>(null);
const tableScrollY = ref<number>();
let tableResizeObserver: ResizeObserver | null = null;

const pagination = reactive({
  current: 1,
  pageSize: 20,
  pageSizeOptions: ['10', '20', '50', '100'],
  showSizeChanger: true,
  showTotal: (t: number) => `共 ${t} 条`,
  total: 0,
});

const logParams = computed(() => ({
  page: pagination.current,
  pageSize: pagination.pageSize,
}));

const {
  data: logsPage,
  loading,
  errorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: computed(() => qk.notification.logs(logParams.value)),
  queryFn: () => getNotificationLogsApi(logParams.value, withListDetailErrorMode()),
  errorFallback: '加载发送日志失败',
});

const logs = computed(() => {
  const page = logsPage.value as any;
  if (Array.isArray(page)) return page;
  return page?.list ?? [];
});

const hasLogs = computed(() => logs.value.length > 0);

const tableScroll = computed(() => {
  if (!hasLogs.value || !tableScrollY.value) return undefined;
  return { y: tableScrollY.value };
});

watch(
  logsPage,
  (page) => {
    const total = (page as any)?.total;
    pagination.total = typeof total === 'number' ? total : logs.value.length;
  },
  { immediate: true },
);

const { data: channelsData } = useListDetailQuery({
  queryKey: qk.notification.channels({ for: 'log-ref' }),
  queryFn: () => getNotificationChannelsApi(withListDetailErrorMode()),
  errorFallback: '加载渠道失败',
});

const { data: rulesData } = useListDetailQuery({
  queryKey: qk.notification.rules({ for: 'log-ref' }),
  queryFn: () => getNotificationRulesApi(withListDetailErrorMode()),
  errorFallback: '加载规则失败',
});

const channelNameMap = computed(() =>
  Object.fromEntries((channelsData.value ?? []).map((item) => [String(item.id), item.name])),
);
const ruleNameMap = computed(() =>
  Object.fromEntries((rulesData.value ?? []).map((item) => [String(item.id), item.name])),
);

const columns = [
  { dataIndex: 'channelId', key: 'channel', title: '渠道' },
  { dataIndex: 'ruleId', key: 'rule', title: '规则' },
  { dataIndex: 'title', key: 'title', title: '主题', ellipsis: true },
  { dataIndex: 'status', key: 'status', title: '状态', width: 100 },
  { dataIndex: 'createdAt', key: 'createdAt', title: '发送时间', width: 180 },
  { key: 'actions', title: '操作', width: 100 },
];

const rowSelection = computed(() => ({
  onChange: (keys: Array<number | string>) => {
    selectedRowKeys.value = keys;
  },
  selectedRowKeys: selectedRowKeys.value,
}));

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
}

async function handleDelete(id: number | string) {
  try {
    await deleteNotificationLogApi(String(id));
    message.success('发送日志已删除');
    await invalidateListDetailQueries(queryClient, ['notification', 'logs']);
    await refetch();
  } catch {
    message.error('删除失败');
  }
}

async function handleBatchDelete() {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请至少选择一条日志');
    return;
  }
  try {
    await batchDeleteNotificationLogsApi({ ids: selectedRowKeys.value });
    message.success(`已删除 ${selectedRowKeys.value.length} 条日志`);
    selectedRowKeys.value = [];
    await invalidateListDetailQueries(queryClient, ['notification', 'logs']);
    await refetch();
  } catch {
    message.error('批量删除失败');
  }
}

async function handleClearAll() {
  try {
    await clearNotificationLogsApi();
    message.success('发送日志已清空');
    selectedRowKeys.value = [];
    await invalidateListDetailQueries(queryClient, ['notification', 'logs']);
    await refetch();
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

/** 按表格容器实际高度计算 body 滚动区，避免表格撑出视口 */
function updateTableScrollY() {
  const wrap = tableWrapRef.value;
  if (!wrap || !hasLogs.value) {
    tableScrollY.value = undefined;
    return;
  }

  const available = wrap.clientHeight;
  if (available <= 0) {
    tableScrollY.value = undefined;
    return;
  }

  const header = wrap.querySelector<HTMLElement>('.ant-table-thead');
  const paginationEl = wrap.querySelector<HTMLElement>('.ant-pagination');
  const headerHeight = header?.offsetHeight ?? 55;
  const paginationHeight = paginationEl
    ? paginationEl.offsetHeight + 16
    : 56;
  const nextY = available - headerHeight - paginationHeight;
  tableScrollY.value = Math.max(200, Math.floor(nextY));

  const bodies = wrap.querySelectorAll<HTMLElement>(
    '.ant-table-body, .ant-table-body-outer',
  );
  bodies.forEach((el) => {
    el.style.height = `${tableScrollY.value}px`;
    el.style.maxHeight = `${tableScrollY.value}px`;
  });
}

function scheduleUpdateTableScrollY() {
  void nextTick(() => {
    updateTableScrollY();
    requestAnimationFrame(() => updateTableScrollY());
  });
}

watch(
  () => [loading.value, logs.value.length] as const,
  () => {
    scheduleUpdateTableScrollY();
  },
);

onMounted(() => {
  tableResizeObserver = new ResizeObserver(() => updateTableScrollY());
  if (tableWrapRef.value) tableResizeObserver.observe(tableWrapRef.value);
  scheduleUpdateTableScrollY();
});

onUnmounted(() => {
  tableResizeObserver?.disconnect();
});
</script>

<template>
  <Page auto-content-height content-class="overflow-hidden">
    <div class="notification-log-page">
      <Card title="发送日志" variant="borderless" class="notification-log-card">
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
            <Popconfirm title="确认清空全部发送日志？此操作不可恢复。" @confirm="handleClearAll">
              <Button danger>清空全部</Button>
            </Popconfirm>
          </Space>
        </template>

        <Alert
          v-if="errorMessage"
          class="mb-3"
          type="error"
          show-icon
          :message="errorMessage"
        />

        <div ref="tableWrapRef" class="notification-log-table-wrap">
          <Table
            :columns="columns"
            :data-source="logs"
            :loading="loading"
            :pagination="pagination"
            :scroll="tableScroll"
            :row-selection="rowSelection"
            class="notification-log-table"
            row-key="id"
            size="middle"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record: raw }">
              <template v-if="column.key === 'channel'">
                {{ channelLabel(raw as unknown as NotificationLogRecord) }}
              </template>

              <template v-if="column.key === 'rule'">
                {{ ruleLabel(raw as unknown as NotificationLogRecord) }}
              </template>

              <template v-if="column.key === 'title'">
                {{
                  subjectText(raw as unknown as NotificationLogRecord).substring(0, 80)
                }}{{
                  subjectText(raw as unknown as NotificationLogRecord).length > 80
                    ? '...'
                    : ''
                }}
              </template>

              <template v-if="column.key === 'status'">
                <Tag
                  :color="
                    statusColor(
                      (raw as unknown as NotificationLogRecord).status as NotificationApi.NotificationLog['status'],
                    )
                  "
                >
                  {{
                    statusLabel(
                      (raw as unknown as NotificationLogRecord).status as NotificationApi.NotificationLog['status'],
                    )
                  }}
                </Tag>
              </template>

              <template v-if="column.key === 'createdAt'">
                {{ (raw as unknown as NotificationLogRecord).createdAt }}
              </template>

              <template v-if="column.key === 'actions'">
                <Popconfirm
                  title="确认删除该发送日志？"
                  @confirm="
                    handleDelete(
                      (raw as unknown as NotificationLogRecord).id as number | string,
                    )
                  "
                >
                  <Button size="small" danger class="table-action-btn table-action-btn--danger">
                    删除
                  </Button>
                </Popconfirm>
              </template>
            </template>
          </Table>
        </div>
      </Card>
    </div>
  </Page>
</template>

<style scoped>
.notification-log-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.notification-log-card {
  display: flex;
  flex: 1;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.notification-log-card :deep(.ant-card-body) {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.notification-log-table-wrap {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.notification-log-table {
  flex: 1;
  height: 100%;
  min-height: 0;
}

.notification-log-table :deep(.ant-spin-nested-loading),
.notification-log-table :deep(.ant-spin-container) {
  height: 100%;
}

.notification-log-table :deep(.ant-spin-container) {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.notification-log-table :deep(.ant-table) {
  flex: 1;
  min-height: 0;
}

.notification-log-table :deep(.ant-table-container) {
  height: 100%;
}

.notification-log-table :deep(.ant-table-pagination.ant-pagination) {
  flex-shrink: 0;
  margin: 12px 0 0;
}

.notification-log-table :deep(.ant-table-cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mb-3 {
  margin-bottom: 12px;
  flex-shrink: 0;
}
</style>
