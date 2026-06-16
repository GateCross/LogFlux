<script lang="ts" setup>
import type { CaddyServerApi } from '#/api/caddy/server';

import { computed, h, onMounted, reactive, ref } from 'vue';

import {
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
} from 'ant-design-vue';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import { clearCaddyLogsApi, getCaddyLogsApi } from '#/api/caddy/server';

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

const loading = ref(false);
const clearing = ref(false);
const logs = ref<CaddyLog[]>([]);
const selectedLog = ref<CaddyLog | null>(null);
const detailVisible = ref(false);
const sortState = ref<{ order: SortOrder }>({ order: 'descend' });

const filters = reactive({
  keyword: '',
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

const hasLogs = computed(() => logs.value.length > 0);

const tableScroll = computed(() => {
  if (!hasLogs.value) return undefined;
  return { x: 1400 };
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

const columns = computed(() => {
  const columnWidths: Partial<Record<ColumnWidthKey, number>> = hasLogs.value
    ? {
        actions: 90,
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
      customRender: ({ text }: { text: string }) => {
        const color = text === 'GET' ? 'blue' : text === 'POST' ? 'green' : 'orange';
        return h(Tag, { color }, () => text || '-');
      },
      dataIndex: 'method',
      key: 'method',
      title: '方法',
      width: columnWidths.method,
    },
    {
      customRender: ({ text }: { text: number }) => {
        let color = 'default';
        if (text >= 200 && text < 300) color = 'green';
        else if (text >= 300 && text < 400) color = 'orange';
        else if (text >= 400) color = 'red';
        return h(Tag, { color }, () => String(text ?? '-'));
      },
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
      customRender: ({ record }: { record: CaddyLog }) =>
        record.location ||
        [record.country, record.province, record.city].filter(Boolean).join(' ') ||
        '-',
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

function formatRangeTime(value: Dayjs | undefined) {
  if (!value) return undefined;
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss');
}

async function fetchLogs() {
  loading.value = true;
  try {
    const [start, end] = filters.timeRange || [];
    const data = await getCaddyLogsApi({
      page: pagination.current,
      pageSize: pagination.pageSize,
      keyword: filters.keyword,
      status: filters.status,
      startTime: formatRangeTime(start),
      endTime: formatRangeTime(end),
      sortBy: sortState.value.order ? 'logTime' : undefined,
      order:
        sortState.value.order === 'ascend'
          ? 'asc'
          : sortState.value.order === 'descend'
            ? 'desc'
            : undefined,
    });

    logs.value = data.list ?? [];
    pagination.total = data.total ?? 0;
  } catch {
    message.error('获取访问日志失败');
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.current = 1;
  fetchLogs();
}

function handleReset() {
  filters.keyword = '';
  filters.status = -1;
  filters.timeRange = undefined;
  pagination.current = 1;
  sortState.value = { order: 'descend' };
  fetchLogs();
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
    logs.value = [];
    pagination.total = 0;
    await fetchLogs();
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
  fetchLogs();
}

function openDetail(record: CaddyLog) {
  selectedLog.value = record;
  detailVisible.value = true;
}

onMounted(() => {
  fetchLogs();
});
</script>

<template>
  <div class="h-full overflow-x-hidden overflow-y-auto p-4">
    <Card title="Caddy 访问日志" :bordered="false" class="access-log-card h-full rounded-lg shadow-sm">
      <div class="flex h-full min-h-0 flex-col">
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
              <Button @click="fetchLogs">刷新</Button>
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

        <div class="access-log-table-wrap">
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
              <template v-if="column.key === 'actions'">
                <Button type="link" size="small" @click="openDetail(record as CaddyLog)">
                  详情
                </Button>
              </template>
            </template>
          </Table>
        </div>
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
            {{
              selectedLog.location ||
              [selectedLog.country, selectedLog.province, selectedLog.city].filter(Boolean).join(' ') ||
              '-'
            }}
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
</template>

<style scoped>
.access-log-filter-form {
  row-gap: 12px;
  margin-bottom: 24px;
}

.access-log-card :deep(.ant-card-body) {
  min-width: 0;
}

.access-log-table-wrap {
  min-width: 0;
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
