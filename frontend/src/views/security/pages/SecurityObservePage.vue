<script setup lang="ts">
import { computed } from 'vue';
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type {
  WafPolicyFalsePositiveFeedbackItem,
  WafPolicyStatsDimensionItem,
  WafPolicyStatsItem,
  WafPolicyStatsTrendItem
} from '@/service/api/caddy-observe';

type DrillLevel = 'host' | 'path' | 'method';
type ObserveView = 'analysis' | 'feedback';

const props = defineProps<{
  activeView: ObserveView;
  setActiveView: (view: ObserveView) => void;
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
  policyStatsDrillHint: (level: DrillLevel) => string;
  policyStatsDrillStatusLabel: (level: DrillLevel) => string;
  isPolicyStatsDrillUnlocked: (level: DrillLevel) => boolean;
  policyStatsDimensionColumns: DataTableColumns<WafPolicyStatsDimensionItem>;
  policyStatsTopHosts: WafPolicyStatsDimensionItem[];
  policyStatsTopPaths: WafPolicyStatsDimensionItem[];
  policyStatsTopMethods: WafPolicyStatsDimensionItem[];
  buildPolicyStatsDimensionRowProps: (
    level: DrillLevel
  ) => (row: WafPolicyStatsDimensionItem) => Record<string, unknown>;
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
  policyFeedbackColumns: DataTableColumns<WafPolicyFalsePositiveFeedbackItem>;
  policyFeedbackTable: WafPolicyFalsePositiveFeedbackItem[];
  policyFeedbackPagination: PaginationProps;
  policyFeedbackCheckedRowKeysInPage: number[];
  handlePolicyFeedbackCheckedRowKeysChange: (rowKeys: Array<string | number>) => void;
  handlePolicyFeedbackPageChange: (page: number) => void;
  handlePolicyFeedbackPageSizeChange: (pageSize: number) => void;
}>();

const policyStatsQueryModel = computed({
  get: () => props.policyStatsQuery,
  set: () => undefined
});
</script>

