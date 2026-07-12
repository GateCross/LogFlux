<script lang="ts" setup>
import type { CaddyServerApi } from '#/api/caddy/server';

import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useQueryClient } from '@tanstack/vue-query';

import { Page } from '@vben/common-ui';

import {
  Alert,
  Button,
  Card,
  DatePicker,
  Descriptions,
  DescriptionsItem,
  Form,
  FormItem,
  Input,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
} from 'antdv-next';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import { clearCaddyLogsApi, getCaddyLogsApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { invalidateListDetailQueries } from '#/api/list-detail-mutation';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';

defineOptions({ name: 'CaddyAccessLog' });

type CaddyLog = CaddyServerApi.CaddyLogItem;
type SortOrder = 'ascend' | 'descend' | false;
type ColumnWidthKey =
  | 'actions'
  | 'clientIp'
  | 'host'
  | 'location'
  | 'logTime'
  | 'method'
  | 'status'
  | 'uri';

const route = useRoute();
const router = useRouter();
const queryClient = useQueryClient();

const clearing = ref(false);
const selectedLog = ref<CaddyLog | null>(null);
const detailVisible = ref(false);
const sortState = ref<{ order: SortOrder }>({ order: 'descend' });
const tableWrapRef = ref<HTMLElement | null>(null);
const tableScrollY = ref<number>();
let tableResizeObserver: ResizeObserver | null = null;

const filters = reactive({
  keyword: '',
  host: '',
  status: -1,
  timeRange: undefined as [Dayjs, Dayjs] | undefined,
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  pageSizeOptions: ['10', '20', '50', '100'],
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
  total: 0,
});

function formatRangeTime(value: Dayjs | undefined) {
  if (!value) return undefined;
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss');
}

function applyHostFromRoute() {
  const raw = route.query.host;
  const host = Array.isArray(raw) ? String(raw[0] ?? '') : String(raw ?? '');
  filters.host = host.trim();
}

const listParams = computed<CaddyServerApi.CaddyLogQuery>(() => {
  const [start, end] = filters.timeRange || [];
  return {
    page: pagination.current,
    pageSize: pagination.pageSize,
    keyword: filters.keyword || undefined,
    host: filters.host || undefined,
    status: filters.status,
    startTime: formatRangeTime(start),
    endTime: formatRangeTime(end),
    sortBy: sortState.value.order ? ('logTime' as const) : undefined,
    order:
      sortState.value.order === 'ascend'
        ? 'asc'
        : sortState.value.order === 'descend'
          ? 'desc'
          : undefined,
  };
});

const {
  data: logsPage,
  loading,
  errorMessage,
  refetch,
} = useListDetailQuery({
  queryKey: computed(() => qk.caddy.logs(listParams.value)),
  queryFn: () =>
    getCaddyLogsApi(listParams.value, withListDetailErrorMode()),
  errorFallback: '获取访问日志失败',
});

const logs = computed(() => logsPage.value?.list ?? []);

watch(
  logsPage,
  (page) => {
    pagination.total = page?.total ?? 0;
    scheduleUpdateTableScrollY();
  },
  { immediate: true },
);

const hasLogs = computed(() => logs.value.length > 0);

const tableScroll = computed(() => {
  if (!hasLogs.value || !tableScrollY.value) return { x: 1400 };
  return { x: 1400, y: tableScrollY.value };
});

const statusOptions = [
  { label: '全部状态', value: -1 },
  { label: '200', value: 200 },
  { label: '201', value: 201 },
  { label: '301', value: 301 },
  { label: '302', value: 302 },
  { label: '400', value: 400 },
  { label: '401', value: 401 },
  { label: '403', value: 403 },
  { label: '404', value: 404 },
  { label: '500', value: 500 },
  { label: '502', value: 502 },
  { label: '503', value: 503 },
];

const rawLogText = computed(() => {
  return normalizeLogText(selectedLog.value?.rawLog);
});

function normalizeLogText(value?: string) {
  if (!value) return '-';
  try {
    const parsed = JSON.parse(value);
    if (typeof parsed !== 'string') {
      return JSON.stringify(parsed, null, 2);
    }
    const trimmed = parsed.trim();
    if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) {
      return parsed;
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return parsed;
    }
  } catch {
    return value;
  }
}

/** 地区展示：优先 location，否则 country/province/city 拼接 */
function formatLocation(record: CaddyLog) {
  return (
    record.location ||
    [record.country, record.province, record.city].filter(Boolean).join(' ') ||
    '-'
  );
}

function methodTagColor(method?: string) {
  if (method === 'GET') return 'blue';
  if (method === 'POST') return 'green';
  return 'orange';
}

