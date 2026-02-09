<template>
  <div class="h-full">
    <n-card title="系统日志" :bordered="false" class="h-full rounded-8px shadow-sm">
      <div class="flex-col h-full min-h-0">
        <div class="mb-4 flex flex-wrap items-end gap-3">
          <n-input
            v-model:value="searchParams.keyword"
            placeholder="搜索 内容/位置/原始日志"
            clearable
            class="w-60"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <icon-ic-round-search class="text-16px" />
            </template>
          </n-input>
          <n-select
            v-model:value="searchParams.source"
            :options="sourceOptions"
            class="w-36"
          />
          <n-select
            v-model:value="searchParams.level"
            :options="levelOptions"
            class="w-32"
          />
          <n-date-picker
            v-model:formatted-value="searchParams.timeRange"
            type="datetimerange"
            value-format="yyyy-MM-dd HH:mm:ss"
            clearable
            class="w-72"
          />
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm text-gray-500">快速筛选</span>
            <n-button-group size="small">
              <n-button
                v-for="item in quickPresetOptions"
                :key="item.label"
                :type="isQuickPresetActive(item) ? 'primary' : 'default'"
                @click="applyQuickPreset(item)"
              >
                {{ item.label }}
              </n-button>
            </n-button-group>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm text-gray-500">自动刷新</span>
            <n-button-group size="small">
              <n-button
                v-for="item in autoRefreshOptions"
                :key="item.value"
                :type="item.value === autoRefreshSeconds ? 'primary' : 'default'"
                @click="handleAutoRefreshChange(item.value)"
              >
                {{ item.label }}
              </n-button>
            </n-button-group>
          </div>
          <n-space>
            <n-button type="primary" @click="handleSearch">
              <template #icon>
                <icon-ic-round-search />
              </template>
              搜索
            </n-button>
            <n-button @click="handleRefresh">
              <template #icon>
                <icon-ic-round-refresh />
              </template>
              刷新
            </n-button>
            <n-button tertiary @click="handleReset">重置</n-button>
          </n-space>
        </div>

        <n-data-table
          remote
          :columns="columns"
          :data="tableData"
          :loading="loading"
          :pagination="pagination"
          :row-key="row => row.id"
          class="h-full"
          flex-height
          :scroll-x="1200"
          :resizable="true"
          @update:sorter="handleSorterChange"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
          size="small"
        />
      </div>
    </n-card>

    <n-modal v-model:show="showDetail" preset="card" title="日志详情" class="w-720px max-h-85vh">
      <div class="max-h-70vh overflow-auto">
        <n-descriptions bordered size="small" :column="1" v-if="selectedLog">
          <n-descriptions-item label="时间">{{ selectedLog.logTime }}</n-descriptions-item>
          <n-descriptions-item label="级别">{{ selectedLog.level }}</n-descriptions-item>
          <n-descriptions-item label="来源">{{ sourceLabel(selectedLog.source) }}</n-descriptions-item>
          <n-descriptions-item label="内容">{{ selectedLog.message }}</n-descriptions-item>
          <n-descriptions-item label="位置">{{ selectedLog.caller || '-' }}</n-descriptions-item>
          <n-descriptions-item label="Trace">{{ selectedLog.traceId || '-' }}</n-descriptions-item>
          <n-descriptions-item label="Span">{{ selectedLog.spanId || '-' }}</n-descriptions-item>
          <n-descriptions-item label="扩展字段">
            <div class="w-full">
              <pre class="log-detail-pre">{{ extraDataText }}</pre>
              <div class="mt-2">
                <n-button
                  v-if="canToggleExtraData"
                  text
                  type="primary"
                  size="small"
                  @click="toggleDetailExpand('extraData')"
                >
                  {{ detailExpandState.extraData ? '收起' : '展开' }}
                </n-button>
              </div>
            </div>
          </n-descriptions-item>
          <n-descriptions-item label="原始日志">
            <div class="w-full">
              <pre class="log-detail-pre">{{ rawLogText }}</pre>
              <div class="mt-2 flex items-center gap-3">
                <n-button
                  v-if="canToggleRawLog"
                  text
                  type="primary"
                  size="small"
                  @click="toggleDetailExpand('rawLog')"
                >
                  {{ detailExpandState.rawLog ? '收起' : '展开' }}
                </n-button>
                <n-button text type="primary" size="small" @click="copySingleLog(selectedLog)">复制日志</n-button>
              </div>
            </div>
          </n-descriptions-item>
        </n-descriptions>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, h, computed } from 'vue';
import { NTag, NButton, useMessage } from 'naive-ui';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import { fetchSystemLogs } from '@/service/api/system-log';

interface SystemLog {
  id: number;
  logTime: string;
  level: string;
  message: string;
  caller: string;
  traceId?: string;
  spanId?: string;
  source: string;
  rawLog: string;
  extraData: string;
}

const message = useMessage();
const loading = ref(false);
const tableData = ref<SystemLog[]>([]);
const selectedLog = ref<SystemLog | null>(null);
const showDetail = ref(false);
const autoRefreshTimer = ref<number | null>(null);
const autoRefreshSeconds = ref(loadAutoRefreshSeconds());

