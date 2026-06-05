<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useDebounceFn } from '@vueuse/core';
import { Icon } from '@iconify/vue';
import { Button, Card, Col, Row, Space } from 'ant-design-vue';
import type {
  DashboardSummaryResp,
} from '#/api/dashboard';
import { getDashboardSummaryApi } from '#/api/dashboard';
import HeaderBanner from './modules/header-banner.vue';
import StatCard from './modules/stat-card.vue';
import TrendChart from './modules/trend-chart.vue';
import MapChart from './modules/map-chart.vue';
import RecentLogs from './modules/recent-logs.vue';
import type { StatCard as StatCardItem } from './data';

defineOptions({ name: 'Dashboard' });

const summary = ref<DashboardSummaryResp | null>(null);
const refreshTimer = ref<ReturnType<typeof setInterval> | null>(null);

const timeRanges = [
  { key: '1h', label: '1小时', hours: 1 },
  { key: '12h', label: '12小时', hours: 12 },
  { key: '1d', label: '1天', hours: 24 },
  { key: '3d', label: '3天', hours: 72 },
  { key: '7d', label: '一周', hours: 168 },
  { key: '30d', label: '一个月', hours: 720 },
];
const intervalOptions = [
  { key: '30s', label: '30秒', seconds: 30 },
  { key: '1m', label: '1分钟', seconds: 60 },
  { key: '5m', label: '5分钟', seconds: 300 },
];
const activeRangeKey = ref<string>(
  localStorage.getItem('logflux:dashboard.range') || timeRanges[0]!.key,
);
const activeIntervalKey = ref<string>(
  localStorage.getItem('logflux:dashboard.interval') || intervalOptions[1]!.key,
);

const activeRange = computed(
  () => timeRanges.find((item) => item.key === activeRangeKey.value) ?? timeRanges[0]!,
);
const activeInterval = computed(
  () =>
    intervalOptions.find((item) => item.key === activeIntervalKey.value) ?? intervalOptions[1]!,
);

const rangeText = computed(() => {
  if (!summary.value) {
    return `最近 ${activeRange.value.label}`;
  }
  return `${summary.value.range.startTime} ~ ${summary.value.range.endTime}`;
});

const headerStats = computed(() => {
  const stats = summary.value?.stats;
  const errorStats = summary.value?.errorStats;
  return [
    {
      id: 0,
      label: '请求数',
      value: stats?.requests ?? 0,
      color: '#3b82f6',
      icon: 'mdi:arrow-decision-outline',
    },
    {
      id: 1,
      label: '4xx',
      value: errorStats?.error4xx ?? 0,
      color: '#f59e0b',
      icon: 'mdi:alert-outline',
    },
    {
      id: 2,
      label: '5xx',
      value: errorStats?.error5xx ?? 0,
      color: '#ef4444',
      icon: 'mdi:server-network-off',
    },
  ];
});

const statCards = computed<StatCardItem[]>(() => {
  const stats = summary.value?.stats;
  return [
    {
      id: 'req',
      title: '请求次数',
      value: stats?.requests ?? 0,
      icon: 'mdi:arrow-decision-outline',
      color: '#3b82f6',
    },
    {
      id: 'pv',
      title: '访问次数 (PV)',
      value: stats?.pv ?? 0,
      icon: 'mdi:eye-outline',
      color: '#10b981',
    },
    {
      id: 'uv',
      title: '独立访客 (UV)',
      value: stats?.uv ?? 0,
      icon: 'mdi:account-group-outline',
      color: '#8b5cf6',
    },
    {
      id: 'ip',
      title: '独立 IP',
      value: stats?.uniqueIp ?? 0,
      icon: 'mdi:ip-network-outline',
      color: '#f59e0b',
    },
    {
      id: 'block',
      title: '拦截次数',
      value: stats?.blocked ?? 0,
      icon: 'mdi:shield-check-outline',
      color: '#ef4444',
    },
    {
      id: 'attack',
      title: '攻击 IP',
      value: stats?.attackIp ?? 0,
      icon: 'mdi:bug-outline',
      color: '#f97316',
    },
  ];
});

const ERROR_CARD_META = [
  { title: '4xx 错误数', icon: 'mdi:alert-outline', color: '#f59e0b' },
  { title: '4xx 拦截数', icon: 'mdi:shield-lock-outline', color: '#3b82f6' },
  { title: '5xx 错误数', icon: 'mdi:server-network-off', color: '#ef4444' },
] as const;

const errorStats = computed(() => {
  const errors = summary.value?.errorStats;
  const total = summary.value?.stats.requests ?? 0;
  const rateValue = (value: number) => (total > 0 ? (value / total) * 100 : 0);
  const rateText = (value: number) => {
    const r = rateValue(value);
    return r < 0.01 && r > 0 ? '<0.01%' : `${r.toFixed(2)}%`;
  };
  const values = [errors?.error4xx ?? 0, errors?.blocked4xx ?? 0, errors?.error5xx ?? 0];
  return ERROR_CARD_META.map((meta, i) => ({
    ...meta,
    value: values[i]!,
    rate: rateText(values[i]!),
    rateNum: rateValue(values[i]!),
    bg: `linear-gradient(135deg, ${meta.color}14 0%, ${meta.color}05 100%)`,
    border: meta.color,
  }));
});

