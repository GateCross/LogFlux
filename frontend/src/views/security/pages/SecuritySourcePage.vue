<template>
  <div class="flex flex-col gap-3">
    <n-alert type="info" :show-icon="true" class="rounded-8px">
      <template #header>{{ pageTitle }}</template>
      <div>
        CRS 支持在线同步（含检查）、上传、激活与回滚；Coraza 引擎依赖 Caddy 二进制，仅提供 GitHub Release 版本检查，不支持在线替换引擎。
      </div>
    </n-alert>

    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">Coraza 引擎版本检查</div>
          <div class="mt-1 text-xs text-gray-500">用于发现 Coraza 引擎新版本并生成升级建议（需通过镜像发布流程升级）。</div>
        </div>
        <div class="flex gap-2">
          <n-button size="small" :loading="engineLoading" @click="handleRefreshEngineStatus">刷新状态</n-button>
          <n-button size="small" type="primary" :loading="engineChecking" @click="handleCheckEngine">检查上游版本</n-button>
        </div>
      </div>

      <n-grid cols="4" x-gap="12" y-gap="10" class="mt-4">
        <n-gi>
          <div class="text-xs text-gray-500">当前版本</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.currentVersion) }}</div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">最新版本</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.latestVersion) }}</div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">可升级</div>
          <div class="text-sm font-medium">
            <n-tag :type="engineStatus?.canUpgrade ? 'warning' : 'success'" :bordered="false">
              {{ engineStatus?.canUpgrade ? '是' : '否' }}
            </n-tag>
          </div>
        </n-gi>
        <n-gi>
          <div class="text-xs text-gray-500">最近检查时间</div>
          <div class="text-sm font-medium">{{ displayEngineValue(engineStatus?.checkedAt) }}</div>
        </n-gi>
      </n-grid>

      <n-alert v-if="engineUnavailable" type="warning" :show-icon="true" class="mt-4">
        当前引擎状态接口暂不可用，已切换为占位模式，请检查后端日志。
      </n-alert>
      <n-alert v-else-if="engineStatus?.message" type="info" :show-icon="true" class="mt-4">
        {{ engineStatus?.message }}
      </n-alert>
    </n-card>

    <n-card :bordered="false" class="rounded-8px shadow-sm">
      <SourceTabContent
        :source-query="sourceQuery"
        :source-columns="sourceColumns"
        :source-table="sourceTable"
        :source-loading="sourceLoading"
        :source-pagination="sourcePagination"
        :table-fixed-height="tableFixedHeight"
        :fetch-sources="fetchSources"
        :reset-source-query="resetSourceQuery"
        :handle-add-source="handleAddSource"
        :open-upload-modal="openUploadModal"
        :handle-source-page-change="handleSourcePageChange"
        :handle-source-page-size-change="handleSourcePageSizeChange"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafEngineStatusResp, WafSourceItem } from '@/service/api/caddy-source';
import SourceTabContent from '../tabs/SourceTabContent.vue';

defineProps<{
  pageTitle: string;
  engineLoading: boolean;
  engineChecking: boolean;
  engineUnavailable: boolean;
  engineStatus: WafEngineStatusResp | null;
  handleRefreshEngineStatus: () => void;
  handleCheckEngine: () => void | Promise<void>;
  displayEngineValue: (value: unknown) => string;
  sourceQuery: { name: string };
  sourceColumns: DataTableColumns<WafSourceItem>;
  sourceTable: WafSourceItem[];
  sourceLoading: boolean;
  sourcePagination: PaginationProps;
  tableFixedHeight: number;
  fetchSources: () => void | Promise<void>;
  resetSourceQuery: () => void;
  handleAddSource: () => void;
  openUploadModal: () => void;
  handleSourcePageChange: (page: number) => void;
  handleSourcePageSizeChange: (pageSize: number) => void;
}>();
</script>