const detailExpandState = reactive({
  extraData: false,
  rawLog: false
});
const detailPreviewMaxLength = 1200;

const rawLogFullText = computed(() => normalizeJson(selectedLog.value?.rawLog));
const extraDataFullText = computed(() => normalizeJson(selectedLog.value?.extraData));
const rawLogText = computed(() => renderDetailText(rawLogFullText.value, detailExpandState.rawLog));
const extraDataText = computed(() => renderDetailText(extraDataFullText.value, detailExpandState.extraData));
const canToggleRawLog = computed(() => shouldCollapseText(rawLogFullText.value));
const canToggleExtraData = computed(() => shouldCollapseText(extraDataFullText.value));

const searchParams = reactive({
  keyword: '',
  source: '',
  level: '',
  timeRange: null as [string, string] | null
});

const autoRefreshOptions = [
  { label: '关闭', value: 0 },
  { label: '5秒', value: 5 },
  { label: '10秒', value: 10 }
];

const quickPresetOptions = [
  { label: '仅后端', source: 'backend', level: '' },
  { label: '仅 Caddy 后台', source: 'caddy_runtime', level: '' },
  { label: '仅错误级别', source: '', level: 'error' }
];

type SortOrder = 'ascend' | 'descend' | false;

const sortState = ref<{ columnKey: string; order: SortOrder }>({
  columnKey: 'logTime',
  order: 'descend'
});

const pagination = reactive<PaginationProps>({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  itemCount: 0,
  onChange: (page: number) => {
    pagination.page = page;
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize;
    pagination.page = 1;
  }
});

const columns: DataTableColumns<SystemLog> = [
  {
    title: '时间',
    key: 'logTime',
    width: 160,
    resizable: true,
    sorter: 'default',
    defaultSortOrder: 'descend',
    ellipsis: { tooltip: true }
  },
  {
    title: '级别',
    key: 'level',
    width: 90,
    resizable: true,
    render(row) {
      return h(NTag, { type: levelTagType(row.level), size: 'small' }, { default: () => row.level || '-' });
    }
  },
  {
    title: '来源',
    key: 'source',
    width: 120,
    resizable: true,
    render(row) {
      return h(NTag, { type: sourceTagType(row.source), size: 'small' }, { default: () => sourceLabel(row.source) });
    }
  },
  {
    title: '内容',
    key: 'message',
    minWidth: 240,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '位置',
    key: 'caller',
    minWidth: 180,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: 'Trace',
    key: 'traceId',
    width: 180,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: 'Span',
    key: 'spanId',
    width: 160,
    resizable: true,
    ellipsis: { tooltip: true }
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    resizable: true,
    fixed: 'right',
    render(row) {
      return h('div', { class: 'flex items-center gap-2' }, [
        h(
          NButton,
          {
            size: 'tiny',
            onClick: () => openDetail(row)
          },
          { default: () => '详情' }
        ),
        h(
          NButton,
          {
            size: 'tiny',
            tertiary: true,
            onClick: () => copySingleLog(row)
          },
          { default: () => '复制' }
        )
      ]);
    }
  }
];

const sourceOptions = [
  { label: '全部来源', value: '' },
  { label: '后端', value: 'backend' },
  { label: 'Caddy 后台', value: 'caddy_runtime' }
];

const levelOptions = [
  { label: '全部级别', value: '' },
  { label: 'debug', value: 'debug' },
  { label: 'info', value: 'info' },
  { label: 'warn', value: 'warn' },
  { label: 'error', value: 'error' }
];

async function fetchData() {
  if (loading.value) {
    return;
  }
  loading.value = true;
  try {
    const [startTime, endTime] = searchParams.timeRange ?? [undefined, undefined];
    const { data, error } = await fetchSystemLogs({
      page: pagination.page || 1,
      pageSize: pagination.pageSize || 20,
      keyword: searchParams.keyword,
      source: searchParams.source || undefined,
      level: searchParams.level || undefined,
      startTime,
      endTime,
      sortBy: sortState.value.order ? 'logTime' : undefined,
      order: sortState.value.order === 'ascend' ? 'asc' : sortState.value.order === 'descend' ? 'desc' : undefined
    });

    if (error) {
      message.error('获取系统日志失败');
      return;
    }

    if (data) {
      tableData.value = data.list || [];
      pagination.itemCount = data.total || 0;
    }
  } catch (err) {
    console.error(err);
    message.error('系统错误');
  } finally {
    loading.value = false;
  }
}

function loadAutoRefreshSeconds() {
  const saved = Number(localStorage.getItem('logflux:system-log.autoRefreshSeconds') || 0);
  if (saved === 5 || saved === 10) {
    return saved;
  }
  return 0;
}

function clearAutoRefresh() {
  if (autoRefreshTimer.value !== null) {
    window.clearInterval(autoRefreshTimer.value);
    autoRefreshTimer.value = null;
  }
}

