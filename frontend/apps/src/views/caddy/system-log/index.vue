<script lang="ts" setup>
import type { SystemLogApi } from '#/api/system/log';
import type { TableColumnsType } from 'ant-design-vue';

import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  DatePicker,
  Descriptions,
  DescriptionsItem,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  message,
} from 'ant-design-vue';

import { clearSystemLogsApi, getSystemLogsApi } from '#/api/system/log';

type SortOrder = 'ascend' | 'descend' | false;
type SystemLog = SystemLogApi.LogItem;

const loading = ref(false);
const clearing = ref(false);
const dataSource = ref<SystemLog[]>([]);
const selectedLog = ref<SystemLog | null>(null);
const detailVisible = ref(false);
const autoRefreshTimer = ref<number | null>(null);
const autoRefreshSeconds = ref(loadAutoRefreshSeconds());
const tableWrapRef = ref<HTMLElement | null>(null);
const tableScrollY = ref<number>();
let tableResizeObserver: ResizeObserver | null = null;

const detailPreviewMaxLength = 1200;
const detailExpandState = reactive({
  extraData: false,
  rawLog: false,
});

const filters = reactive({
  keyword: '',
  level: '',
  source: '',
  timeRange: undefined as [string, string] | undefined,
});

const pagination = reactive({
  current: 1,
  pageSize: 20,
  pageSizeOptions: ['10', '20', '50', '100'],
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
  total: 0,
});

const hasLogs = computed(() => dataSource.value.length > 0);

const tableScroll = computed(() => {
  if (!hasLogs.value || !tableScrollY.value) return { x: 1310 };
  return { x: 1310, y: tableScrollY.value };
});

const sortState = ref<{ columnKey: string; order: SortOrder }>({
  columnKey: 'logTime',
  order: 'descend',
});

const sourceOptions = [
  { label: '全部来源', value: '' },
  { label: '后端', value: 'backend' },
  { label: 'Caddy 后台', value: 'caddy_runtime' },
  { label: 'Caddy 本机访问', value: 'caddy_internal' },
];

const levelOptions = [
  { label: '全部级别', value: '' },
  { label: 'debug', value: 'debug' },
  { label: 'info', value: 'info' },
  { label: 'warn', value: 'warn' },
  { label: 'error', value: 'error' },
];

const autoRefreshOptions = [
  { label: '自动刷新: 关', value: 0 },
  { label: '自动刷新: 5秒', value: 5 },
  { label: '自动刷新: 10秒', value: 10 },
];

const quickPresetOptions = [
  { label: '仅后端', level: '', source: 'backend', value: 'backend' },
  { label: '仅 Caddy 后台', level: '', source: 'caddy_runtime', value: 'caddy_runtime' },
  { label: '仅本机访问', level: '', source: 'caddy_internal', value: 'caddy_internal' },
  { label: '仅错误级别', level: 'error', source: '', value: 'error' },
];

const rawLogFullText = computed(() => normalizeJson(selectedLog.value?.rawLog));
const extraDataFullText = computed(() => normalizeJson(selectedLog.value?.extraData));
const rawLogText = computed(() => renderDetailText(rawLogFullText.value, detailExpandState.rawLog));
const extraDataText = computed(() => renderDetailText(extraDataFullText.value, detailExpandState.extraData));
const canToggleRawLog = computed(() => shouldCollapseText(rawLogFullText.value));
const canToggleExtraData = computed(() => shouldCollapseText(extraDataFullText.value));

const columns: TableColumnsType<SystemLog> = [
  {
    dataIndex: 'logTime',
    key: 'logTime',
    sorter: true,
    title: '时间',
    width: 170,
  },
  {
    dataIndex: 'level',
    key: 'level',
    title: '级别',
    width: 90,
  },
  {
    dataIndex: 'source',
    key: 'source',
    title: '来源',
    width: 130,
  },
  {
    dataIndex: 'message',
    ellipsis: true,
    key: 'message',
    title: '内容',
    width: 320,
  },
  {
    dataIndex: 'caller',
    ellipsis: true,
    key: 'caller',
    title: '位置',
    width: 220,
  },
  {
    dataIndex: 'traceId',
    ellipsis: true,
    key: 'traceId',
    title: 'Trace',
    width: 180,
  },
  {
    dataIndex: 'spanId',
    ellipsis: true,
    key: 'spanId',
    title: 'Span',
    width: 160,
  },
  {
    fixed: 'right',
    key: 'actions',
    title: '操作',
    width: 130,
  },
];