<template>
  <div class="flex flex-col gap-3">
    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">观测与处置</div>
          <div class="mt-1 text-xs text-gray-500">
            将效果分析与误报处置分区，降低工具栏密度，同时保持 URL 筛选恢复与批量处理链路。
          </div>
        </div>
        <div class="flex gap-2">
          <NButton :type="activeView === 'analysis' ? 'primary' : 'default'" @click="setActiveView('analysis')">
            效果分析
          </NButton>
          <NButton :type="activeView === 'feedback' ? 'primary' : 'default'" @click="setActiveView('feedback')">
            误报处置
          </NButton>
        </div>
      </div>
    </NCard>

    <template v-if="activeView === 'analysis'">
      <NAlert type="info" :show-icon="true" class="rounded-8px">
        统计口径基于策略绑定作用域与请求日志；“疑似误报”当前为启发式指标（安全端点被拦截），用于辅助调优参考。
      </NAlert>

      <NCard :bordered="false" class="rounded-8px shadow-sm">
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <NSelect
            v-model:value="policyStatsQueryModel.policyId"
            :options="policyStatsPolicyOptions"
            clearable
            placeholder="策略范围"
            class="w-240px"
          />
          <NSelect v-model:value="policyStatsQueryModel.window" :options="observeWindowOptions" class="w-180px" />
          <NInputNumber
            v-model:value="policyStatsQueryModel.intervalSec"
            :show-button="false"
            :min="60"
            :max="86400"
            placeholder="趋势粒度（秒）"
            class="w-180px"
          />
          <NInputNumber
            v-model:value="policyStatsQueryModel.topN"
            :show-button="false"
            :min="1"
            :max="50"
            placeholder="TopN"
            class="w-120px"
          />
          <NButton type="primary" :loading="policyStatsLoading" @click="fetchPolicyStats">
            <template #icon>
              <icon-carbon-search />
            </template>
            查询
          </NButton>
          <NButton @click="resetPolicyStatsQuery">重置</NButton>
          <NButton :disabled="!hasPolicyStatsDrillFilters" @click="clearPolicyStatsDrillFilters">清空下钻</NButton>
          <NButton secondary @click="handleCopyPolicyStatsLink">
            <template #icon>
              <icon-carbon-link />
            </template>
            复制筛选链接
          </NButton>
          <NButton secondary :disabled="!policyStatsPreviousSnapshot" @click="handleExportPolicyStatsCompareCsv">
            导出对比 CSV
          </NButton>
          <NButton secondary @click="handleExportPolicyStatsCsv">导出 CSV</NButton>
        </div>

        <NGrid cols="5" x-gap="12" y-gap="10">
          <NGi>
            <NCard size="small" :bordered="false">
              <div class="text-xs text-gray-500">命中</div>
              <div class="text-lg font-semibold">
                {{ policyStatsSummary.hitCount || 0 }}
              </div>
            </NCard>
          </NGi>
          <NGi>
            <NCard size="small" :bordered="false">
              <div class="text-xs text-gray-500">拦截</div>
              <div class="text-lg font-semibold">
                {{ policyStatsSummary.blockedCount || 0 }}
              </div>
            </NCard>
          </NGi>
          <NGi>
            <NCard size="small" :bordered="false">
              <div class="text-xs text-gray-500">放行</div>
              <div class="text-lg font-semibold">
                {{ policyStatsSummary.allowedCount || 0 }}
              </div>
            </NCard>
          </NGi>
          <NGi>
            <NCard size="small" :bordered="false">
              <div class="text-xs text-gray-500">疑似误报</div>
              <div class="text-lg font-semibold">
                {{ policyStatsSummary.suspectedFalsePositiveCount || 0 }}
              </div>
            </NCard>
          </NGi>
          <NGi>
            <NCard size="small" :bordered="false">
              <div class="text-xs text-gray-500">拦截率</div>
              <div class="text-lg font-semibold">
                {{ formatRatePercent(policyStatsSummary.blockRate) }}
              </div>
            </NCard>
          </NGi>
        </NGrid>

        <div class="mt-3 text-xs text-gray-500">
          统计区间：{{ policyStatsRange.startTime || '-' }} ~ {{ policyStatsRange.endTime || '-' }}，粒度
          {{ policyStatsRange.intervalSec || 0 }} 秒
        </div>
        <div v-if="policyStatsPreviousSnapshot" class="mt-1 text-xs text-gray-500">
          对比基线：{{ policyStatsPreviousSnapshot.capturedAt }}
        </div>
        <div class="mt-1 text-xs text-gray-500">
          下钻过滤：Host={{ policyStatsQuery.host || '-' }} / Path={{ policyStatsQuery.path || '-' }} / Method={{
            policyStatsQuery.method || '-'
          }}
        </div>
        <div class="mt-1 text-xs text-gray-500">下钻顺序：先点 Top Host，再点 Top Path，最后点 Top Method。</div>
        <div class="mt-2 flex flex-wrap items-center gap-2">
          <span class="text-xs text-gray-500">当前下钻标签：</span>
          <NTag v-if="policyStatsQuery.host" closable size="small" @close="() => clearPolicyStatsDrillLevel('host')">
            Host: {{ policyStatsQuery.host }}
          </NTag>
          <NTag
            v-if="policyStatsQuery.path"
            closable
            size="small"
            type="info"
            @close="() => clearPolicyStatsDrillLevel('path')"
          >
            Path: {{ policyStatsQuery.path }}
          </NTag>
          <NTag
            v-if="policyStatsQuery.method"
            closable
            size="small"
            type="warning"
            @close="() => clearPolicyStatsDrillLevel('method')"
          >
            Method: {{ policyStatsQuery.method }}
          </NTag>
          <span v-if="!hasPolicyStatsDrillFilters" class="text-xs text-gray-400">-</span>
        </div>
      </NCard>

      <NCard :bordered="false" size="small" class="rounded-8px shadow-sm">
        <div class="mb-2 text-sm font-semibold">命中趋势</div>
        <NDataTable
          :columns="policyStatsTrendColumns"
          :data="policyStatsTrend"
          :loading="policyStatsLoading"
          :pagination="false"
          :row-key="row => row.time"
          :max-height="260"
          class="min-h-120px"
        />
      </NCard>

      <NCard :bordered="false" size="small" class="rounded-8px shadow-sm">
        <div class="mb-2 text-sm font-semibold">策略命中统计</div>
        <NDataTable
          :columns="policyStatsColumns"
          :data="policyStatsTable"
          :loading="policyStatsLoading"
          :pagination="false"
          :row-key="row => row.policyId"
          :max-height="320"
          class="min-h-160px"
        />
      </NCard>

      <NGrid cols="3" x-gap="12" y-gap="12">
        <NGi>
          <NCard :bordered="false" size="small" class="rounded-8px shadow-sm">
            <div class="mb-2 flex items-center gap-2 text-sm font-semibold">
              <span>Top Host</span>
              <NTooltip trigger="hover">
                <template #trigger>
                  <span class="inline-flex items-center text-green-600">
                    <icon-carbon-unlocked />
                  </span>
                </template>
                {{ policyStatsDrillHint('host') }}
              </NTooltip>
              <NTag size="small" type="success" :bordered="false">{{ policyStatsDrillStatusLabel('host') }}</NTag>
            </div>
            <NDataTable
              :columns="policyStatsDimensionColumns"
              :data="policyStatsTopHosts"
              :loading="policyStatsLoading"
              :pagination="false"
              :row-props="buildPolicyStatsDimensionRowProps('host')"
              :row-key="row => `host-${row.key}`"
              :max-height="260"
              class="min-h-120px"
            />
          </NCard>
        </NGi>
        <NGi>
          <NCard :bordered="false" size="small" class="rounded-8px shadow-sm">
            <div class="mb-2 flex items-center gap-2 text-sm font-semibold">
              <span>Top Path</span>
              <NTooltip trigger="hover">
                <template #trigger>
                  <span
                    class="inline-flex items-center"
                    :class="isPolicyStatsDrillUnlocked('path') ? 'text-green-600' : 'text-gray-400'"
                  >
                    <icon-carbon-unlocked v-if="isPolicyStatsDrillUnlocked('path')" />
                    <icon-carbon-locked v-else />
                  </span>
                </template>
                {{ policyStatsDrillHint('path') }}
              </NTooltip>
              <NTag size="small" :type="isPolicyStatsDrillUnlocked('path') ? 'success' : 'default'" :bordered="false">
                {{ policyStatsDrillStatusLabel('path') }}
              </NTag>
            </div>
            <NDataTable
              :columns="policyStatsDimensionColumns"
              :data="policyStatsTopPaths"
              :loading="policyStatsLoading"
              :pagination="false"
              :row-props="buildPolicyStatsDimensionRowProps('path')"
              :row-key="row => `path-${row.key}`"
              :max-height="260"
              class="min-h-120px"
            />
          </NCard>
        </NGi>
        <NGi>
          <NCard :bordered="false" size="small" class="rounded-8px shadow-sm">
            <div class="mb-2 flex items-center gap-2 text-sm font-semibold">
              <span>Top Method</span>
              <NTooltip trigger="hover">
                <template #trigger>
                  <span
                    class="inline-flex items-center"
                    :class="isPolicyStatsDrillUnlocked('method') ? 'text-green-600' : 'text-gray-400'"
                  >
                    <icon-carbon-unlocked v-if="isPolicyStatsDrillUnlocked('method')" />
                    <icon-carbon-locked v-else />
                  </span>
                </template>
                {{ policyStatsDrillHint('method') }}
              </NTooltip>
              <NTag size="small" :type="isPolicyStatsDrillUnlocked('method') ? 'success' : 'default'" :bordered="false">
                {{ policyStatsDrillStatusLabel('method') }}
              </NTag>
            </div>
            <NDataTable
              :columns="policyStatsDimensionColumns"
              :data="policyStatsTopMethods"
              :loading="policyStatsLoading"
              :pagination="false"
              :row-props="buildPolicyStatsDimensionRowProps('method')"
              :row-key="row => `method-${row.key}`"
              :max-height="260"
              class="min-h-120px"
            />
          </NCard>
        </NGi>
      </NGrid>
    </template>

    <NCard v-else :bordered="false" class="rounded-8px shadow-sm">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <NSelect
          :value="policyFeedbackStatusFilter"
          :options="policyFeedbackStatusFilterOptions"
          placeholder="反馈状态"
          class="w-160px"
          @update:value="
            value => {
              setPolicyFeedbackStatusFilter(value as '' | 'pending' | 'confirmed' | 'resolved');
              handlePolicyFeedbackStatusFilterChange();
            }
          "
        />
        <NInput
          :value="policyFeedbackAssigneeFilter"
          clearable
          placeholder="责任人"
          class="w-160px"
          @update:value="value => setPolicyFeedbackAssigneeFilter(value)"
          @keyup.enter="handlePolicyFeedbackStatusFilterChange"
        />
        <NSelect
          :value="policyFeedbackSlaStatusFilter"
          :options="policyFeedbackSlaStatusOptions"
          class="w-160px"
          @update:value="
            value => {
              setPolicyFeedbackSlaStatusFilter(value as 'all' | 'normal' | 'overdue' | 'resolved');
              handlePolicyFeedbackStatusFilterChange();
            }
          "
        />
        <NButton type="warning" secondary @click="openPolicyFeedbackModal">标记误报</NButton>
        <NButton
          type="warning"
          secondary
          :disabled="!hasPolicyFeedbackSelection"
          @click="openPolicyFeedbackBatchProcessModal"
        >
          批量处理（{{ policyFeedbackCheckedRowKeys.length }}）
        </NButton>
        <NButton secondary :loading="policyFeedbackLoading" @click="fetchPolicyFalsePositiveFeedbacks">
          刷新反馈
        </NButton>
      </div>

      <NAlert type="info" :show-icon="true" class="mb-3">
        当前处置视图沿用分析视图的策略范围与下钻条件，可直接从统计上下文进入反馈处理和 exclusion 草稿生成。
      </NAlert>

      <NDataTable
        remote
        :columns="policyFeedbackColumns"
        :data="policyFeedbackTable"
        :loading="policyFeedbackLoading"
        :pagination="policyFeedbackPagination"
        :checked-row-keys="policyFeedbackCheckedRowKeysInPage"
        :row-key="row => row.id"
        :max-height="420"
        class="min-h-200px"
        :scroll-x="1800"
        @update:checked-row-keys="handlePolicyFeedbackCheckedRowKeysChange"
        @update:page="handlePolicyFeedbackPageChange"
        @update:page-size="handlePolicyFeedbackPageSizeChange"
      />
    </NCard>
  </div>
</template>