function restartAutoRefresh() {
  clearAutoRefresh();
  if (autoRefreshSeconds.value > 0) {
    autoRefreshTimer.value = window.setInterval(() => {
      fetchData();
    }, autoRefreshSeconds.value * 1000);
  }
}

function handleAutoRefreshChange(value: number) {
  if (value === autoRefreshSeconds.value) {
    return;
  }
  autoRefreshSeconds.value = value;
  localStorage.setItem('logflux:system-log.autoRefreshSeconds', String(value));
  restartAutoRefresh();
  if (value > 0) {
    fetchData();
  }
}

function handleSearch() {
  pagination.page = 1;
  fetchData();
}

function handleRefresh() {
  fetchData();
}

function handleReset() {
  searchParams.keyword = '';
  searchParams.source = '';
  searchParams.level = '';
  searchParams.timeRange = null;
  pagination.page = 1;
  sortState.value = { columnKey: 'logTime', order: 'descend' };
  fetchData();
}

function openDetail(row: SystemLog) {
  selectedLog.value = row;
  detailExpandState.extraData = false;
  detailExpandState.rawLog = false;
  showDetail.value = true;
}

function applyQuickPreset(preset: { source: string; level: string }) {
  searchParams.source = preset.source;
  searchParams.level = preset.level;
  pagination.page = 1;
  fetchData();
}

function isQuickPresetActive(preset: { source: string; level: string }) {
  return searchParams.source === preset.source && searchParams.level === preset.level;
}

function handlePageChange(page: number) {
  pagination.page = page;
  fetchData();
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize;
  pagination.page = 1;
  fetchData();
}

function handleSorterChange(sorter: any) {
  const normalized = Array.isArray(sorter) ? sorter[0] : sorter;
  if (!normalized) {
    sortState.value = { columnKey: 'logTime', order: false };
  } else {
    const order: SortOrder = normalized.order === 'ascend' || normalized.order === 'descend' ? normalized.order : false;
    sortState.value = {
      columnKey: normalized.columnKey ? String(normalized.columnKey) : 'logTime',
      order
    };
  }
  pagination.page = 1;
  fetchData();
}

function levelTagType(level: string) {
  switch ((level || '').toLowerCase()) {
    case 'debug':
      return 'default';
    case 'info':
      return 'info';
    case 'warn':
      return 'warning';
    case 'error':
      return 'error';
    default:
      return 'default';
  }
}

function sourceTagType(source: string) {
  if (source === 'backend') return 'success';
  if (source === 'caddy_runtime') return 'warning';
  return 'default';
}

function sourceLabel(source: string) {
  if (source === 'backend') return '后端';
  if (source === 'caddy_runtime') return 'Caddy 后台';
  return source || '-';
}

function normalizeJson(value?: string) {
  if (!value) return '-';
  try {
    const parsed = JSON.parse(value);
    if (typeof parsed === 'string') return parsed;
    return JSON.stringify(parsed, null, 2);
  } catch {
    return value;
  }
}

function shouldCollapseText(value: string) {
  return value !== '-' && value.length > detailPreviewMaxLength;
}

function renderDetailText(value: string, expanded: boolean) {
  if (!shouldCollapseText(value) || expanded) {
    return value;
  }
  return `${value.slice(0, detailPreviewMaxLength)}\n...(已截断，点击展开查看完整内容)`;
}

function toggleDetailExpand(field: 'extraData' | 'rawLog') {
  detailExpandState[field] = !detailExpandState[field];
}

function buildLogCopyText(log: SystemLog) {
  return [
    `时间: ${log.logTime || '-'}`,
    `级别: ${log.level || '-'}`,
    `来源: ${sourceLabel(log.source)}`,
    `内容: ${log.message || '-'}`,
    `位置: ${log.caller || '-'}`,
    `Trace: ${log.traceId || '-'}`,
    `Span: ${log.spanId || '-'}`,
    `扩展字段: ${normalizeJson(log.extraData)}`,
    `原始日志: ${normalizeJson(log.rawLog)}`
  ].join('\n');
}

function fallbackCopyText(content: string) {
  const textarea = document.createElement('textarea');
  textarea.value = content;
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const success = document.execCommand('copy');
  document.body.removeChild(textarea);
  return success;
}

async function copySingleLog(log: SystemLog | null) {
  if (!log) {
    message.warning('未找到可复制的日志内容');
    return;
  }

  const text = buildLogCopyText(log);
  try {
    await navigator.clipboard.writeText(text);
    message.success('日志已复制');
  } catch {
    if (fallbackCopyText(text)) {
      message.success('日志已复制');
      return;
    }
    message.error('复制失败');
  }
}

onMounted(() => {
  fetchData();
  restartAutoRefresh();
});

onUnmounted(() => {
  clearAutoRefresh();
});
</script>

<style scoped>
.log-detail-pre {
  margin: 0;
  max-height: 320px;
  overflow: auto;
  border-radius: 6px;
  background-color: rgb(245 245 245);
  padding: 10px;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.5;
  font-size: 12px;
}
</style>
