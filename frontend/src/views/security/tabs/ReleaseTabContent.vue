<template>
  <div>
    <div class="mb-3 flex flex-wrap gap-2 items-center">
      <n-select v-model:value="releaseQuery.status" :options="releaseStatusOptions" clearable placeholder="状态" class="w-160px" />
      <n-button type="primary" @click="fetchReleases">
        <template #icon>
          <icon-carbon-search />
        </template>
        查询
      </n-button>
      <n-button @click="resetReleaseQuery">重置</n-button>
      <n-button type="warning" @click="openRollbackModal">回滚到历史版本</n-button>
      <n-button type="error" @click="handleClearReleases">清空非激活版本</n-button>
    </div>

    <n-data-table
      remote
      :columns="releaseColumns"
      :data="releaseTable"
      :loading="releaseLoading"
      :pagination="releasePagination"
      :row-key="row => row.id"
      :max-height="tableFixedHeight"
      class="min-h-260px"
      @update:page="handleReleasePageChange"
      @update:page-size="handleReleasePageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafReleaseItem } from '@/service/api/caddy';

defineProps<{
  releaseQuery: { status: string };
  releaseStatusOptions: Array<{ label: string; value: string }>;
  fetchReleases: () => void | Promise<void>;
  resetReleaseQuery: () => void;
  openRollbackModal: () => void;
  handleClearReleases: () => void;

  releaseColumns: DataTableColumns<WafReleaseItem>;
  releaseTable: WafReleaseItem[];
  releaseLoading: boolean;
  releasePagination: PaginationProps;
  tableFixedHeight: number;
  handleReleasePageChange: (page: number) => void;
  handleReleasePageSizeChange: (pageSize: number) => void;
}>();
</script>
