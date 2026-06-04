<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, Card, Popconfirm, Space, Table, Tag, message } from 'ant-design-vue';

import {
  clearWafJobHistoryApi,
  clearWafReleaseHistoryApi,
  getWafJobListApi,
  getWafReleaseListApi,
  rollbackWafReleaseApi,
} from '#/api/caddy/release';

defineOptions({ name: 'SecurityOps' });

const loading = ref(false);
const releases = ref<any[]>([]);
const jobs = ref<any[]>([]);

const releaseColumns = [
  { dataIndex: 'version', key: 'version', title: '版本' },
  { dataIndex: 'kind', key: 'kind', title: '类型', width: 120 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 120 },
  { dataIndex: 'updatedAt', key: 'updatedAt', title: '更新时间' },
];

const jobColumns = [
  { dataIndex: 'action', key: 'action', title: '动作' },
  { dataIndex: 'triggerMode', key: 'triggerMode', title: '触发方式', width: 120 },
  { dataIndex: 'status', key: 'status', title: '状态', width: 120 },
  { dataIndex: 'message', key: 'message', title: '消息', ellipsis: true },
  { dataIndex: 'createdAt', key: 'createdAt', title: '创建时间' },
];

async function fetchData() {
  loading.value = true;
  try {
    const [releaseList, jobList] = await Promise.all([
      getWafReleaseListApi(),
      getWafJobListApi(),
    ]);
    releases.value = releaseList;
    jobs.value = jobList;
  } finally {
    loading.value = false;
  }
}

async function handleRollback() {
  await rollbackWafReleaseApi();
  message.success('已触发 WAF 发布回滚');
  await fetchData();
}

async function handleClearReleases() {
  await clearWafReleaseHistoryApi();
  message.success('发布历史已清理');
  await fetchData();
}

async function handleClearJobs() {
  await clearWafJobHistoryApi();
  message.success('任务历史已清理');
  await fetchData();
}

onMounted(fetchData);
</script>

<template>
  <Page title="WAF 运维" description="管理 WAF 发布、回滚和后台任务。">
    <Space direction="vertical" size="large" style="width: 100%;">
      <Card title="发布操作">
        <template #extra>
          <Space>
            <Button :loading="loading" @click="fetchData">刷新</Button>
            <Popconfirm title="确认回滚到上一可用版本？" @confirm="handleRollback">
              <Button>回滚</Button>
            </Popconfirm>
            <Popconfirm title="确认清理发布历史？" @confirm="handleClearReleases">
              <Button danger>清理发布历史</Button>
            </Popconfirm>
          </Space>
        </template>
        <Table :columns="releaseColumns" :data-source="releases" :loading="loading" row-key="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <Tag :color="record.status === 'active' ? 'green' : record.status === 'failed' ? 'red' : 'blue'">
                {{ record.status }}
              </Tag>
            </template>
          </template>
        </Table>
      </Card>

      <Card title="后台任务">
        <template #extra>
          <Popconfirm title="确认清理任务历史？" @confirm="handleClearJobs">
            <Button danger>清理任务历史</Button>
          </Popconfirm>
        </template>
        <Table :columns="jobColumns" :data-source="jobs" :loading="loading" row-key="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <Tag :color="record.status === 'success' || record.status === 'completed' ? 'green' : record.status === 'failed' ? 'red' : 'blue'">
                {{ record.status }}
              </Tag>
            </template>
          </template>
        </Table>
      </Card>
    </Space>
  </Page>
</template>
