<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafJobItem } from '@/service/api/caddy-release-job';

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

<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <NSelect
        v-model:value="jobQuery.status"
        :options="jobStatusOptions"
        clearable
        placeholder="状态"
        class="w-160px"
      />
      <NSelect
        v-model:value="jobQuery.action"
        :options="jobActionOptions"
        clearable
        placeholder="动作"
        class="w-160px"
      />
      <NButton type="primary" @click="fetchJobs">
        <template #icon>
          <icon-carbon-search />
        </template>
        查询
      </NButton>
      <NButton @click="resetJobQuery">重置</NButton>
      <NButton type="success" @click="refreshCurrentTab">刷新</NButton>
      <NButton type="error" @click="handleClearJobs">清空任务日志</NButton>
    </div>

    <NDataTable
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
