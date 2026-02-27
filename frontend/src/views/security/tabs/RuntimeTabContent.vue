<template>
  <div>
    <n-alert type="warning" :show-icon="true" class="mb-3">
      建议先使用 DetectionOnly（仅检测）观察，再切换到 On（阻断）。On 模式发布会触发二次确认。
    </n-alert>

    <div class="mb-3 flex flex-wrap gap-2 items-center">
      <n-input v-model:value="policyQuery.name" placeholder="按策略名称搜索" clearable class="w-220px" @keyup.enter="fetchPolicies" />
      <n-button type="primary" @click="fetchPolicies">
        <template #icon>
          <icon-carbon-search />
        </template>
        查询
      </n-button>
      <n-button @click="resetPolicyQuery">重置</n-button>
      <n-button type="primary" @click="handleAddPolicy">
        <template #icon>
          <icon-ic-round-plus />
        </template>
        新增策略
      </n-button>
    </div>

    <n-data-table
      remote
      :columns="policyColumns"
      :data="policyTable"
      :loading="policyLoading"
      :pagination="policyPagination"
      :row-key="row => row.id"
      :max-height="tableFixedHeight"
      class="min-h-260px"
      :scroll-x="1700"
      :resizable="true"
      @update:page="handlePolicyPageChange"
      @update:page-size="handlePolicyPageSizeChange"
    />

    <n-card :bordered="false" size="small" class="mt-3">
      <div class="text-sm font-semibold mb-2">策略指令预览 {{ policyPreviewPolicyName ? `(${policyPreviewPolicyName})` : '' }}</div>
      <n-spin :show="policyPreviewLoading">
        <n-input
          :value="policyPreviewDirectives"
          type="textarea"
          :autosize="{ minRows: 8, maxRows: 14 }"
          readonly
          placeholder="点击策略列表中的“预览”查看渲染后的 Coraza directives"
        />
      </n-spin>
    </n-card>

    <div class="mt-3 text-sm font-semibold">最近发布记录</div>
    <n-data-table
      remote
      class="mt-2 min-h-220px"
      :columns="policyRevisionColumns"
      :data="policyRevisionTable"
      :loading="policyRevisionLoading"
      :pagination="policyRevisionPagination"
      :row-key="row => row.id"
      :scroll-x="1100"
      :resizable="true"
      @update:page="handlePolicyRevisionPageChange"
      @update:page-size="handlePolicyRevisionPageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafPolicyItem, WafPolicyRevisionItem } from '@/service/api/caddy';

defineProps<{
  policyQuery: { name: string };
  policyColumns: DataTableColumns<WafPolicyItem>;
  policyTable: WafPolicyItem[];
  policyLoading: boolean;
  policyPagination: PaginationProps;
  tableFixedHeight: number;
  fetchPolicies: () => void | Promise<void>;
  resetPolicyQuery: () => void;
  handleAddPolicy: () => void;
  handlePolicyPageChange: (page: number) => void;
  handlePolicyPageSizeChange: (pageSize: number) => void;

  policyPreviewPolicyName: string;
  policyPreviewLoading: boolean;
  policyPreviewDirectives: string;

  policyRevisionColumns: DataTableColumns<WafPolicyRevisionItem>;
  policyRevisionTable: WafPolicyRevisionItem[];
  policyRevisionLoading: boolean;
  policyRevisionPagination: PaginationProps;
  handlePolicyRevisionPageChange: (page: number) => void;
  handlePolicyRevisionPageSizeChange: (pageSize: number) => void;
}>();
</script>
