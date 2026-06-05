<script lang="ts" setup>
import type { SystemLogApi } from '#/api/system/log';
import type { TableColumnsType } from 'ant-design-vue';

import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';

import {
  Button,
  Card,
  DatePicker,
  Descriptions,
  DescriptionsItem,
  Form,
  FormItem,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  message,
} from 'ant-design-vue';

import { getSystemLogsApi } from '#/api/system/log';

type SortOrder = 'ascend' | 'descend' | false;
type SystemLog = SystemLogApi.LogItem;

const loading = ref(false);
const dataSource = ref<SystemLog[]>([]);
const selectedLog = ref<SystemLog | null>(null);
const detailVisible = ref(false);
const autoRefreshTimer = ref<number | null>(null);
const autoRefreshSeconds = ref(loadAutoRefreshSeconds());

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
  { label: '关闭', value: 0 },
  { label: '5秒', value: 5 },
  { label: '10秒', value: 10 },
];

const quickPresetOptions = [
  { label: '仅后端', level: '', source: 'backend' },
  { label: '仅 Caddy 后台', level: '', source: 'caddy_runtime' },
  { label: '仅本机访问', level: '', source: 'caddy_internal' },
  { label: '仅错误级别', level: 'error', source: '' },
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

onMounted(() => {
  void fetchLogs();
  restartAutoRefresh();
});

onUnmounted(() => {
  clearAutoRefresh();
});
</script>

<template>
  <Page description="查看后端与 Caddy 运行时日志。" title="系统日志">
    <div class="system-log-page">
      <Card :bordered="false" class="system-log-card" title="系统日志">
        <Form class="log-filter-form" layout="inline">
          <FormItem label="关键字">
            <Input.Search
              v-model:value="filters.keyword"
              allow-clear
              placeholder="搜索 内容/位置/原始日志"
              style="width: 260px;"
              @search="handleSearch"
            />
          </FormItem>
          <FormItem label="来源">
            <Select v-model:value="filters.source" :options="sourceOptions" style="width: 150px;" @change="handleSearch" />
          </FormItem>
          <FormItem label="级别">
            <Select v-model:value="filters.level" :options="levelOptions" style="width: 130px;" @change="handleSearch" />
          </FormItem>
          <FormItem label="时间">
            <DatePicker.RangePicker
              v-model:value="filters.timeRange"
              show-time
              value-format="YYYY-MM-DD HH:mm:ss"
              style="width: 330px;"
            />
          </FormItem>
          <FormItem>
            <Space>
              <Button type="primary" @click="handleSearch">搜索</Button>
              <Button @click="fetchLogs">刷新</Button>
              <Button @click="handleReset">重置</Button>
            </Space>
          </FormItem>
        </Form>

        <div class="filter-strip">
          <span class="strip-label">快速筛选</span>
          <Space wrap>
            <Button
              v-for="item in quickPresetOptions"
              :key="item.label"
              size="small"
              :type="isQuickPresetActive(item) ? 'primary' : 'default'"
              @click="applyQuickPreset(item)"
            >
              {{ item.label }}
            </Button>
          </Space>
          <span class="strip-label refresh-label">自动刷新</span>
          <Space wrap>
            <Button
              v-for="item in autoRefreshOptions"
              :key="item.value"
              size="small"
              :type="item.value === autoRefreshSeconds ? 'primary' : 'default'"
              @click="handleAutoRefreshChange(item.value)"
            >
              {{ item.label }}
            </Button>
          </Space>
        </div>

        <Table
          :columns="columns"
          :data-source="dataSource"
          :loading="loading"
          :pagination="pagination"
          :scroll="{ x: 1310 }"
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
  padding: 16px;
}

.system-log-card {
  min-height: calc(100vh - 140px);
}

.log-filter-form {
  row-gap: 12px;
  margin-bottom: 12px;
}

.filter-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.strip-label {
  color: #667085;
  font-size: 13px;
}

.refresh-label {
  margin-left: 12px;
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
