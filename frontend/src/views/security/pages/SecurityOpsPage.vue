<script setup lang="ts">
import type { DataTableColumns, PaginationProps } from 'naive-ui';
import type { WafJobItem, WafReleaseItem } from '@/service/api/caddy-release-job';
import JobTabContent from '../tabs/JobTabContent.vue';
import ReleaseTabContent from '../tabs/ReleaseTabContent.vue';

type OpsSection = 'release' | 'job';

defineProps<{
  activeSection: OpsSection;
  navigateToTab: (tab: OpsSection) => void | Promise<void>;
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
  jobQuery: { status: string; action: string };
  jobStatusOptions: Array<{ label: string; value: string }>;
  jobActionOptions: Array<{ label: string; value: string }>;
  fetchJobs: () => void | Promise<void>;
  resetJobQuery: () => void;
  refreshCurrentSection: () => void;
  handleClearJobs: () => void;
  jobColumns: DataTableColumns<WafJobItem>;
  jobTable: WafJobItem[];
  jobLoading: boolean;
  jobPagination: PaginationProps;
  handleJobPageChange: (page: number) => void;
  handleJobPageSizeChange: (pageSize: number) => void;
}>();
</script>

<template>
  <div class="flex flex-col gap-3">
    <NCard :bordered="false" class="rounded-8px shadow-sm">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div class="text-base font-semibold">发布运维</div>
          <div class="mt-1 text-xs text-gray-500">
            将版本发布与任务审计收束到同一运维域，保留激活、回滚、清理与执行追踪主路径。
          </div>
        </div>
        <div class="flex gap-2">
          <NButton :type="activeSection === 'release' ? 'primary' : 'default'" @click="navigateToTab('release')">
            发布管理
          </NButton>
          <NButton :type="activeSection === 'job' ? 'primary' : 'default'" @click="navigateToTab('job')">
            任务审计
          </NButton>
        </div>
      </div>
    </NCard>

    <NCard v-if="activeSection === 'release'" :bordered="false" class="rounded-8px shadow-sm">
      <ReleaseTabContent
        :release-query="releaseQuery"
        :release-status-options="releaseStatusOptions"
        :fetch-releases="fetchReleases"
        :reset-release-query="resetReleaseQuery"
        :open-rollback-modal="openRollbackModal"
        :handle-clear-releases="handleClearReleases"
        :release-columns="releaseColumns"
        :release-table="releaseTable"
        :release-loading="releaseLoading"
        :release-pagination="releasePagination"
        :table-fixed-height="tableFixedHeight"
        :handle-release-page-change="handleReleasePageChange"
        :handle-release-page-size-change="handleReleasePageSizeChange"
      />
    </NCard>

    <NCard v-else :bordered="false" class="rounded-8px shadow-sm">
      <JobTabContent
        :job-query="jobQuery"
        :job-status-options="jobStatusOptions"
        :job-action-options="jobActionOptions"
        :fetch-jobs="fetchJobs"
        :reset-job-query="resetJobQuery"
        :refresh-current-tab="refreshCurrentSection"
        :handle-clear-jobs="handleClearJobs"
        :job-columns="jobColumns"
        :job-table="jobTable"
        :job-loading="jobLoading"
        :job-pagination="jobPagination"
        :table-fixed-height="tableFixedHeight"
        :handle-job-page-change="handleJobPageChange"
        :handle-job-page-size-change="handleJobPageSizeChange"
      />
    </NCard>
  </div>
</template>