async function fetchLogs() {
  if (loading.value) return;
  loading.value = true;
  try {
    const [startTime, endTime] = filters.timeRange ?? [undefined, undefined];
    const data = await getSystemLogsApi({
      endTime,
      keyword: filters.keyword,
      level: filters.level || undefined,
      order:
        sortState.value.order === 'ascend'
          ? 'asc'
          : sortState.value.order === 'descend'
            ? 'desc'
            : undefined,
      page: pagination.current,
      pageSize: pagination.pageSize,
      sortBy: sortState.value.order ? 'logTime' : undefined,
      source: filters.source || undefined,
      startTime,
    });
    dataSource.value = data.list ?? [];
    pagination.total = data.total ?? 0;
  } catch {
    message.error('获取系统日志失败');
  } finally {
    loading.value = false;
    void nextTick(updateTableScrollY);
  }
}

function loadAutoRefreshSeconds() {
  const saved = Number(localStorage.getItem('logflux:system-log.autoRefreshSeconds') || 0);
  return saved === 5 || saved === 10 ? saved : 0;
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
      void fetchLogs();
    }, autoRefreshSeconds.value * 1000);
  }
}

function handleAutoRefreshChange(value: number) {
  if (value === autoRefreshSeconds.value) return;
  autoRefreshSeconds.value = value;
  localStorage.setItem('logflux:system-log.autoRefreshSeconds', String(value));
  restartAutoRefresh();
  if (value > 0) void fetchLogs();
}

function handleSearch() {
  pagination.current = 1;
  void fetchLogs();
}

function handleReset() {
  filters.keyword = '';
  filters.source = '';
  filters.level = '';
  filters.timeRange = undefined;
  pagination.current = 1;
  sortState.value = { columnKey: 'logTime', order: 'descend' };
  void fetchLogs();
}

async function handleClearLogs() {
  if (clearing.value) return;
  clearing.value = true;
  try {
    await clearSystemLogsApi();
    message.success('系统日志已清空');
    selectedLog.value = null;
    detailVisible.value = false;
    pagination.current = 1;
    dataSource.value = [];
    pagination.total = 0;
    await fetchLogs();
  } catch {
    message.error('清空系统日志失败');
  } finally {
    clearing.value = false;
  }
}

function handleTableChange(pag: any, _filters: any, sorter: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  const normalized = Array.isArray(sorter) ? sorter[0] : sorter;
  sortState.value = {
    columnKey: normalized?.columnKey ? String(normalized.columnKey) : 'logTime',
    order: normalized?.order === 'ascend' || normalized?.order === 'descend' ? normalized.order : false,
  };
  void fetchLogs();
}

function openDetail(row: SystemLog) {
  selectedLog.value = row;
  detailExpandState.extraData = false;
  detailExpandState.rawLog = false;
  detailVisible.value = true;
}

function applyQuickPreset(preset: { level: string; source: string }) {
  filters.source = preset.source;
  filters.level = preset.level;
  pagination.current = 1;
  void fetchLogs();
}

function handleAutoRefreshSelectChange(value: unknown) {
  if (typeof value === 'number') handleAutoRefreshChange(value);
}

function isQuickPresetActive(preset: { level: string; source: string }) {
  return filters.source === preset.source && filters.level === preset.level;
}

function levelTagColor(level: string) {
  switch ((level || '').toLowerCase()) {
    case 'debug': {
      return 'default';
    }
    case 'info': {
      return 'blue';
    }
    case 'warn':
    case 'warning': {
      return 'orange';
    }
    case 'error': {
      return 'red';
    }
    default: {
      return 'default';
    }
  }
}