function statusTagColor(status?: number) {
  if (status == null) return 'default';
  if (status >= 200 && status < 300) return 'green';
  if (status >= 300 && status < 400) return 'orange';
  if (status >= 400) return 'red';
  return 'default';
}

const columns = computed(() => {
  const columnWidths: Partial<Record<ColumnWidthKey, number>> = hasLogs.value
    ? {
        actions: 100,
        clientIp: 150,
        host: 180,
        location: 180,
        logTime: 170,
        method: 90,
        status: 90,
        uri: 320,
      }
    : {};

  return [
    {
      dataIndex: 'logTime',
      defaultSortOrder: 'descend' as const,
      key: 'logTime',
      sorter: true,
      title: '时间',
      width: columnWidths.logTime,
    },
    {
      dataIndex: 'method',
      key: 'method',
      title: '方法',
      width: columnWidths.method,
    },
    {
      dataIndex: 'status',
      key: 'status',
      title: '状态',
      width: columnWidths.status,
    },
    { dataIndex: 'host', ellipsis: true, key: 'host', title: '域名', width: columnWidths.host },
    { dataIndex: 'uri', ellipsis: true, key: 'uri', title: '路径', width: columnWidths.uri },
    {
      dataIndex: 'clientIp',
      ellipsis: true,
      key: 'clientIp',
      title: '来源 IP',
      width: columnWidths.clientIp,
    },
    {
      dataIndex: 'location',
      ellipsis: true,
      key: 'location',
      title: '地区',
      width: columnWidths.location,
    },
    {
      fixed: hasLogs.value ? ('right' as const) : undefined,
      key: 'actions',
      title: '操作',
      width: columnWidths.actions,
    },
  ];
});

function syncHostQueryToRoute() {
  const nextHost = filters.host.trim();
  const current =
    typeof route.query.host === 'string'
      ? route.query.host
      : Array.isArray(route.query.host)
        ? String(route.query.host[0] ?? '')
        : '';
  if (nextHost === current) return;
  const query = { ...route.query };
  if (nextHost) {
    query.host = nextHost;
  } else {
    delete query.host;
  }
  router.replace({ query });
}

function handleSearch() {
  pagination.current = 1;
  syncHostQueryToRoute();
}

function handleReset() {
  filters.keyword = '';
  filters.host = '';
  filters.status = -1;
  filters.timeRange = undefined;
  pagination.current = 1;
  sortState.value = { order: 'descend' };
  syncHostQueryToRoute();
}

async function handleClearLogs() {
  if (clearing.value) return;
  clearing.value = true;
  try {
    await clearCaddyLogsApi();
    message.success('访问日志已清空');
    selectedLog.value = null;
    detailVisible.value = false;
    pagination.current = 1;
    await invalidateListDetailQueries(queryClient, ['caddy', 'logs']);
  } catch {
    message.error('清空访问日志失败');
  } finally {
    clearing.value = false;
  }
}

function handleTableChange(pag: any, _filters: any, sorter: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;

  const normalized = Array.isArray(sorter) ? sorter[0] : sorter;
  const order = normalized?.order;
  sortState.value = {
    order: order === 'ascend' || order === 'descend' ? order : false,
  };
  scheduleUpdateTableScrollY();
}

function openDetail(record: CaddyLog) {
  selectedLog.value = record;
  detailVisible.value = true;
}

/** 按表格容器实际高度计算 body 滚动区，避免下方大块空白 */
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
  () => route.query.host,
  () => {
    applyHostFromRoute();
    pagination.current = 1;
  },
  { immediate: true },
);

