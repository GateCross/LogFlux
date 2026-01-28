<template>
  <div class="h-full overflow-hidden">
    <n-card title="Caddy 访问日志" :bordered="false" class="h-full rounded-8px shadow-sm">
      <div class="flex-col h-full">
        <n-space class="mb-4" justify="space-between">
          <n-input v-model:value="searchParams.keyword" placeholder="搜索 Host/URI/IP" clearable @keyup.enter="handleSearch">
            <template #prefix>
              <icon-ic-round-search class="text-16px" />
            </template>
          </n-input>
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
          </n-space>
        </n-space>

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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue';
import { NTag, useMessage } from 'naive-ui';
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
const searchParams = reactive({
  keyword: ''
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
  }
];

async function fetchData() {
  loading.value = true;
  try {
    const { data, error } = await fetchCaddyLogs({
      page: pagination.page || 1,
      pageSize: pagination.pageSize || 20,
      keyword: searchParams.keyword
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