function sourceTagColor(source: string) {
  if (source === 'backend') return 'green';
  if (source === 'caddy_runtime') return 'orange';
  if (source === 'caddy_internal') return 'blue';
  return 'default';
}

function sourceLabel(source: string) {
  if (source === 'backend') return '后端';
  if (source === 'caddy_runtime') return 'Caddy 后台';
  if (source === 'caddy_internal') return 'Caddy 本机访问';
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
  if (!shouldCollapseText(value) || expanded) return value;
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
    `原始日志: ${normalizeJson(log.rawLog)}`,
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

function updateTableScrollY() {
  const wrap = tableWrapRef.value;
  if (!wrap || !hasLogs.value) {
    tableScrollY.value = undefined;
    return;
  }

  const header = wrap.querySelector<HTMLElement>('.ant-table-thead');
  const paginationEl = wrap.querySelector<HTMLElement>('.ant-pagination');
  const headerHeight = header?.offsetHeight ?? 55;
  const paginationHeight = paginationEl ? paginationEl.offsetHeight + 12 : 44;
  const nextY = wrap.clientHeight - headerHeight - paginationHeight - 2;

  tableScrollY.value = Math.max(240, nextY);
}

onMounted(() => {
  void fetchLogs();
  restartAutoRefresh();
  tableResizeObserver = new ResizeObserver(() => updateTableScrollY());
  if (tableWrapRef.value) tableResizeObserver.observe(tableWrapRef.value);
  void nextTick(updateTableScrollY);
});

onUnmounted(() => {
  clearAutoRefresh();
  tableResizeObserver?.disconnect();
});
</script>

<template>
  <Page auto-content-height content-class="overflow-hidden">
    <div class="system-log-page">
      <Card :bordered="false" class="system-log-card" title="系统日志">
        <div class="log-filter-panel">
          <div class="log-filter-main">
            <Input.Search
              v-model:value="filters.keyword"
              allow-clear
              class="toolbar-keyword"
              placeholder="关键字"
              @search="handleSearch"
            />
            <Select
              v-model:value="filters.source"
              :options="sourceOptions"
              class="toolbar-source"
              @change="handleSearch"
            />
            <Select
              v-model:value="filters.level"
              :options="levelOptions"
              class="toolbar-level"
              @change="handleSearch"
            />
            <DatePicker.RangePicker
              v-model:value="filters.timeRange"
              class="toolbar-time"
              show-time
              value-format="YYYY-MM-DD HH:mm:ss"
              @change="handleSearch"
            />
            <Button type="primary" @click="handleSearch">搜索</Button>
            <Button @click="fetchLogs">刷新</Button>
            <Button @click="handleReset">重置</Button>
          </div>

          <div class="log-filter-extra">
            <div class="preset-actions">
              <Button
                v-for="item in quickPresetOptions"
                :key="item.value"
                size="small"
                :type="isQuickPresetActive(item) ? 'primary' : 'default'"
                @click="applyQuickPreset(item)"
              >
                {{ item.label }}
              </Button>
            </div>
            <Select
              :options="autoRefreshOptions"
              :value="autoRefreshSeconds"
              class="toolbar-refresh"
              @change="handleAutoRefreshSelectChange"
            />
            <Popconfirm
              title="确认清空全部系统日志？此操作不可恢复。"
              @confirm="handleClearLogs"
            >
              <Button danger :loading="clearing">
                清空日志
              </Button>
            </Popconfirm>
          </div>
        </div>

        <div ref="tableWrapRef" class="system-log-table-wrap">
          <Table
            :columns="columns"
            :data-source="dataSource"
            :loading="loading"
            :pagination="pagination"
            :scroll="tableScroll"
            class="system-log-table"
            row-key="id"
            size="middle"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'level'">
                <Tag :color="levelTagColor(record.level)">{{ record.level || '-' }}</Tag>
              </template>
              <template v-if="column.key === 'source'">
                <Tag :color="sourceTagColor(record.source)">{{ sourceLabel(record.source) }}</Tag>
              </template>
              <template v-if="column.key === 'actions'">
                <Space>
                  <Button size="small" type="link" @click="openDetail(record as SystemLog)">详情</Button>
                  <Button size="small" type="link" @click="copySingleLog(record as SystemLog)">复制</Button>
                </Space>
              </template>
            </template>
          </Table>
        </div>
      </Card>

      <Modal v-model:open="detailVisible" title="日志详情" :footer="null" width="760px">
        <Descriptions v-if="selectedLog" bordered size="small" :column="1">
          <DescriptionsItem label="时间">{{ selectedLog.logTime }}</DescriptionsItem>
          <DescriptionsItem label="级别">{{ selectedLog.level }}</DescriptionsItem>
          <DescriptionsItem label="来源">{{ sourceLabel(selectedLog.source) }}</DescriptionsItem>
          <DescriptionsItem label="内容">{{ selectedLog.message }}</DescriptionsItem>
          <DescriptionsItem label="位置">{{ selectedLog.caller || '-' }}</DescriptionsItem>
          <DescriptionsItem label="Trace">{{ selectedLog.traceId || '-' }}</DescriptionsItem>
          <DescriptionsItem label="Span">{{ selectedLog.spanId || '-' }}</DescriptionsItem>
          <DescriptionsItem label="扩展字段">
            <pre class="log-detail-pre">{{ extraDataText }}</pre>
            <Button v-if="canToggleExtraData" size="small" type="link" @click="toggleDetailExpand('extraData')">
              {{ detailExpandState.extraData ? '收起' : '展开' }}
            </Button>
          </DescriptionsItem>
          <DescriptionsItem label="原始日志">
            <pre class="log-detail-pre">{{ rawLogText }}</pre>
            <Space>
              <Button v-if="canToggleRawLog" size="small" type="link" @click="toggleDetailExpand('rawLog')">
                {{ detailExpandState.rawLog ? '收起' : '展开' }}
              </Button>
              <Button size="small" type="link" @click="copySingleLog(selectedLog)">复制日志</Button>
            </Space>
          </DescriptionsItem>
        </Descriptions>
      </Modal>
    </div>
  </Page>
</template>

<style scoped>
.system-log-page {
  height: 100%;
  min-height: 0;
}

.system-log-card {
  height: 100%;
  min-height: 0;
}

.system-log-card :deep(.ant-card-body) {
  display: flex;
  flex-direction: column;
  height: calc(100% - 57px);
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.log-filter-panel {
  padding: 12px;
  margin-bottom: 12px;
  background: #f8fafc;
  border: 1px solid #eef2f7;
  border-radius: 6px;
}

.log-filter-main,
.log-filter-extra {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.log-filter-extra {
  justify-content: space-between;
  padding-top: 10px;
  margin-top: 10px;
  border-top: 1px solid #eef2f7;
}

.toolbar-keyword {
  width: 260px;
}

.toolbar-source {
  width: 150px;
}

.toolbar-level {
  width: 130px;
}

.toolbar-refresh {
  width: 150px;
}

.toolbar-time {
  width: 330px;
}

.preset-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

@media (max-width: 960px) {
  .toolbar-keyword,
  .toolbar-source,
  .toolbar-level,
  .toolbar-time,
  .toolbar-refresh {
    width: 100%;
  }

  .log-filter-extra {
    align-items: stretch;
    flex-direction: column;
  }

  .preset-actions {
    width: 100%;
  }
}

.system-log-table-wrap {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.system-log-table :deep(.ant-table-cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.system-log-table {
  height: 100%;
}

.system-log-table :deep(.ant-spin-nested-loading),
.system-log-table :deep(.ant-spin-container) {
  height: 100%;
}

.system-log-table :deep(.ant-spin-container) {
  display: flex;
  flex-direction: column;
}

.system-log-table :deep(.ant-table) {
  flex: 1;
  min-height: 0;
}

.system-log-table :deep(.ant-table-pagination.ant-pagination) {
  margin: 12px 0 0;
}

.log-detail-pre {
  max-height: 320px;
  margin: 0 0 8px;
  overflow: auto;
  padding: 10px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  background: #f5f5f5;
  border-radius: 6px;
}
</style>
