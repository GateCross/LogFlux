<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import HeaderBanner from './modules/header-banner.vue';
import StatCard from './modules/stat-card.vue';
import TrendChart from './modules/trend-chart.vue';
import MapChart from './modules/map-chart.vue';
import type { StatCard as StatCardItem } from './data';
import {
  fetchDashboardSummary,
  type DashboardSummaryResp,
  type DashboardRecentItem
} from '@/service/api/dashboard';

const summary = ref<DashboardSummaryResp | null>(null);
const refreshTimer = ref<number | null>(null);

const timeRanges = [
  { key: '1h', label: '1小时', hours: 1 },
  { key: '12h', label: '12小时', hours: 12 },
  { key: '1d', label: '1天', hours: 24 },
  { key: '3d', label: '3天', hours: 72 },
  { key: '7d', label: '一周', hours: 168 },
  { key: '30d', label: '一个月', hours: 720 }
];
const intervalOptions = [
  { key: '30s', label: '30秒', seconds: 30 },
  { key: '1m', label: '1分钟', seconds: 60 },
  { key: '5m', label: '5分钟', seconds: 300 }
];
const activeRangeKey = ref<string>(localStorage.getItem('logflux:dashboard.range') || timeRanges[0].key);
const activeIntervalKey = ref<string>(
  localStorage.getItem('logflux:dashboard.interval') || intervalOptions[1].key
);

const activeRange = computed(() => timeRanges.find(item => item.key === activeRangeKey.value) ?? timeRanges[0]);
const activeInterval = computed(
  () => intervalOptions.find(item => item.key === activeIntervalKey.value) ?? intervalOptions[1]
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
    { id: 0, label: '请求数', value: stats?.requests ?? 0 },
    { id: 1, label: '4xx', value: errorStats?.error4xx ?? 0 },
    { id: 2, label: '5xx', value: errorStats?.error5xx ?? 0 }
  ];
});

const statCards = computed<StatCardItem[]>(() => {
  const stats = summary.value?.stats;
  return [
    { id: 'req', title: '请求次数', value: stats?.requests ?? 0, icon: 'carbon:http', color: '#3b82f6' },
    { id: 'pv', title: '访问次数 (PV)', value: stats?.pv ?? 0, icon: 'carbon:view', color: '#10b981' },
    { id: 'uv', title: '独立访客 (UV)', value: stats?.uv ?? 0, icon: 'carbon:user', color: '#8b5cf6' },
    { id: 'ip', title: '独立 IP', value: stats?.uniqueIp ?? 0, icon: 'carbon:nacl', color: '#f59e0b' },
    { id: 'block', title: '拦截次数', value: stats?.blocked ?? 0, icon: 'carbon:security', color: '#ef4444' },
    { id: 'attack', title: '攻击 IP', value: stats?.attackIp ?? 0, icon: 'carbon:warning-alt', color: '#f97316' }
  ];
});

const errorStats = computed(() => {
  const errors = summary.value?.errorStats;
  const total = summary.value?.stats.requests ?? 0;
  const rateText = (value: number) => (total > 0 ? `${((value / total) * 100).toFixed(2)}%` : '0%');
  return [
    { title: '4xx 错误数', value: errors?.error4xx ?? 0, rate: rateText(errors?.error4xx ?? 0), type: 'error' },
    { title: '4xx 拦截数', value: errors?.blocked4xx ?? 0, rate: rateText(errors?.blocked4xx ?? 0), type: 'info' },
    { title: '5xx 错误数', value: errors?.error5xx ?? 0, rate: rateText(errors?.error5xx ?? 0), type: 'error' }
  ];
});

const trendTimes = computed(() => summary.value?.trend?.map(item => item.time) ?? []);
const trendValues = computed(() => summary.value?.trend?.map(item => item.value) ?? []);
const geoData = computed(() => summary.value?.geo ?? []);
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
    const { data, error } = await fetchDashboardSummary({
      startTime: formatDateTime(start),
      endTime: formatDateTime(now),
      intervalSec: activeInterval.value.seconds,
      topN: 6,
      recentLimit: 6
    });
    if (!error) {
      summary.value = data ?? null;
    }
  } catch {}
}

function handleRangeChange(key: string) {
  if (key === activeRangeKey.value) {
    return;
  }
  activeRangeKey.value = key;
  localStorage.setItem('logflux:dashboard.range', key);
  loadSummary();
}

function handleIntervalChange(key: string) {
  if (key === activeIntervalKey.value) {
    return;
  }
  activeIntervalKey.value = key;
  localStorage.setItem('logflux:dashboard.interval', key);
  loadSummary();
}

