<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Input,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antdv-next';
import {
  getWafEventListApi,
  getWafEventStatsApi,
} from '#/api/caddy/waf-event';
import type { CaddyWafEventApi } from '#/api/caddy/waf-event';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import StatCard from '#/views/dashboard/modules/stat-card.vue';
import MapChart from '#/views/dashboard/modules/map-chart.vue';
import type { StatCard as StatCardItem } from '#/views/dashboard/data';
import WafTrendChart from './modules/waf-trend-chart.vue';

defineOptions({ name: 'LogsSecurity' });

const timeRanges = [
  { key: '1h', label: '1小时', hours: 1 },
  { key: '12h', label: '12小时', hours: 12 },
  { key: '1d', label: '1天', hours: 24 },
  { key: '3d', label: '3天', hours: 72 },
  { key: '7d', label: '一周', hours: 168 },
];
const intervalOptions = [
  { key: '1m', label: '1分钟', seconds: 60 },
  { key: '5m', label: '5分钟', seconds: 300 },
  { key: '30m', label: '30分钟', seconds: 1800 },
];
const activeRangeKey = ref<string>(
  localStorage.getItem('logflux:security.range') || timeRanges[2]!.key,
);
const activeIntervalKey = ref<string>(
  localStorage.getItem('logflux:security.interval') || intervalOptions[1]!.key,
);

const activeRange = computed(
  () => timeRanges.find((item) => item.key === activeRangeKey.value) ?? timeRanges[2]!,
);
const activeInterval = computed(
  () =>
    intervalOptions.find((item) => item.key === activeIntervalKey.value) ??
    intervalOptions[1]!,
);

