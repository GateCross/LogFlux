<template>
  <div>
    <div class="mb-3 flex flex-wrap gap-2 items-center">
      <n-input v-model:value="sourceQuery.name" placeholder="按名称搜索" clearable class="w-220px" @keyup.enter="fetchSources" />
      <n-button type="primary" @click="fetchSources">
        <template #icon>
          <icon-carbon-search />
        </template>
        查询
      </n-button>
      <n-button @click="resetSourceQuery">重置</n-button>
      <n-button type="primary" @click="handleAddSource">
        <template #icon>
          <icon-ic-round-plus />
        </template>
        新增源
      </n-button>
      <n-button type="success" @click="openUploadModal">
        <template #icon>
          <icon-carbon-cloud-upload />
        </template>
        上传规则包
      </n-button>
    </div>

    <n-data-table
      remote
      :columns="sourceColumns"
      :data="sourceTable"
      :loading="sourceLoading"
      :pagination="sourcePagination"
      :row-key="row => row.id"
      :max-height="tableFixedHeight"
      class="min-h-260px"
      @update:page="handleSourcePageChange"
      @update:page-size="handleSourcePageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafSourceItem } from '@/service/api/caddy';

defineProps<{
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
