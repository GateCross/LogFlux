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
  activateWafReleaseApi,
  clearWafReleaseHistoryApi,
  getWafReleaseListApi,
  rollbackWafReleaseApi,
} from '#/api/caddy/release';

defineOptions({ name: 'SecurityRelease' });

const loading = ref(false);
const dataList = ref<any[]>([]);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: 'Version', dataIndex: 'version', key: 'version', width: 150 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 120 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
  { title: '操作', key: 'actions', width: 240, fixed: 'right' as const },
];

async function fetchData() {
  loading.value = true;
  try {
    dataList.value = await getWafReleaseListApi();
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false;
  }
}

async function handleActivate(id: number) {
  try {
    await activateWafReleaseApi(id);
    message.success('Release activated');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handleRollback() {
  try {
    await rollbackWafReleaseApi();
    message.success('Rollback completed');
    await fetchData();
  } catch {
    // error handled by interceptor
  }
}

async function handleClearHistory() {
  try {
    await clearWafReleaseHistoryApi();
    message.success('Release history cleared');
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
  <Page title="WAF Releases" description="Manage WAF release versions. Activate, rollback, and clear release history.">
    <Card>
      <template #extra>
        <Space>
          <Popconfirm
            title="Rollback to the previous release?"
            @confirm="handleRollback"
          >
            <Button>Rollback</Button>
          </Popconfirm>
          <Popconfirm
            title="Clear all release history? This cannot be undone."
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
                record.status === 'active'
                  ? 'green'
                  : record.status === 'archived'
                    ? 'default'
                    : 'blue'
              "
            >
              {{ record.status }}
            </Tag>
          </template>
          <template v-if="column.key === 'actions'">
            <Space>
              <Popconfirm
                title="Activate this release?"
                @confirm="handleActivate(record.id)"
              >
                <Button size="small" type="link">Activate</Button>
              </Popconfirm>
            </Space>
          </template>
        </template>
      </Table>
    </Card>
  </Page>
</template>