const trendTimes = computed(() => summary.value?.trend?.map((item) => item.time) ?? []);
const trendValues = computed(() => summary.value?.trend?.map((item) => item.value) ?? []);
const geoWorldData = computed(() => summary.value?.geo ?? []);
const geoChinaData = computed(() => summary.value?.geoProvince ?? []);
const recentLogs = computed(() => summary.value?.recent ?? []);

function formatDateTime(value: Date) {
  const pad = (num: number) => String(num).padStart(2, '0');
  const yyyy = value.getFullYear();
  const MM = pad(value.getMonth() + 1);
  const dd = pad(value.getDate());
  const hh = pad(value.getHours());
  const mm = pad(value.getMinutes());
  const ss = pad(value.getSeconds());
  return `${yyyy}-${MM}-${dd} ${hh}:${mm}:${ss}`;
}

async function loadSummary() {
  try {
    const now = new Date();
    const start = new Date(now.getTime() - activeRange.value.hours * 3600 * 1000);
    const data = await getDashboardSummaryApi({
      startTime: formatDateTime(start),
      endTime: formatDateTime(now),
      intervalSec: activeInterval.value.seconds,
      topN: 6,
      recentLimit: 6,
    });
    if (data) {
      summary.value = data;
    }
  } catch {
    // error handled by request interceptor
  }
}

const debouncedLoadSummary = useDebounceFn(loadSummary, 300);

function handleRangeChange(key: string) {
  if (key === activeRangeKey.value) return;
  activeRangeKey.value = key;
  localStorage.setItem('logflux:dashboard.range', key);
  debouncedLoadSummary();
}

function handleIntervalChange(key: string) {
  if (key === activeIntervalKey.value) return;
  activeIntervalKey.value = key;
  localStorage.setItem('logflux:dashboard.interval', key);
  debouncedLoadSummary();
}

onMounted(() => {
  loadSummary();
  refreshTimer.value = setInterval(loadSummary, 30_000);
});

onUnmounted(() => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value);
  }
});
</script>

<template>
  <div class="dashboard-page h-full overflow-y-auto p-4">
    <Space direction="vertical" :size="16" class="min-h-full w-full">
      <HeaderBanner :range-text="rangeText" :stats="headerStats" />

      <Card :bordered="false" class="rounded-2xl shadow-sm">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <div class="text-sm text-gray-500">时间范围</div>
            <Button.Group size="small">
              <Button
                v-for="item in timeRanges"
                :key="item.key"
                :type="item.key === activeRangeKey ? 'primary' : 'default'"
                @click="handleRangeChange(item.key)"
              >
                {{ item.label }}
              </Button>
            </Button.Group>
          </div>
          <div class="flex flex-wrap items-center gap-3">
            <div class="text-sm text-gray-500">采样时间</div>
            <Button.Group size="small">
              <Button
                v-for="item in intervalOptions"
                :key="item.key"
                :type="item.key === activeIntervalKey ? 'primary' : 'default'"
                @click="handleIntervalChange(item.key)"
              >
                {{ item.label }}
              </Button>
            </Button.Group>
          </div>
        </div>
      </Card>

      <Row :gutter="[16, 16]">
        <Col v-for="item in statCards" :key="item.id" :xs="12" :sm="8" :lg="4">
          <StatCard :data="item" />
        </Col>
      </Row>

      <Row :gutter="[16, 16]">
        <Col v-for="item in errorStats" :key="item.title" :xs="24" :sm="12" :lg="8">
          <div
            class="error-card group relative overflow-hidden rounded-2xl p-5 transition-all duration-300 hover:shadow-lg hover:-translate-y-0.5"
            :style="{ background: item.bg, borderLeft: `3px solid ${item.border}` }"
          >
            <div class="flex items-start justify-between">
              <div class="flex flex-col gap-1">
                <span class="text-sm text-gray-500 font-medium">{{ item.title }}</span>
                <div class="flex items-baseline gap-2">
                  <span class="text-3xl font-bold" :style="{ color: item.color }">{{ item.value }}</span>
                  <span
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                    :style="{ color: item.color, backgroundColor: `${item.color}1a` }"
                  >
                    {{ item.rate }}
                  </span>
                </div>
              </div>
              <div
                class="relative h-12 w-12 flex items-center justify-center rounded-full text-xl transition-transform duration-300 group-hover:scale-110"
                :style="{
                  color: item.color,
                  background: `linear-gradient(135deg, ${item.color}20, ${item.color}08)`,
                  boxShadow: `0 0 20px ${item.color}18`,
                }"
              >
                <Icon :icon="item.icon" />
              </div>
            </div>
            <div class="mt-3 h-1.5 w-full overflow-hidden rounded-full" :style="{ backgroundColor: `${item.color}12` }">
              <div
                class="rate-bar h-full rounded-full transition-all duration-700"
                :style="{ width: `${Math.min(item.rateNum, 100)}%`, backgroundColor: item.color }"
              />
            </div>
          </div>
        </Col>
      </Row>

      <Row :gutter="[16, 16]">
        <Col :xs="24" :lg="16">
          <MapChart :china-data="geoChinaData" :world-data="geoWorldData" />
        </Col>
        <Col :xs="24" :lg="8">
          <Space direction="vertical" :size="16" class="w-full">
            <TrendChart :times="trendTimes" :values="trendValues" />
            <RecentLogs :logs="recentLogs" />
          </Space>
        </Col>
      </Row>
    </Space>
  </div>
</template>

<style scoped>
.dashboard-page {
  overscroll-behavior: contain;
}

.error-card {
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.rate-bar {
  min-width: 2px;
}
</style>
