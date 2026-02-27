import { reactive, ref } from 'vue';
import type { FormInst, PaginationProps } from 'naive-ui';
import {
  activateWafRelease,
  clearWafJobs,
  clearWafReleases,
  fetchWafJobList,
  fetchWafReleaseList,
  rollbackWafRelease,
  type WafJobItem,
  type WafJobStatus,
  type WafReleaseItem,
  type WafReleaseStatus
} from '@/service/api/caddy';

type MessageApi = {
  success: (content: string) => void;
};

type DialogApi = {
  warning: (options: {
    title?: string;
    content?: string;
    positiveText?: string;
    negativeText?: string;
    onPositiveClick?: () => void | Promise<void>;
  }) => void;
};

interface UseWafReleaseJobOptions {
  message: MessageApi;
  dialog: DialogApi;
  ensureSourceNamesByIds?: (sourceIds: number[]) => Promise<void>;
  ensureUserNamesByIds?: (userIds: Array<number | string>) => Promise<void>;
}

export function useWafReleaseJob(options: UseWafReleaseJobOptions) {
  const { message, dialog, ensureSourceNamesByIds, ensureUserNamesByIds } = options;

  const releaseQuery = reactive({
    status: '' as '' | WafReleaseStatus
  });

  const releaseLoading = ref(false);
  const releaseTable = ref<WafReleaseItem[]>([]);
  const releasePagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 20,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100]
  });

  const rollbackModalVisible = ref(false);
  const rollbackSubmitting = ref(false);
  const rollbackFormRef = ref<FormInst | null>(null);
  const rollbackForm = reactive({
    target: 'last_good' as 'last_good' | 'version',
    version: ''
  });

  const jobQuery = reactive({
    status: '' as '' | WafJobStatus,
    action: ''
  });

  const jobLoading = ref(false);
  const jobTable = ref<WafJobItem[]>([]);
  const jobPagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 20,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100]
  });

  async function fetchReleases() {
    releaseLoading.value = true;
    try {
      const { data, error } = await fetchWafReleaseList({
        page: releasePagination.page as number,
        pageSize: releasePagination.pageSize as number,
        kind: 'crs',
        status: releaseQuery.status
      });
      if (!error && data) {
        const list = data.list || [];
        await ensureSourceNamesByIds?.(list.map(item => Number(item.sourceId || 0)));
        releaseTable.value = list;
        releasePagination.itemCount = data.total || 0;
      }
    } finally {
      releaseLoading.value = false;
    }
  }

  function resetReleaseQuery() {
    releaseQuery.status = '';
    releasePagination.page = 1;
    fetchReleases();
  }

  function handleReleasePageChange(page: number) {
    releasePagination.page = page;
    fetchReleases();
  }

  function handleReleasePageSizeChange(pageSize: number) {
    releasePagination.pageSize = pageSize;
    releasePagination.page = 1;
    fetchReleases();
  }

  function handleActivateRelease(row: WafReleaseItem) {
    dialog.warning({
      title: '激活确认',
      content: `确认激活版本 ${row.version} 吗？`,
      positiveText: '确认',
      negativeText: '取消',
      async onPositiveClick() {
        const { error } = await activateWafRelease(row.id);
        if (!error) {
          message.success('激活已提交');
          fetchReleases();
          fetchJobs();
        }
      }
    });
  }

  function handleClearReleases() {
    dialog.warning({
      title: '清空确认',
      content: '将清空版本发布管理中所有非激活的 CRS 版本（含文件目录），确认继续？',
      positiveText: '确认清空',
      negativeText: '取消',
      async onPositiveClick() {
        const { error } = await clearWafReleases({ kind: 'crs' });
        if (!error) {
          message.success('已清空非激活版本');
          fetchReleases();
          fetchJobs();
        }
      }
    });
  }

  function openRollbackModal() {
    rollbackForm.target = 'last_good';
    rollbackForm.version = '';
    rollbackModalVisible.value = true;
  }

  async function handleSubmitRollback() {
    await rollbackFormRef.value?.validate();
    rollbackSubmitting.value = true;
    try {
      const payload =
        rollbackForm.target === 'version'
          ? { target: 'version' as const, version: rollbackForm.version.trim() }
          : { target: 'last_good' as const };

      const { error } = await rollbackWafRelease(payload);
      if (!error) {
        message.success('回滚任务已提交');
        rollbackModalVisible.value = false;
        fetchReleases();
        fetchJobs();
      }
    } finally {
      rollbackSubmitting.value = false;
    }
  }

  async function fetchJobs() {
    jobLoading.value = true;
    try {
      const { data, error } = await fetchWafJobList({
        page: jobPagination.page as number,
        pageSize: jobPagination.pageSize as number,
        status: jobQuery.status,
        action: jobQuery.action || undefined
      });
      if (!error && data) {
        const list = data.list || [];
        await ensureSourceNamesByIds?.(list.map(item => Number(item.sourceId || 0)));
        await ensureUserNamesByIds?.(list.map(item => item.operator));
        jobTable.value = list;
        jobPagination.itemCount = data.total || 0;
      }
    } finally {
      jobLoading.value = false;
    }
  }

  function resetJobQuery() {
    jobQuery.status = '';
    jobQuery.action = '';
    jobPagination.page = 1;
    fetchJobs();
  }

  function handleJobPageChange(page: number) {
    jobPagination.page = page;
    fetchJobs();
  }

  function handleJobPageSizeChange(pageSize: number) {
    jobPagination.pageSize = pageSize;
    jobPagination.page = 1;
    fetchJobs();
  }

  function handleClearJobs() {
    dialog.warning({
      title: '清空确认',
      content: '将清空全部任务日志记录，确认继续？',
      positiveText: '确认清空',
      negativeText: '取消',
      async onPositiveClick() {
        const { error } = await clearWafJobs();
        if (!error) {
          message.success('任务日志已清空');
          fetchJobs();
        }
      }
    });
  }

  return {
    releaseQuery,
    releaseLoading,
    releaseTable,
    releasePagination,
    rollbackModalVisible,
    rollbackSubmitting,
    rollbackFormRef,
    rollbackForm,
    jobQuery,
    jobLoading,
    jobTable,
    jobPagination,
    fetchReleases,
    resetReleaseQuery,
    handleReleasePageChange,
    handleReleasePageSizeChange,
    handleActivateRelease,
    handleClearReleases,
    openRollbackModal,
    handleSubmitRollback,
    fetchJobs,
    resetJobQuery,
    handleJobPageChange,
    handleJobPageSizeChange,
    handleClearJobs
  };
}