function formatDateTime(value: Date) {
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())} ${pad(value.getHours())}:${pad(value.getMinutes())}:${pad(value.getSeconds())}`;
}

const statsParams = computed(() => {
  const now = new Date();
  const start = new Date(now.getTime() - activeRange.value.hours * 3600 * 1000);
  return {
    startTime: formatDateTime(start),
    endTime: formatDateTime(now),
    intervalSec: activeInterval.value.seconds,
    topN: 8,
  };
});

const {
  data: statsData,
  errorMessage: statsError,
  refetch: refetchStats,
} = useListDetailQuery({
  queryKey: computed(() => qk.caddy.wafEventStats(statsParams.value)),
  queryFn: () => getWafEventStatsApi(statsParams.value, withListDetailErrorMode()),
  errorFallback: '加载安全统计失败',
});

const stats = computed(() => statsData.value ?? null);

const detected = computed(() => (stats.value?.total ?? 0) - (stats.value?.blocked ?? 0));

const statCards = computed<StatCardItem[]>(() => [
  {
    id: 'total',
    title: 'WAF 命中',
    value: stats.value?.total ?? 0,
    icon: 'mdi:target',
    color: '#f59e0b',
  },
  {
    id: 'blocked',
    title: '拦截请求',
    value: stats.value?.blocked ?? 0,
    icon: 'mdi:shield-lock-outline',
    color: '#ef4444',
  },
  {
    id: 'detect',
    title: '仅检测',
    value: detected.value < 0 ? 0 : detected.value,
    icon: 'mdi:eye-outline',
    color: '#3b82f6',
  },
]);

const trendTimes = computed(() => stats.value?.trend?.map((item) => item.time) ?? []);
const trendTotals = computed(() => stats.value?.trend?.map((item) => item.total) ?? []);
const trendBlockeds = computed(() => stats.value?.trend?.map((item) => item.blocked) ?? []);
const geoWorldData = computed(() => stats.value?.countries ?? []);

const topBoards = computed(() => [
  { title: 'Top 攻击规则', items: stats.value?.topRules ?? [] },
  { title: 'Top 攻击 IP', items: stats.value?.topIps ?? [] },
  { title: 'Top 防护站点', items: stats.value?.topHosts ?? [] },
]);

/** CRS 严重度展示：数字 1-5 映射为可读标签 */
const severityMeta: Record<string, { label: string; color: string }> = {
  '1': { label: 'S1', color: 'blue' },
  '2': { label: 'S2', color: 'cyan' },
  '3': { label: 'S3', color: 'orange' },
  '4': { label: 'S4', color: 'red' },
  '5': { label: 'S5', color: 'volcano' },
};

function severityTag(value: string) {
  const upper = value.trim().toUpperCase();
  if (severityMeta[value.trim()]) {
    return severityMeta[value.trim()]!;
  }
  if (upper === 'CRITICAL') return { label: '严重', color: 'volcano' };
  if (upper === 'ERROR') return { label: '错误', color: 'red' };
  if (upper === 'WARNING') return { label: '警告', color: 'orange' };
  if (upper === 'NOTICE') return { label: '提示', color: 'blue' };
  return { label: value || '-', color: 'default' };
}

// ---------- 事件列表 ----------

const filters = reactive({
  action: 'all' as 'all' | 'block' | 'detect',
  ruleId: '',
  clientIp: '',
  keyword: '',
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

function formatRangeTime(value: Dayjs | undefined) {
  return value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : undefined;
}

const listParams = computed<CaddyWafEventApi.WafEventListParams>(() => {
  const [start, end] = filters.timeRange || [];
  return {
    page: pagination.current,
    pageSize: pagination.pageSize,
    action: filters.action === 'all' ? undefined : filters.action,
    ruleId: filters.ruleId.trim() || undefined,
    clientIp: filters.clientIp.trim() || undefined,
    keyword: filters.keyword.trim() || undefined,
    startTime: formatRangeTime(start),
    endTime: formatRangeTime(end),
  };
});

const {
  data: eventsPage,
  loading,
  errorMessage: listError,
  refetch: refetchList,
} = useListDetailQuery({
  queryKey: computed(() => qk.caddy.wafEvents(listParams.value)),
  queryFn: () => getWafEventListApi(listParams.value, withListDetailErrorMode()),
  errorFallback: '获取安全事件失败',
});

const events = computed(() => eventsPage.value?.list ?? []);

watch(
  eventsPage,
  (page) => {
    pagination.total = page?.total ?? 0;
  },
  { immediate: true },
);

function handleFilterChange() {
  pagination.current = 1;
  void refetchList();
}

function handleTableChange(pag: { current?: number; pageSize?: number }) {
  if (pag.current) pagination.current = pag.current;
  if (pag.pageSize) pagination.pageSize = pag.pageSize;
}

function handleRangeChange(key: string) {
  if (key === activeRangeKey.value) return;
  activeRangeKey.value = key;
  localStorage.setItem('logflux:security.range', key);
}

function handleIntervalChange(key: string) {
  if (key === activeIntervalKey.value) return;
  activeIntervalKey.value = key;
  localStorage.setItem('logflux:security.interval', key);
}

const columns = [
  { title: '时间', dataIndex: 'eventTime', key: 'eventTime', width: 170 },
  { title: '动作', dataIndex: 'interrupted', key: 'interrupted', width: 84 },
  { title: '规则', dataIndex: 'ruleId', key: 'ruleId', width: 110 },
  { title: '命中信息', dataIndex: 'ruleMsg', key: 'ruleMsg', ellipsis: true },
  { title: '严重度', dataIndex: 'severity', key: 'severity', width: 90 },
  { title: '来源 IP', dataIndex: 'clientIp', key: 'clientIp', width: 140 },
  { title: '位置', key: 'location', width: 150, ellipsis: true },
  { title: '站点', dataIndex: 'host', key: 'host', width: 160, ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
];

function formatLocation(record: CaddyWafEventApi.WafEvent) {
  return [record.country, record.province, record.city]
    .filter((part) => part && part.trim() !== '')
    .join(' / ') || '-';
}

/** 表格插槽的 record 收窄为事件类型 */
function asEvent(record: unknown): CaddyWafEventApi.WafEvent {
  return record as CaddyWafEventApi.WafEvent;
}

const refreshTimer = ref<ReturnType<typeof setInterval> | null>(null);
refreshTimer.value = setInterval(() => {
  void refetchStats();
  void refetchList();
}, 30_000);

onUnmounted(() => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value);
  }
});
</script>

<template>
  <div class="security-page h-full overflow-y-auto p-4">
    <Space direction="vertical" :size="16" class="min-h-full w-full">
      <Alert v-if="statsError" type="error" show-icon :message="statsError" />
      <Alert v-if="listError" type="error" show-icon :message="listError" />

      <Card variant="borderless" class="rounded-2xl shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <div class="text-sm text-gray-500 dark:text-gray-400">时间范围</div>
            <div class="inline-flex">
              <Button
                v-for="item in timeRanges"
                :key="item.key"
                :type="item.key === activeRangeKey ? 'primary' : 'default'"
                @click="handleRangeChange(item.key)"
              >
                {{ item.label }}
              </Button>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <div class="text-sm text-gray-500 dark:text-gray-400">采样时间</div>
            <div class="inline-flex">
              <Button
                v-for="item in intervalOptions"
                :key="item.key"
                :type="item.key === activeIntervalKey ? 'primary' : 'default'"
                @click="handleIntervalChange(item.key)"
              >
                {{ item.label }}
              </Button>
            </div>
          </div>
        </div>
      </Card>

      <Row :gutter="[16, 16]">
        <Col v-for="item in statCards" :key="item.id" class="security-col" :xs="12" :lg="8">
          <StatCard :data="item" />
        </Col>
      </Row>

      <Row :gutter="[16, 16]">
        <Col class="security-col" :xs="24" :lg="14">
          <WafTrendChart
            :times="trendTimes"
            :totals="trendTotals"
            :blockeds="trendBlockeds"
          />
        </Col>
        <Col class="security-col" :xs="24" :lg="10">
          <MapChart :china-data="[]" :world-data="geoWorldData" />
        </Col>
      </Row>

      <Row :gutter="[16, 16]">
        <Col v-for="board in topBoards" :key="board.title" class="security-col" :xs="24" :md="8">
          <Card variant="borderless" class="h-full rounded-2xl shadow-sm" :title="board.title">
            <div class="top-list">
              <div v-for="item in board.items" :key="item.name" class="top-row">
                <span class="top-name" :title="item.name">{{ item.name }}</span>
                <span class="top-count">{{ item.count }}</span>
              </div>
              <div v-if="board.items.length === 0" class="top-empty">暂无数据</div>
            </div>
          </Card>
        </Col>
      </Row>

      <Card variant="borderless" class="rounded-2xl shadow-sm" title="事件明细">
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <Select
            v-model:value="filters.action"
            style="width: 120px"
            :options="[
              { label: '全部动作', value: 'all' },
              { label: '拦截', value: 'block' },
              { label: '仅检测', value: 'detect' },
            ]"
            @change="handleFilterChange"
          />
          <Input
            v-model:value="filters.ruleId"
            allow-clear
            placeholder="规则 ID，如 942100"
            style="width: 170px"
            @press-enter="handleFilterChange"
          />
          <Input
            v-model:value="filters.clientIp"
            allow-clear
            placeholder="来源 IP"
            style="width: 150px"
            @press-enter="handleFilterChange"
          />
          <Input
            v-model:value="filters.keyword"
            allow-clear
            placeholder="URI / 规则消息关键字"
            style="width: 220px"
            @press-enter="handleFilterChange"
          />
          <DatePicker.RangePicker
            v-model:value="filters.timeRange"
            show-time
            @change="handleFilterChange"
          />
        </div>

        <Table
          :columns="columns"
          :data-source="events"
          :loading="loading"
          :pagination="pagination"
          :scroll="{ x: 1200 }"
          row-key="id"
          size="small"
          @change="handleTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'interrupted'">
              <Tag :color="asEvent(record).interrupted ? 'red' : 'orange'">
                {{ asEvent(record).interrupted ? '拦截' : '检测' }}
              </Tag>
            </template>
            <template v-else-if="column.key === 'severity'">
              <Tag :color="severityTag(asEvent(record).severity).color">
                {{ severityTag(asEvent(record).severity).label }}
              </Tag>
            </template>
            <template v-else-if="column.key === 'location'">
              {{ formatLocation(asEvent(record)) }}
            </template>
            <template v-else-if="column.key === 'ruleMsg'">
              <Typography.Text
                style="max-width: 320px"
                :ellipsis="{ tooltip: asEvent(record).ruleMsg || '-' }"
                :content="asEvent(record).ruleMsg || '-'"
              />
            </template>
            <template v-else-if="column.key === 'status'">
              <Typography.Text
                :type="asEvent(record).status >= 400 ? 'danger' : undefined"
              >
                {{ asEvent(record).status || '-' }}
              </Typography.Text>
            </template>
          </template>
          <template #expandedRowRender="{ record }">
            <div class="detail-grid">
              <div>
                <span class="detail-label">事务 ID：</span>{{ asEvent(record).transactionId || '-' }}
              </div>
              <div>
                <span class="detail-label">请求：</span>{{ asEvent(record).method }} {{ asEvent(record).uri }}
              </div>
              <div>
                <span class="detail-label">User-Agent：</span>{{ asEvent(record).userAgent || '-' }}
              </div>
            </div>
          </template>
        </Table>
      </Card>
    </Space>
  </div>
</template>

<style scoped>
.security-page {
  overscroll-behavior: contain;
}

.security-col {
  display: flex;
}

.security-col > :deep(*) {
  width: 100%;
}

.top-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 220px;
}

.top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.top-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: hsl(var(--foreground) / 75%);
}

.top-count {
  font-weight: 600;
  color: #ef4444;
}

.top-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 200px;
  color: hsl(var(--foreground) / 35%);
}

.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: hsl(var(--foreground) / 70%);
}

.detail-label {
  color: hsl(var(--foreground) / 50%);
}
</style>
