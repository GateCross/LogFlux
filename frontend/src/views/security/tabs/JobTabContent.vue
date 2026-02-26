<template>
  <div>
    <div class="mb-3 flex flex-wrap gap-2 items-center">
      <n-select v-model:value="jobQuery.status" :options="jobStatusOptions" clearable placeholder="状态" class="w-160px" />
      <n-select v-model:value="jobQuery.action" :options="jobActionOptions" clearable placeholder="动作" class="w-160px" />
      <n-button type="primary" @click="fetchJobs">
        <template #icon>
          <icon-carbon-search />
        </template>
        查询
      </n-button>
      <n-button @click="resetJobQuery">重置</n-button>
      <n-button type="success" @click="refreshCurrentTab">刷新</n-button>
      <n-button type="error" @click="handleClearJobs">清空任务日志</n-button>
    </div>

    <n-data-table
      remote
      :columns="jobColumns"
      :data="jobTable"
      :loading="jobLoading"
      :pagination="jobPagination"
      :row-key="row => row.id"
      :max-height="tableFixedHeight"
      class="min-h-260px"
      :scroll-x="1500"
      :resizable="true"
      @update:page="handleJobPageChange"
      @update:page-size="handleJobPageSizeChange"
    />
  </div>
</template>

<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafJobItem } from '@/service/api/caddy';

defineProps<{
  jobQuery: { status: string; action: string };
  jobStatusOptions: Array<{ label: string; value: string }>;
  jobActionOptions: Array<{ label: string; value: string }>;
  fetchJobs: () => void | Promise<void>;
  resetJobQuery: () => void;
  refreshCurrentTab: () => void;
  handleClearJobs: () => void;

  jobColumns: DataTableColumns<WafJobItem>;
  jobTable: WafJobItem[];
  jobLoading: boolean;
  jobPagination: PaginationProps;
  tableFixedHeight: number;
  handleJobPageChange: (page: number) => void;
  handleJobPageSizeChange: (pageSize: number) => void;
}>();
</script>
