<template>
  <div class="h-full overflow-hidden">
    <n-card title="Caddy 访问日志" :bordered="false" class="h-full rounded-8px shadow-sm">
      <div class="flex-col h-full">
        <div class="mb-4 flex flex-wrap items-end gap-3">
          <n-input
            v-model:value="searchParams.keyword"
            placeholder="搜索 Host/URI/IP"
            clearable
            class="w-56"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <icon-ic-round-search class="text-16px" />
            </template>
          </n-input>
          <n-input
            v-model:value="searchParams.host"
            placeholder="域名过滤"
            clearable
            class="w-44"
            @keyup.enter="handleSearch"
          />
          <n-select
            v-model:value="searchParams.status"
            :options="statusOptions"
            class="w-36"
          />
          <n-date-picker
            v-model:value="searchParams.timeRange"
            type="datetimerange"
            value-format="yyyy-MM-dd HH:mm:ss"
            clearable
            class="w-72"
          />
          <n-space>
            <n-button type="primary" @click="handleSearch">
              <template #icon>
                <icon-ic-round-search />
              </template>
              搜索
            </n-button>
            <n-button @click="handleRefresh">
              <template #icon>
                <icon-ic-round-refresh />
              </template>
              刷新
            </n-button>
            <n-button tertiary @click="handleReset">重置</n-button>
          </n-space>
        </div>

        <n-data-table
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :row-key="row => row.id"
          class="flex-1-hidden"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
          size="small"
        />
      </div>
    </n-card>

    <n-drawer v-model:show="showDetail" width="520">
      <n-drawer-content title="日志详情">
        <n-descriptions bordered size="small" :column="1" v-if="selectedLog">
          <n-descriptions-item label="时间">{{ selectedLog.logTime }}</n-descriptions-item>
          <n-descriptions-item label="方法">{{ selectedLog.method }}</n-descriptions-item>
          <n-descriptions-item label="状态">{{ selectedLog.status }}</n-descriptions-item>
          <n-descriptions-item label="域名">{{ selectedLog.host }}</n-descriptions-item>
          <n-descriptions-item label="路径">{{ selectedLog.uri }}</n-descriptions-item>
          <n-descriptions-item label="大小">{{ selectedLog.size }}</n-descriptions-item>
          <n-descriptions-item label="远端 IP">{{ selectedLog.remoteIp }}</n-descriptions-item>
          <n-descriptions-item label="客户端 IP">{{ selectedLog.clientIp }}</n-descriptions-item>
          <n-descriptions-item label="地区">{{ selectedLog.country }} {{ selectedLog.city }}</n-descriptions-item>
          <n-descriptions-item label="User Agent">{{ selectedLog.userAgent || '-' }}</n-descriptions-item>
        </n-descriptions>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue';
import { NTag, NButton, useMessage } from 'naive-ui';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { fetchCaddyLogs } from '@/service/api/caddy';

interface CaddyLog {
  id: number;
  logTime: string;
  country: string;
  city: string;
  host: string;
  method: string;
  uri: string;
  status: number;
  size: number;
  remoteIp: string;
  clientIp: string;
  userAgent: string;
}

const message = useMessage();
const loading = ref(false);
const tableData = ref<CaddyLog[]>([]);
const selectedLog = ref<CaddyLog | null>(null);
const showDetail = ref(false);
const searchParams = reactive({
  keyword: '',
  host: '',
  status: -1,
  timeRange: null as string[] | null
});

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  itemCount: 0,
  onChange: (page: number) => {
    pagination.page = page;
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize;
    pagination.page = 1;
  }
});

const columns: DataTableColumns<CaddyLog> = [
  {
    title: '时间',
    key: 'logTime',
    width: 160,
    ellipsis: { tooltip: true }
  },
  {
    title: '方法',
    key: 'method',
    width: 80,
    render(row) {
      const type = row.method === 'GET' ? 'info' : row.method === 'POST' ? 'success' : 'warning';
      return h(NTag, { type, size: 'small' }, { default: () => row.method });
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render(row) {
      let type: 'default' | 'success' | 'warning' | 'error' = 'default';
      if (row.status >= 200 && row.status < 300) type = 'success';
      else if (row.status >= 300 && row.status < 400) type = 'warning';
      else if (row.status >= 400) type = 'error';
      return h(NTag, { type, size: 'small' }, { default: () => row.status });
    }
  },
  {
    title: '域名',
    key: 'host',
    width: 150,
    ellipsis: { tooltip: true }
  },
  {
    title: '路径',
    key: 'uri',
    minWidth: 200,
    ellipsis: { tooltip: true }
  },
  {
    title: '来源IP',
    key: 'clientIp',
    width: 130,
    ellipsis: { tooltip: true }
  },
  {
    title: '地区',
    key: 'location',
    width: 150,
    render(row) {
      if (!row.country && !row.city) return '-';
      return `${row.country || ''} ${row.city || ''}`;
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 90,
    render(row) {
      return h(
        NButton,
        {
          size: 'tiny',
          onClick: () => openDetail(row)
        },
        { default: () => '详情' }
      );
    }
  }
];

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
  { label: '503', value: 503 }
];

async function fetchData() {
  loading.value = true;
  try {
    const [startTime, endTime] = searchParams.timeRange || [];
    const { data, error } = await fetchCaddyLogs({
      page: pagination.page || 1,
      pageSize: pagination.pageSize || 20,
      keyword: searchParams.keyword,
      host: searchParams.host,
      status: searchParams.status,
      startTime,
      endTime
    });

    if (error) {
      message.error('获取日志失败');
      return;
    }

    if (data) {
      tableData.value = data.list || [];
      pagination.itemCount = data.total || 0;
    }
  } catch (err) {
    console.error(err);
    message.error('系统错误');
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.page = 1;
  fetchData();
}

function handleRefresh() {
  fetchData();
}

function handleReset() {
  searchParams.keyword = '';
  searchParams.host = '';
  searchParams.status = -1;
  searchParams.timeRange = null;
  pagination.page = 1;
  fetchData();
}

function openDetail(row: CaddyLog) {
  selectedLog.value = row;
  showDetail.value = true;
}

function handlePageChange(page: number) {
  pagination.page = page;
  fetchData();
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize;
  pagination.page = 1;
  fetchData();
}

onMounted(() => {
  fetchData();
});
</script>

<style scoped></style>
