<script lang="ts" setup>
import { onMounted, ref } from 'vue';

import {
  Card,
  Col,
  Row,
  Skeleton,
  Space,
  Statistic,
  Table,
  Tag,
} from 'ant-design-vue';

import { getDashboardSummaryApi } from '#/api/dashboard';

defineOptions({ name: 'Dashboard' });

const loading = ref(true);
const summary = ref<Record<string, any>>({});

onMounted(async () => {
  try {
    summary.value = await getDashboardSummaryApi();
  } finally {
    loading.value = false;
  }
});

/** Recent task status columns for the bottom summary table */
const recentColumns = [
  { dataIndex: 'taskName', key: 'taskName', title: 'Task' },
  { dataIndex: 'status', key: 'status', title: 'Status' },
  { dataIndex: 'duration', key: 'duration', title: 'Duration' },
  { dataIndex: 'triggerType', key: 'triggerType', title: 'Trigger' },
  { dataIndex: 'startTime', key: 'startTime', title: 'Started' },
];
</script>

<template>
  <div class="p-5">
    <Skeleton :loading="loading" active>
      <!-- ── Statistic cards ────────────────────────────────── -->
      <Row :gutter="[16, 16]">
        <Col :xs="24" :sm="12" :lg="6">
          <Card>
            <Statistic
              title="Total Requests"
              :value="summary.totalRequests ?? 0"
              :value-style="{ color: '#1890ff' }"
            />
          </Card>
        </Col>
        <Col :xs="24" :sm="12" :lg="6">
          <Card>
            <Statistic
              title="Error Rate"
              :value="summary.errorRate ?? 0"
              :precision="2"
              suffix="%"
              :value-style="{ color: summary.errorRate > 5 ? '#cf1322' : '#3f8600' }"
            />
          </Card>
        </Col>
        <Col :xs="24" :sm="12" :lg="6">
          <Card>
            <Statistic
              title="Active Tasks"
              :value="summary.activeTasks ?? 0"
            />
          </Card>
        </Col>
        <Col :xs="24" :sm="12" :lg="6">
          <Card>
            <Statistic
              title="Avg Duration (ms)"
              :value="summary.avgDuration ?? 0"
              :precision="0"
            />
          </Card>
        </Col>
      </Row>

      <!-- ── Chart placeholders ─────────────────────────────── -->
      <Row :gutter="[16, 16]" class="mt-4">
        <Col :xs="24" :lg="16">
          <Card title="Request Trend">
            <div
              class="flex h-[320px] items-center justify-center rounded border border-dashed border-gray-300 text-gray-400"
            >
              [ ECharts Trend Chart Placeholder ]
            </div>
          </Card>
        </Col>
        <Col :xs="24" :lg="8">
          <Card title="Status Code Distribution">
            <div
              class="flex h-[320px] items-center justify-center rounded border border-dashed border-gray-300 text-gray-400"
            >
              [ ECharts Pie Chart Placeholder ]
            </div>
          </Card>
        </Col>
      </Row>

      <!-- ── Recent executions table ────────────────────────── -->
      <Card title="Recent Executions" class="mt-4">
        <Table
          :columns="recentColumns"
          :data-source="summary.recentLogs ?? []"
          :pagination="false"
          row-key="id"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <Tag
                :color="
                  record.status === 'success'
                    ? 'green'
                    : record.status === 'failed'
                      ? 'red'
                      : 'blue'
                "
              >
                {{ record.status }}
              </Tag>
            </template>
            <template v-if="column.key === 'triggerType'">
              <Tag :color="record.triggerType === 'auto' ? 'purple' : 'orange'">
                {{ record.triggerType }}
              </Tag>
            </template>
          </template>
        </Table>
      </Card>
    </Skeleton>
  </div>
</template>
