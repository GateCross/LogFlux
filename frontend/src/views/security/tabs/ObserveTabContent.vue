<template>
  <div>
    <n-alert type="info" :show-icon="true" class="mb-3">
      统计口径基于策略绑定作用域与请求日志；“疑似误报”当前为启发式指标（安全端点被拦截），用于辅助调优参考。
    </n-alert>

    <div class="mb-3 flex flex-wrap gap-2 items-center">
      <n-select
        v-model:value="policyStatsQuery.policyId"
        :options="policyStatsPolicyOptions"
        clearable
        placeholder="策略范围"
        class="w-240px"
      />
      <n-select v-model:value="policyStatsQuery.window" :options="observeWindowOptions" class="w-180px" />
      <n-input-number
        v-model:value="policyStatsQuery.intervalSec"
        :show-button="false"
        :min="60"
        :max="86400"
        placeholder="趋势粒度（秒）"
        class="w-180px"
      />
      <n-input-number
        v-model:value="policyStatsQuery.topN"
        :show-button="false"
        :min="1"
        :max="50"
        placeholder="TopN"
        class="w-120px"
      />
      <n-button type="primary" :loading="policyStatsLoading" @click="fetchPolicyStats">
        <template #icon>
          <icon-carbon-search />
        </template>
        查询
      </n-button>
      <n-button @click="resetPolicyStatsQuery">重置</n-button>
      <n-button :disabled="!hasPolicyStatsDrillFilters" @click="clearPolicyStatsDrillFilters">清空下钻</n-button>
      <n-select
        :value="policyFeedbackStatusFilter"
        :options="policyFeedbackStatusFilterOptions"
        placeholder="反馈状态"
        class="w-160px"
        @update:value="value => { setPolicyFeedbackStatusFilter(value as '' | 'pending' | 'confirmed' | 'resolved'); handlePolicyFeedbackStatusFilterChange(); }"
      />
      <n-input
        :value="policyFeedbackAssigneeFilter"
        clearable
        placeholder="责任人"
        class="w-160px"
        @update:value="value => setPolicyFeedbackAssigneeFilter(value)"
        @keyup.enter="handlePolicyFeedbackStatusFilterChange"
      />
      <n-select
        :value="policyFeedbackSlaStatusFilter"
        :options="policyFeedbackSlaStatusOptions"
        class="w-160px"
        @update:value="value => { setPolicyFeedbackSlaStatusFilter(value as 'all' | 'normal' | 'overdue' | 'resolved'); handlePolicyFeedbackStatusFilterChange(); }"
      />
      <n-button type="warning" secondary @click="openPolicyFeedbackModal">标记误报</n-button>
      <n-button type="warning" secondary :disabled="!hasPolicyFeedbackSelection" @click="openPolicyFeedbackBatchProcessModal">
        批量处理（{{ policyFeedbackCheckedRowKeys.length }}）
      </n-button>
      <n-button secondary :loading="policyFeedbackLoading" @click="fetchPolicyFalsePositiveFeedbacks">刷新反馈</n-button>
      <n-button secondary @click="handleCopyPolicyStatsLink">
        <template #icon>
          <icon-carbon-link />
        </template>
        复制筛选链接
      </n-button>
      <n-button secondary :disabled="!policyStatsPreviousSnapshot" @click="handleExportPolicyStatsCompareCsv">导出对比 CSV</n-button>
      <n-button secondary @click="handleExportPolicyStatsCsv">导出 CSV</n-button>
    </div>

    <n-grid cols="5" x-gap="12" y-gap="10">
      <n-gi>
        <n-card size="small" :bordered="false">
          <div class="text-xs text-gray-500">命中</div>
          <div class="text-lg font-semibold">{{ policyStatsSummary.hitCount || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small" :bordered="false">
          <div class="text-xs text-gray-500">拦截</div>
          <div class="text-lg font-semibold">{{ policyStatsSummary.blockedCount || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small" :bordered="false">
          <div class="text-xs text-gray-500">放行</div>
          <div class="text-lg font-semibold">{{ policyStatsSummary.allowedCount || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small" :bordered="false">
          <div class="text-xs text-gray-500">疑似误报</div>
          <div class="text-lg font-semibold">{{ policyStatsSummary.suspectedFalsePositiveCount || 0 }}</div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small" :bordered="false">
          <div class="text-xs text-gray-500">拦截率</div>
          <div class="text-lg font-semibold">{{ formatRatePercent(policyStatsSummary.blockRate) }}</div>
        </n-card>
      </n-gi>
    </n-grid>

    <div class="mt-3 text-xs text-gray-500">
      统计区间：{{ policyStatsRange.startTime || '-' }} ~ {{ policyStatsRange.endTime || '-' }}，粒度 {{ policyStatsRange.intervalSec || 0 }} 秒
    </div>
    <div v-if="policyStatsPreviousSnapshot" class="mt-1 text-xs text-gray-500">对比基线：{{ policyStatsPreviousSnapshot.capturedAt }}</div>
    <div class="mt-1 text-xs text-gray-500">
      下钻过滤：Host={{ policyStatsQuery.host || '-' }} / Path={{ policyStatsQuery.path || '-' }} / Method={{ policyStatsQuery.method || '-' }}
    </div>
    <div class="mt-1 text-xs text-gray-500">下钻顺序：先点 Top Host，再点 Top Path，最后点 Top Method。</div>
    <div class="mt-2 flex flex-wrap gap-2 items-center">
      <span class="text-xs text-gray-500">当前下钻标签：</span>
      <n-tag v-if="policyStatsQuery.host" closable size="small" @close="() => clearPolicyStatsDrillLevel('host')">
        Host: {{ policyStatsQuery.host }}
      </n-tag>
      <n-tag v-if="policyStatsQuery.path" closable size="small" type="info" @close="() => clearPolicyStatsDrillLevel('path')">
        Path: {{ policyStatsQuery.path }}
      </n-tag>
      <n-tag v-if="policyStatsQuery.method" closable size="small" type="warning" @close="() => clearPolicyStatsDrillLevel('method')">
        Method: {{ policyStatsQuery.method }}
      </n-tag>
      <span v-if="!hasPolicyStatsDrillFilters" class="text-xs text-gray-400">-</span>
    </div>

    <n-card :bordered="false" size="small" class="mt-3">
      <div class="text-sm font-semibold mb-2">命中趋势</div>
      <n-data-table
        :columns="policyStatsTrendColumns"
        :data="policyStatsTrend"
        :loading="policyStatsLoading"
        :pagination="false"
        :row-key="row => row.time"
        :max-height="260"
        class="min-h-120px"
      />
    </n-card>

    <n-card :bordered="false" size="small" class="mt-3">
      <div class="text-sm font-semibold mb-2">策略命中统计</div>
      <n-data-table
        :columns="policyStatsColumns"
        :data="policyStatsTable"
        :loading="policyStatsLoading"
        :pagination="false"
        :row-key="row => row.policyId"
        :max-height="320"
        class="min-h-160px"
      />
    </n-card>

    <n-card :bordered="false" size="small" class="mt-3">
      <div class="text-sm font-semibold mb-2">人工误报反馈（当前筛选口径）</div>
      <n-data-table
        remote
        :columns="policyFeedbackColumns"
        :data="policyFeedbackTable"
        :loading="policyFeedbackLoading"
        :pagination="policyFeedbackPagination"
        :checked-row-keys="policyFeedbackCheckedRowKeysInPage"
        :row-key="row => row.id"
        :max-height="300"
        class="min-h-140px"
        :scroll-x="1800"
        @update:checked-row-keys="handlePolicyFeedbackCheckedRowKeysChange"
        @update:page="handlePolicyFeedbackPageChange"
        @update:page-size="handlePolicyFeedbackPageSizeChange"
      />
    </n-card>

    <n-grid cols="3" x-gap="12" y-gap="12" class="mt-3">
      <n-gi>
        <n-card :bordered="false" size="small">
          <div class="text-sm font-semibold mb-2 flex items-center gap-2">
            <span>Top Host</span>
            <n-tooltip trigger="hover">
              <template #trigger>
                <span class="inline-flex items-center text-green-600">
                  <icon-carbon-unlocked />
                </span>
              </template>
              {{ policyStatsDrillHint('host') }}
            </n-tooltip>
            <n-tag size="small" type="success" :bordered="false">{{ policyStatsDrillStatusLabel('host') }}</n-tag>
          </div>
          <n-data-table
            :columns="policyStatsDimensionColumns"
            :data="policyStatsTopHosts"
            :loading="policyStatsLoading"
            :pagination="false"
            :row-props="buildPolicyStatsDimensionRowProps('host')"
            :row-key="row => `host-${row.key}`"
            :max-height="260"
            class="min-h-120px"
          />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" size="small">
          <div class="text-sm font-semibold mb-2 flex items-center gap-2">
            <span>Top Path</span>
            <n-tooltip trigger="hover">
              <template #trigger>
                <span class="inline-flex items-center" :class="isPolicyStatsDrillUnlocked('path') ? 'text-green-600' : 'text-gray-400'">
                  <icon-carbon-unlocked v-if="isPolicyStatsDrillUnlocked('path')" />
                  <icon-carbon-locked v-else />
                </span>
              </template>
              {{ policyStatsDrillHint('path') }}
            </n-tooltip>
            <n-tag size="small" :type="isPolicyStatsDrillUnlocked('path') ? 'success' : 'default'" :bordered="false">
              {{ policyStatsDrillStatusLabel('path') }}
            </n-tag>
          </div>
          <n-data-table
            :columns="policyStatsDimensionColumns"
            :data="policyStatsTopPaths"
            :loading="policyStatsLoading"
            :pagination="false"
            :row-props="buildPolicyStatsDimensionRowProps('path')"
            :row-key="row => `path-${row.key}`"
            :max-height="260"
            class="min-h-120px"
          />
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" size="small">
          <div class="text-sm font-semibold mb-2 flex items-center gap-2">
            <span>Top Method</span>
            <n-tooltip trigger="hover">
              <template #trigger>
                <span class="inline-flex items-center" :class="isPolicyStatsDrillUnlocked('method') ? 'text-green-600' : 'text-gray-400'">
                  <icon-carbon-unlocked v-if="isPolicyStatsDrillUnlocked('method')" />
                  <icon-carbon-locked v-else />
                </span>
              </template>
              {{ policyStatsDrillHint('method') }}
            </n-tooltip>
            <n-tag size="small" :type="isPolicyStatsDrillUnlocked('method') ? 'success' : 'default'" :bordered="false">
              {{ policyStatsDrillStatusLabel('method') }}
            </n-tag>
          </div>
          <n-data-table
            :columns="policyStatsDimensionColumns"
            :data="policyStatsTopMethods"
            :loading="policyStatsLoading"
            :pagination="false"
            :row-props="buildPolicyStatsDimensionRowProps('method')"
            :row-key="row => `method-${row.key}`"
            :max-height="260"
            class="min-h-120px"
          />
        </n-card>
      </n-gi>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type {
  WafPolicyStatsDimensionItem,
  WafPolicyStatsItem,
  WafPolicyStatsTrendItem,
  WafPolicyFalsePositiveFeedbackItem
} from '@/service/api/caddy';

type DrillLevel = 'host' | 'path' | 'method';

defineProps<{
  policyStatsQuery: {
    policyId: number | '' | null;
    window: '1h' | '6h' | '24h' | '7d';
    intervalSec: number;
    topN: number;
    host: string;
    path: string;
    method: string;
  };
  policyStatsPolicyOptions: Array<{ label: string; value: number | '' }>;
  observeWindowOptions: Array<{ label: string; value: string }>;
  policyStatsLoading: boolean;
  fetchPolicyStats: () => void | Promise<void>;
  resetPolicyStatsQuery: () => void;
  hasPolicyStatsDrillFilters: boolean;
  clearPolicyStatsDrillFilters: () => void;

  policyFeedbackStatusFilter: '' | 'pending' | 'confirmed' | 'resolved';
  policyFeedbackStatusFilterOptions: Array<{ label: string; value: string }>;
  setPolicyFeedbackStatusFilter: (value: '' | 'pending' | 'confirmed' | 'resolved') => void;
  policyFeedbackAssigneeFilter: string;
  setPolicyFeedbackAssigneeFilter: (value: string) => void;
  policyFeedbackSlaStatusFilter: 'all' | 'normal' | 'overdue' | 'resolved';
  policyFeedbackSlaStatusOptions: Array<{ label: string; value: string }>;
  setPolicyFeedbackSlaStatusFilter: (value: 'all' | 'normal' | 'overdue' | 'resolved') => void;
  handlePolicyFeedbackStatusFilterChange: () => void;
  openPolicyFeedbackModal: () => void;
  openPolicyFeedbackBatchProcessModal: () => void;
  hasPolicyFeedbackSelection: boolean;
  policyFeedbackCheckedRowKeys: number[];
  policyFeedbackLoading: boolean;
  fetchPolicyFalsePositiveFeedbacks: () => void | Promise<void>;

  handleCopyPolicyStatsLink: () => void | Promise<void>;
  handleExportPolicyStatsCompareCsv: () => void;
  handleExportPolicyStatsCsv: () => void;

  policyStatsSummary: WafPolicyStatsItem;
  policyStatsRange: { startTime: string; endTime: string; intervalSec: number };
  policyStatsPreviousSnapshot: { capturedAt: string } | null;
  formatRatePercent: (value: number) => string;

  clearPolicyStatsDrillLevel: (level: DrillLevel) => void;
  policyStatsTrendColumns: DataTableColumns<WafPolicyStatsTrendItem>;
  policyStatsTrend: WafPolicyStatsTrendItem[];
  policyStatsColumns: DataTableColumns<WafPolicyStatsItem>;
  policyStatsTable: WafPolicyStatsItem[];

  policyFeedbackColumns: DataTableColumns<WafPolicyFalsePositiveFeedbackItem>;
  policyFeedbackTable: WafPolicyFalsePositiveFeedbackItem[];
  policyFeedbackPagination: PaginationProps;
  policyFeedbackCheckedRowKeysInPage: number[];
  handlePolicyFeedbackCheckedRowKeysChange: (rowKeys: Array<string | number>) => void;
  handlePolicyFeedbackPageChange: (page: number) => void;
  handlePolicyFeedbackPageSizeChange: (pageSize: number) => void;

  policyStatsDrillHint: (level: DrillLevel) => string;
  policyStatsDrillStatusLabel: (level: DrillLevel) => string;
  isPolicyStatsDrillUnlocked: (level: DrillLevel) => boolean;
  policyStatsDimensionColumns: DataTableColumns<WafPolicyStatsDimensionItem>;
  policyStatsTopHosts: WafPolicyStatsDimensionItem[];
  policyStatsTopPaths: WafPolicyStatsDimensionItem[];
  policyStatsTopMethods: WafPolicyStatsDimensionItem[];
  buildPolicyStatsDimensionRowProps: (level: DrillLevel) => (row: WafPolicyStatsDimensionItem) => Record<string, unknown>;
}>();
</script>
