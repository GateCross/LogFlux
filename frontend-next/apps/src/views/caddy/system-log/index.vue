<script lang="ts" setup>
import { h, onMounted, reactive, ref } from 'vue';
import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  Form,
  FormItem,
  InputSearch,
  Select,
  SelectOption,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import { getSystemLogsApi } from '#/api/system/log';
import type { SystemLogApi } from '#/api/system/log';

// --------------- state ---------------
const loading = ref(false);
const dataSource = ref<SystemLogApi.LogItem[]>([]);
const total = ref(0);

const pagination = reactive({
  current: 1,
  pageSize: 20,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50', '100'],
  showTotal: (t: number) => `Total ${t} records`,
});

const filters = reactive({
  keyword: '',
  level: undefined as string | undefined,
});

// --------------- columns ---------------
const columns = [
  {
    title: 'Timestamp',
    dataIndex: 'timestamp',
    key: 'timestamp',
    width: 180,
    sorter: true,
  },
  {
    title: 'Level',
    dataIndex: 'level',
    key: 'level',
    width: 100,
    customRender: ({ text }: { text: string }) => {
      const colorMap: Record<string, string> = {
        error: 'red',
        warn: 'orange',
        warning: 'orange',
        info: 'blue',
        debug: 'default',
      };
      return h(Tag, { color: colorMap[String(text).toLowerCase()] ?? 'default' }, () =>
        String(text).toUpperCase(),
      );
    },
  },
  {
    title: 'Source',
    dataIndex: 'source',
    key: 'source',
    width: 150,
  },
  {
    title: 'Message',
    dataIndex: 'message',
    key: 'message',
    ellipsis: true,
  },
  {
    title: 'Details',
    dataIndex: 'details',
    key: 'details',
    ellipsis: true,
  },
];

const levelOptions = ['INFO', 'WARN', 'ERROR', 'DEBUG'];

// --------------- data fetching ---------------
async function fetchLogs() {
  loading.value = true;
  try {
    const params: Record<string, any> = {
      page: pagination.current,
      pageSize: pagination.pageSize,
    };
    if (filters.keyword) params.keyword = filters.keyword;
    if (filters.level) params.level = filters.level;

    const res = await getSystemLogsApi(params);
    // API may return array directly or paginated object
    dataSource.value = res.list ?? [];
    total.value = res.total ?? 0;
  } catch {
    // error handled by request interceptor
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
  filters.level = undefined;
  pagination.current = 1;
  fetchLogs();
}

function handleTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  fetchLogs();
}

// --------------- lifecycle ---------------
onMounted(() => {
  fetchLogs();
});
</script>

<template>
  <Page description="Browse system-level logs" title="System Logs">
    <Card>
      <!-- Filters -->
      <Form layout="inline" style="margin-bottom: 16px;">
        <FormItem label="Search">
          <InputSearch
            v-model:value="filters.keyword"
            placeholder="Search messages..."
            style="width: 260px;"
            @search="handleSearch"
          />
        </FormItem>
        <FormItem label="Level">
          <Select
            v-model:value="filters.level"
            placeholder="All"
            allow-clear
            style="width: 120px;"
            @change="handleSearch"
          >
            <SelectOption v-for="l in levelOptions" :key="l" :value="l">
              {{ l }}
            </SelectOption>
          </Select>
        </FormItem>
        <FormItem>
          <Space>
            <Button type="primary" @click="handleSearch">Search</Button>
            <Button @click="handleReset">Reset</Button>
          </Space>
        </FormItem>
      </Form>

      <!-- Table -->
      <Table
        :columns="columns"
        :data-source="dataSource"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      />
    </Card>
  </Page>
</template>
