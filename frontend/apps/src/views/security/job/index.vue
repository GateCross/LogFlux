<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  message,
  Popconfirm,
  Space,
  Table,
  Tag,
} from 'ant-design-vue';

import {
  clearWafJobHistoryApi,
  getWafJobListApi,
} from '#/api/caddy/release';

defineOptions({ name: 'SecurityJob' });

const loading = ref(false);
const dataList = ref<any[]>([]);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '类型', dataIndex: 'type', key: 'type', width: 140 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 120 },
  { title: '消息', dataIndex: 'message', key: 'message', ellipsis: true },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
  { title: 'Finished At', dataIndex: 'finishedAt', key: 'finishedAt', width: 180 },
];

async function fetchData() {
  loading.value = true;
  try {
    dataList.value = await getWafJobListApi();
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false;
  }
}

async function handleClearHistory() {
  try {
    await clearWafJobHistoryApi();
    message.success('Job history cleared');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

onMounted(() => {
  fetchData();
});
</script>

<template>
  <Page title="WAF Jobs" description="View WAF background job history and clear completed jobs.">
    <Card>
      <template #extra>
        <Space>
          <Popconfirm
            title="Clear all job history? This cannot be undone."
            @confirm="handleClearHistory"
          >
            <Button danger>Clear History</Button>
          </Popconfirm>
          <Button @click="fetchData">Refresh</Button>
        </Space>
      </template>

      <Table
        :columns="columns"
        :data-source="dataList"
        :loading="loading"
        row-key="id"
        :scroll="{ x: 900 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <Tag
              :color="
                record.status === 'completed'
                  ? 'green'
                  : record.status === 'failed'
                    ? 'red'
                    : record.status === 'running'
                      ? 'blue'
                      : 'default'
              "
            >
              {{ record.status }}
            </Tag>
          </template>
        </template>
      </Table>
    </Card>
  </Page>
</template>
