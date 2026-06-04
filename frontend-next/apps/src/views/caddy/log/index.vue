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
  Select,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import { getCaddyLogsApi } from '#/api/caddy/server';

defineOptions({ name: 'CaddyAccessLog' });

type CaddyLog = CaddyServerApi.CaddyLogItem;
type SortOrder = 'ascend' | 'descend' | false;

const loading = ref(false);
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
  const rawLog = selectedLog.value?.rawLog;
  if (!rawLog) return '-';
  try {
    const parsed = JSON.parse(rawLog);
    return typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2);
  } catch {
    return rawLog;
  }
});

const columns = [
  {
    dataIndex: 'logTime',
    defaultSortOrder: 'descend' as const,
    key: 'logTime',
    sorter: true,
    title: '时间',
    width: 170,
  },
  {
    customRender: ({ text }: { text: string }) => {
      const color = text === 'GET' ? 'blue' : text === 'POST' ? 'green' : 'orange';
      return h(Tag, { color }, () => text || '-');
    },
    dataIndex: 'method',
    key: 'method',
    title: '方法',
    width: 90,
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
    width: 90,
  },
  { dataIndex: 'host', ellipsis: true, key: 'host', title: '域名', width: 180 },
  { dataIndex: 'uri', ellipsis: true, key: 'uri', title: '路径', width: 280 },
  {
    dataIndex: 'clientIp',
    ellipsis: true,
    key: 'clientIp',
    title: '来源 IP',
    width: 150,
  },
  {
    customRender: ({ record }: { record: CaddyLog }) =>
      record.location ||
      [record.country, record.province, record.city].filter(Boolean).join(' ') ||
      '-',
    key: 'location',
    title: '地区',
    width: 180,
  },
  { fixed: 'right' as const, key: 'actions', title: '操作', width: 90 },
];

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
  <div class="h-full overflow-hidden p-4">
    <Card title="Caddy 访问日志" :bordered="false" class="h-full rounded-lg shadow-sm">
      <div class="flex h-full min-h-0 flex-col">
        <Form layout="inline" class="mb-4">
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
            </Space>
          </FormItem>
        </Form>

        <Table
          :columns="columns"
          :data-source="logs"
          :loading="loading"
          :pagination="pagination"
          :scroll="{ x: 1200, y: 'calc(100vh - 300px)' }"
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
            <Input.TextArea :value="rawLogText" readonly :auto-size="{ minRows: 4, maxRows: 12 }" />
          </DescriptionsItem>
        </Descriptions>
      </div>
    </Modal>
  </div>
</template>