function methodClass(method: string) {
  const key = method.toUpperCase();
  if (key === 'GET') return 'bg-green-100 text-green-600';
  if (key === 'POST') return 'bg-blue-100 text-blue-600';
  if (key === 'PUT' || key === 'PATCH') return 'bg-amber-100 text-amber-700';
  if (key === 'DELETE') return 'bg-red-100 text-red-600';
  return 'bg-gray-100 text-gray-600';
}

function statusClass(status: number) {
  if (status >= 500) return 'text-red-500';
  if (status >= 400) return 'text-orange-500';
  if (status >= 300) return 'text-blue-500';
  return 'text-green-600';
}

function formatRecentMeta(item: DashboardRecentItem) {
  const time = item.logTime?.slice(11) || '';
  const ip = item.remoteIp ? ` · ${item.remoteIp}` : '';
  return `${time}${ip}`;
}

onMounted(() => {
  loadSummary();
  refreshTimer.value = window.setInterval(loadSummary, 30000);
});

onUnmounted(() => {
  if (refreshTimer.value) {
    window.clearInterval(refreshTimer.value);
  }
});
</script>

<template>
  <NSpace vertical :size="16">
    <HeaderBanner :range-text="rangeText" :stats="headerStats" />

    <NCard :bordered="false" class="rounded-2xl shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="text-sm text-gray-500">时间范围</div>
          <NButtonGroup size="small">
            <NButton
              v-for="item in timeRanges"
              :key="item.key"
              :type="item.key === activeRangeKey ? 'primary' : 'default'"
              @click="handleRangeChange(item.key)"
            >
              {{ item.label }}
            </NButton>
          </NButtonGroup>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <div class="text-sm text-gray-500">采样时间</div>
          <NButtonGroup size="small">
            <NButton
              v-for="item in intervalOptions"
              :key="item.key"
              :type="item.key === activeIntervalKey ? 'primary' : 'default'"
              @click="handleIntervalChange(item.key)"
            >
              {{ item.label }}
            </NButton>
          </NButtonGroup>
        </div>
      </div>
    </NCard>

    <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem v-for="item in statCards" :key="item.id" span="24 s:12 m:8 l:4">
        <StatCard :data="item" />
      </NGridItem>
    </NGrid>

    <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem v-for="item in errorStats" :key="item.title" span="24 s:12 m:8">
        <NCard :border="false" class="rounded-2xl shadow-sm h-full">
           <div class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <span class="text-gray-500">{{ item.title }}</span>
              <div :class="['i-carbon:warning-filled', item.type === 'error' ? 'text-red-500' : 'text-orange-500']"></div>
            </div>
            <div class="flex items-end gap-2">
              <span class="text-2xl font-bold">{{ item.value }}</span>
              <span class="text-xs text-gray-500 flex items-center bg-gray-100 px-1 rounded">
                {{ item.rate }}
              </span>
            </div>
           </div>
        </NCard>
      </NGridItem>
    </NGrid>

    <NGrid :x-gap="16" :y-gap="16" responsive="screen" item-responsive>
      <NGridItem span="24 l:16">
        <MapChart :data="geoData" />
      </NGridItem>
      <NGridItem span="24 l:8">
         <NSpace vertical :size="16" class="h-full">
            <TrendChart :times="trendTimes" :values="trendValues" />
            <NCard title="实时日志" class="flex-1 rounded-2xl shadow-sm">
               <div v-if="recentLogs.length === 0" class="text-xs text-gray-400">暂无日志</div>
               <div v-else class="flex flex-col gap-3 text-xs">
                <div
                  v-for="item in recentLogs"
                  :key="item.id"
                  class="flex justify-between items-center border-b border-gray-100 pb-2 gap-3"
                >
                  <div class="flex gap-2 items-center min-w-0 flex-1">
                    <span :class="['px-1 rounded', methodClass(item.method)]">{{ item.method || 'N/A' }}</span>
                    <span class="text-gray-600 truncate">{{ item.uri || '-' }}</span>
                  </div>
                  <div class="flex items-center gap-2 shrink-0 whitespace-nowrap">
                    <span :class="statusClass(item.status)">{{ item.status }}</span>
                    <span class="text-gray-400 whitespace-nowrap">{{ formatRecentMeta(item) }}</span>
                  </div>
                </div>
               </div>
            </NCard>
         </NSpace>
      </NGridItem>
    </NGrid>
  </NSpace>
</template>

<style scoped></style>