watch(
  () => [filters.host, loading.value, logs.value.length] as const,
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
    <div class="access-log-page">
      <Card title="Caddy 访问日志" variant="borderless" class="access-log-card">
        <Alert
          v-if="errorMessage"
          type="error"
          show-icon
          class="mb-3"
          :message="errorMessage"
        />
        <Form layout="inline" class="access-log-filter-form">
          <FormItem label="搜索">
            <Input.Search
              v-model:value="filters.keyword"
              placeholder="域名 / URI / IP"
              style="width: 240px"
              allow-clear
              @search="handleSearch"
            />
          </FormItem>
          <FormItem label="域名">
            <Input
              v-model:value="filters.host"
              placeholder="精确域名 host"
              style="width: 200px"
              allow-clear
              @press-enter="handleSearch"
            />
          </FormItem>
          <FormItem label="状态">
            <Select
              v-model:value="filters.status"
              :options="statusOptions"
              style="width: 130px"
              @change="handleSearch"
            />
          </FormItem>
          <FormItem label="时间">
            <DatePicker.RangePicker
              v-model:value="filters.timeRange"
              show-time
              style="width: 360px"
            />
          </FormItem>
          <FormItem>
            <Space>
              <Button type="primary" @click="handleSearch">搜索</Button>
              <Button @click="() => refetch()">刷新</Button>
              <Button @click="handleReset">重置</Button>
              <Popconfirm
                title="确认清空全部访问日志？此操作不可恢复。"
                @confirm="handleClearLogs"
              >
                <Button danger :loading="clearing">清空日志</Button>
              </Popconfirm>
            </Space>
          </FormItem>
        </Form>

        <div v-if="filters.host" class="mb-3 text-sm text-gray-500">
          当前按域名过滤：
          <Tag color="blue">{{ filters.host }}</Tag>
        </div>

        <div ref="tableWrapRef" class="access-log-table-wrap">
          <Table
            :columns="columns"
            :data-source="logs"
            :loading="loading"
            :pagination="pagination"
            :scroll="tableScroll"
            class="access-log-table"
            row-key="id"
            size="middle"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'method'">
                <Tag :color="methodTagColor((record as CaddyLog).method)">
                  {{ (record as CaddyLog).method || '-' }}
                </Tag>
              </template>
              <template v-else-if="column.key === 'status'">
                <Tag :color="statusTagColor((record as CaddyLog).status)">
                  {{ (record as CaddyLog).status ?? '-' }}
                </Tag>
              </template>
              <template v-else-if="column.key === 'location'">
                {{ formatLocation(record as CaddyLog) }}
              </template>
              <template v-else-if="column.key === 'actions'">
                <button
                  type="button"
                  class="table-action-btn"
                  @click="openDetail(record as CaddyLog)"
                >
                  详情
                </button>
              </template>
            </template>
          </Table>
        </div>
      </Card>

      <Modal v-model:open="detailVisible" title="日志详情" width="760px" :footer="null">
        <div class="max-h-[70vh] overflow-auto">
          <Descriptions v-if="selectedLog" bordered :column="1" size="small">
            <DescriptionsItem label="时间">{{ selectedLog.logTime }}</DescriptionsItem>
            <DescriptionsItem label="方法">{{ selectedLog.method }}</DescriptionsItem>
            <DescriptionsItem label="状态">{{ selectedLog.status }}</DescriptionsItem>
            <DescriptionsItem label="域名">{{ selectedLog.host }}</DescriptionsItem>
            <DescriptionsItem label="路径">{{ selectedLog.uri }}</DescriptionsItem>
            <DescriptionsItem label="大小">{{ selectedLog.size }}</DescriptionsItem>
            <DescriptionsItem label="远端 IP">{{ selectedLog.remoteIp }}</DescriptionsItem>
            <DescriptionsItem label="客户端 IP">{{ selectedLog.clientIp }}</DescriptionsItem>
            <DescriptionsItem label="地区">
              {{ formatLocation(selectedLog) }}
            </DescriptionsItem>
            <DescriptionsItem label="User Agent">
              {{ selectedLog.userAgent || '-' }}
            </DescriptionsItem>
            <DescriptionsItem label="原始日志">
              <pre class="log-detail-pre">{{ rawLogText }}</pre>
            </DescriptionsItem>
          </Descriptions>
        </div>
      </Modal>
    </div>
  </Page>
</template>

<style scoped>
.access-log-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.access-log-card {
 display: flex;
 flex: 1;
 flex-direction: column;
 height: 100%;
 min-height: 0;
}

.access-log-card :deep(.ant-card-body) {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.access-log-filter-form {
  row-gap: 12px;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.access-log-table-wrap {
 display: flex;
 flex: 1;
 flex-direction: column;
 min-width: 0;
 min-height: 0;
 overflow: hidden;
}

.access-log-table {
 flex: 1;
 height: 100%;
 min-height: 0;
}

.access-log-table :deep(.ant-spin-nested-loading),
.access-log-table :deep(.ant-spin-container) {
 height: 100%;
}

.access-log-table :deep(.ant-spin-container) {
 display: flex;
 flex-direction: column;
 height: 100%;
}

.access-log-table :deep(.ant-table) {
  flex: 1;
  min-height: 0;
}

.access-log-table :deep(.ant-table-container) {
 height: 100%;
}

.access-log-table :deep(.ant-table-pagination.ant-pagination) {
 flex-shrink: 0;
 margin: 12px 0 0;
}

.access-log-table :deep(.ant-table-cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-log-table :deep(.ant-tag) {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.log-detail-pre {
  max-height: 320px;
  margin: 0 0 8px;
  overflow: auto;
  padding: 10px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  background: #f5f5f5;
  border-radius: 6px;
}
</style>
