<script lang="ts" setup>
import { h, onMounted, reactive, ref } from 'vue';
import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  Form,
  FormItem,
  Input,
  InputSearch,
  Select,
  SelectOption,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import { getCaddyLogsApi } from '#/api/caddy/server';
import type { CaddyServerApi } from '#/api/caddy/server';

// --------------- state ---------------
const loading = ref(false);
const dataSource = ref<CaddyServerApi.CaddyLogItem[]>([]);
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
  method: undefined as string | undefined,
  status: undefined as string | undefined,
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
    title: 'Method',
    dataIndex: 'method',
    key: 'method',
    width: 90,
  },
  {
    title: 'URI',
    dataIndex: 'uri',
    key: 'uri',
    ellipsis: true,
  },
  {
    title: 'Status',
    dataIndex: 'status',
    key: 'status',
    width: 80,
    customRender: ({ text }: { text: number }) => {
      let color = 'default';
      if (text >= 200 && text < 300) color = 'green';
      else if (text >= 300 && text < 400) color = 'blue';
      else if (text >= 400 && text < 500) color = 'orange';
      else if (text >= 500) color = 'red';
      return h(Tag, { color }, () => text);
    },
  },
  {
    title: 'Duration',
    dataIndex: 'duration',
    key: 'duration',
    width: 100,
    customRender: ({ text }: { text: number }) => {
      if (text == null) return '-';
      return text >= 1000 ? `${(text / 1000).toFixed(2)}s` : `${text}ms`;
    },
  },
  {
    title: 'Remote IP',
    dataIndex: 'remoteIP',
    key: 'remoteIP',
    width: 140,
  },
];

const methodOptions = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];
const statusOptions = ['200-299', '300-399', '400-499', '500+'];

// --------------- data fetching ---------------
async function fetchLogs() {
  loading.value = true;
  try {
    const params: Record<string, any> = {
      page: pagination.current,
      pageSize: pagination.pageSize,
    };
    if (filters.keyword) params.keyword = filters.keyword;
    if (filters.method) params.method = filters.method;
    if (filters.status) params.status = filters.status;

    const res = await getCaddyLogsApi(params);
    dataSource.value = res.items ?? [];
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
  filters.method = undefined;
  filters.status = undefined;
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
  <Page description="Browse Caddy HTTP access logs" title="Access Logs">
    <Card>
      <!-- Filters -->
      <Form layout="inline" style="margin-bottom: 16px;">
        <FormItem label="Search">
          <InputSearch
            v-model:value="filters.keyword"
            placeholder="Search by URI, IP..."
            style="width: 240px;"
            @search="handleSearch"
          />
        </FormItem>
        <FormItem label="Method">
          <Select
            v-model:value="filters.method"
            placeholder="All"
            allow-clear
            style="width: 120px;"
            @change="handleSearch"
          >
            <SelectOption v-for="m in methodOptions" :key="m" :value="m">
              {{ m }}
            </SelectOption>
          </Select>
        </FormItem>
        <FormItem label="Status">
          <Select
            v-model:value="filters.status"
            placeholder="All"
            allow-clear
            style="width: 120px;"
            @change="handleSearch"
          >
            <SelectOption v-for="s in statusOptions" :key="s" :value="s">
              {{ s }}
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
