<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';

import { Button, Card, Space, Statistic, Table, Tag } from 'ant-design-vue';

import {
  getFalsePositiveFeedbackListApi,
  getWafPolicyStatsApi,
} from '#/api/caddy/observe';

defineOptions({ name: 'SecurityObserve' });

const statsLoading = ref(false);
const feedbackLoading = ref(false);
const statsData = ref<any>({});
const feedbackList = ref<any[]>([]);

const statsColumns = [
  { title: 'Metric', dataIndex: 'metric', key: 'metric' },
  { title: 'Value', dataIndex: 'value', key: 'value' },
];

const feedbackColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: 'Rule ID', dataIndex: 'ruleId', key: 'ruleId', width: 150 },
  { title: 'URI', dataIndex: 'uri', key: 'uri', ellipsis: true },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 120 },
  { title: 'Created At', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
  { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
];

async function fetchStats() {
  statsLoading.value = true;
  try {
    const res = await getWafPolicyStatsApi();
    statsData.value = Array.isArray(res) ? res : (res?.data ?? res ?? {});
  } catch {
    // error handled by interceptor
  } finally {
    statsLoading.value = false;
  }
}

async function fetchFeedback() {
  feedbackLoading.value = true;
  try {
    const res = await getFalsePositiveFeedbackListApi();
    feedbackList.value = Array.isArray(res) ? res : (res?.data ?? []);
  } catch {
    // error handled by interceptor
  } finally {
    feedbackLoading.value = false;
  }
}

function handleRefresh() {
  fetchStats();
  fetchFeedback();
}

onMounted(() => {
  fetchStats();
  fetchFeedback();
});
</script>

<template>
  <Page title="WAF Observe" description="Monitor WAF policy statistics and review false positive feedback.">
    <Card title="Policy Statistics" class="mb-4">
      <template #extra>
        <Button @click="fetchStats">Refresh</Button>
      </template>
      <Table
        :columns="statsColumns"
        :data-source="Array.isArray(statsData) ? statsData : Object.entries(statsData).map(([metric, value]) => ({ metric, value }))"
        :loading="statsLoading"
        row-key="metric"
        :pagination="false"
      />
    </Card>

    <Card title="False Positive Feedback">
      <template #extra>
        <Space>
          <Button @click="fetchFeedback">Refresh</Button>
        </Space>
      </template>

      <Table
        :columns="feedbackColumns"
        :data-source="feedbackList"
        :loading="feedbackLoading"
        row-key="id"
        :scroll="{ x: 900 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <Tag
              :color="
                record.status === 'resolved'
                  ? 'green'
                  : record.status === 'pending'
                    ? 'orange'
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
