<script lang="ts" setup>
/**
 * 站点详情「日志」Tab：只读查看当前站点的访问日志与错误日志。
 * 两类日志来自同一份采集数据，后端按 level / logger 分流，这里只切换 logKind。
 */
import { computed, ref, watch } from 'vue';

import {
  Alert,
  Empty,
  RadioButton,
  RadioGroup,
  Select,
  Space,
  Table,
  Tag,
} from 'antdv-next';
import type { TableColumnsType, TablePaginationConfig } from 'antdv-next';

import { getCaddyLogsApi, type CaddyServerApi } from '#/api/caddy/server';
import { withListDetailErrorMode } from '#/api/list-detail';
import { qk } from '#/api/query-keys';
import { useListDetailQuery } from '#/composables/use-list-detail-query';
import { statusTagColor } from '#/views/logs/shared/log-status-color';

defineOptions({ name: 'SiteLogPanel' });

const props = defineProps<{
  /** 站点主域名，作为日志查询的默认 host */
  host: string;
  /** 站点全部域名；多于 1 个时提供切换 */
  domains?: string[];
}>();

type LogKind = 'access' | 'error';
type CaddyLog = CaddyServerApi.CaddyLogItem;

/** 当前查询的 host：默认主域名，多域名站点可在切换器改选 */
const activeHost = ref(props.host);

// 父组件切换站点或主域名更新时，查询回到主域名
watch(
  () => props.host,
  (next) => {
    activeHost.value = next;
  },
);

const hostOptions = computed(() =>
  (props.domains ?? [])
    .map((domain) => domain.trim())
    .filter(Boolean)
    .map((domain) => ({ label: domain, value: domain })),
);

/** 只有一个域名时没有切换的必要 */
const showHostSwitcher = computed(() => hostOptions.value.length > 1);

const logKind = ref<LogKind>('access');
const page = ref(1);
const pageSize = ref(20);

const logKindLabel = computed(() =>
  logKind.value === 'access' ? '访问日志' : '错误日志',
);

// 切换日志种类时回到第一页，避免停在超出新结果集的页码上
watch(logKind, () => {
  page.value = 1;
});

// 切换域名时回到第一页；查询键包含 host，切值自动重查
function handleHostChange() {
  page.value = 1;
}

const {
  data: logsPage,
  loading,
  errorMessage,
} = useListDetailQuery({
  queryKey: computed(() =>
    qk.caddy.logs({
      host: activeHost.value,
      logKind: logKind.value,
      page: page.value,
      pageSize: pageSize.value,
    }),
  ),
  queryFn: () =>
    getCaddyLogsApi(
      {
        host: activeHost.value,
        logKind: logKind.value,
        page: page.value,
        pageSize: pageSize.value,
      },
      withListDetailErrorMode(),
    ),
  errorFallback: '获取站点日志失败',
  enabled: computed(() => Boolean(activeHost.value)),
});

const logs = computed(() => logsPage.value?.list ?? []);
const total = computed(() => logsPage.value?.total ?? 0);

/** 禁用查询时 isPending 恒真，无域名时不展示转圈 */
const tableLoading = computed(() => loading.value && Boolean(activeHost.value));

/** 两视图共用列；错误日志额外补 level / logger */
const columns = computed<TableColumnsType<CaddyLog>>(() => {
  const base: TableColumnsType<CaddyLog> = [
    { dataIndex: 'logTime', key: 'logTime', title: '时间', width: 180 },
    { dataIndex: 'host', key: 'host', title: 'host', width: 180, ellipsis: true },
    { dataIndex: 'method', key: 'method', title: '方法', width: 90 },
    { dataIndex: 'uri', key: 'uri', title: 'URI', ellipsis: true },
    { dataIndex: 'status', key: 'status', title: '状态', width: 90 },
    {
      dataIndex: 'clientIp',
      key: 'clientIp',
      title: '客户端 IP',
      width: 150,
      ellipsis: true,
    },
  ];
  if (logKind.value === 'error') {
    base.push(
      { dataIndex: 'level', key: 'level', title: 'level', width: 90 },
      { dataIndex: 'logger', key: 'logger', title: 'logger', width: 200, ellipsis: true },
    );
  }
  return base;
});

/** 服务端分页：Table 只负责展示与回传页码 */
const pagination = computed<TablePaginationConfig>(() => ({
  current: page.value,
  pageSize: pageSize.value,
  pageSizeOptions: ['10', '20', '50', '100'],
  showSizeChanger: true,
  showTotal: (count: number) => `共 ${count} 条`,
  total: total.value,
}));

function handleTableChange(next: TablePaginationConfig) {
  // 改每页条数时回到第一页，否则可能落到空页
  if (next.pageSize && next.pageSize !== pageSize.value) {
    pageSize.value = next.pageSize;
    page.value = 1;
    return;
  }
  page.value = next.current ?? 1;
}

/** 错误级别着色；未知值保持中性 */
function levelTagColor(level?: string) {
  const key = (level ?? '').toLowerCase();
  if (key === 'error') return 'red';
  if (key === 'warn' || key === 'warning') return 'orange';
  if (key === 'fatal' || key === 'panic') return 'magenta';
  return 'default';
}
</script>

<template>
  <div class="site-log-panel">
    <div class="log-toolbar">
      <Space size="small">
        <RadioGroup v-model:value="logKind" button-style="solid" size="small">
          <RadioButton value="access">访问日志</RadioButton>
          <RadioButton value="error">错误日志</RadioButton>
        </RadioGroup>
        <Select
          v-if="showHostSwitcher"
          v-model:value="activeHost"
          :options="hostOptions"
          size="small"
          class="host-select"
          @change="handleHostChange"
        />
      </Space>
      <span class="log-tip">
        仅展示当前域名（{{ activeHost || '未配置域名' }}）的日志，只读
      </span>
    </div>

    <Alert
      v-if="errorMessage"
      class="mb-3"
      type="error"
      show-icon
      :message="errorMessage"
    />

    <Table
      :columns="columns"
      :data-source="logs"
      :loading="tableLoading"
      :pagination="pagination"
      :scroll="{ x: 1080 }"
      row-key="id"
      size="middle"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <Tag :color="statusTagColor((record as CaddyLog).status)">
            {{ (record as CaddyLog).status || '-' }}
          </Tag>
        </template>
        <template v-else-if="column.key === 'level'">
          <Tag :color="levelTagColor((record as CaddyLog).level)">
            {{ (record as CaddyLog).level || '-' }}
          </Tag>
        </template>
        <template v-else-if="column.key === 'method'">
          {{ (record as CaddyLog).method || '-' }}
        </template>
        <template v-else-if="column.key === 'clientIp'">
          {{ (record as CaddyLog).clientIp || '-' }}
        </template>
        <template v-else-if="column.key === 'logger'">
          {{ (record as CaddyLog).logger || '-' }}
        </template>
      </template>
      <!-- 错误由上方 Alert 承担，空态只说明没有记录 -->
      <template #emptyText>
        <Empty :description="`暂无${logKindLabel}记录`" />
      </template>
    </Table>
  </div>
</template>

<style scoped>
.log-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.log-tip {
  font-size: 12px;
  color: hsl(var(--muted-foreground));
}

.host-select {
  width: 220px;
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
