import { computed, reactive, ref, type Ref } from 'vue';
import type { FormInst, PaginationProps } from 'naive-ui';
import {
  createWafPolicy,
  deleteWafPolicy,
  fetchWafPolicyList,
  fetchWafPolicyRevisionList,
  previewWafPolicy,
  publishWafPolicy,
  rollbackWafPolicy,
  updateWafPolicy,
  validateWafPolicy,
  type WafPolicyAuditEngine,
  type WafPolicyAuditLogFormat,
  type WafPolicyEngineMode,
  type WafPolicyItem,
  type WafPolicyRevisionItem
} from '@/service/api/caddy-policy';

type MessageApi = {
  success: (content: string) => void;
  warning: (content: string) => void;
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

interface UseWafPolicyOptions {
  message: MessageApi;
  dialog: DialogApi;
  ensureUserNamesByIds?: (userIds: Array<number | string>) => Promise<void>;
  onPolicyListSynced?: () => void;
  getCurrentRevisionPolicyId?: () => number | undefined;
}

export function useWafPolicy(options: UseWafPolicyOptions) {
  const { message, dialog, ensureUserNamesByIds, onPolicyListSynced, getCurrentRevisionPolicyId } = options;

  const policyQuery = reactive({
    name: ''
  });

  const policyLoading = ref(false);
  const policyTable = ref<WafPolicyItem[]>([]);
  const policyPagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 20,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100]
  });

  const policyModalVisible = ref(false);
  const policyModalMode = ref<'add' | 'edit'>('add');
  const policySubmitting = ref(false);
  const policyFormRef = ref<FormInst | null>(null);
  const policyForm = reactive({
    id: 0,
    name: '',
    description: '',
    enabled: true,
    isDefault: false,
    engineMode: 'detectiononly' as WafPolicyEngineMode,
    auditEngine: 'relevantonly' as WafPolicyAuditEngine,
    auditLogFormat: 'json' as WafPolicyAuditLogFormat,
    auditRelevantStatus: '^(?:5|4(?!04))',
    requestBodyAccess: true,
    requestBodyLimit: 10 * 1024 * 1024,
    requestBodyNoFilesLimit: 1024 * 1024,
    config: ''
  });

  const policyModalTitle = computed(() => (policyModalMode.value === 'add' ? '新增运行策略' : '编辑运行策略'));
  const policyPreviewLoading = ref(false);
  const policyPreviewPolicyName = ref('');
  const policyPreviewDirectives = ref('');

  const policyRevisionLoading = ref(false);
  const policyRevisionTable = ref<WafPolicyRevisionItem[]>([]);
  const policyRevisionPagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 10,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50]
  });

  const crsPolicyOptions = computed(() =>
    policyTable.value.map(item => ({
      label: `${item.name}${item.isDefault ? '（默认）' : ''}`,
      value: item.id
    }))
  );

  async function fetchPolicies() {
    policyLoading.value = true;
    try {
      const { data, error } = await fetchWafPolicyList({
        page: policyPagination.page as number,
        pageSize: policyPagination.pageSize as number,
        name: policyQuery.name.trim() || undefined
      });
      if (!error && data) {
        const list = data.list || [];
        const total = data.total || 0;

        if (!policyQuery.name.trim() && total > 0 && list.length === 0 && (policyPagination.page as number) > 1) {
          policyPagination.page = 1;
          await fetchPolicies();
          return;
        }

        policyTable.value = list;
        policyPagination.itemCount = total;
        onPolicyListSynced?.();
      }
    } finally {
      policyLoading.value = false;
    }
  }

  function resetPolicyQuery() {
    policyQuery.name = '';
    policyPagination.page = 1;
    fetchPolicies();
  }

  function handlePolicyPageChange(page: number) {
    policyPagination.page = page;
    fetchPolicies();
  }

  function handlePolicyPageSizeChange(pageSize: number) {
    policyPagination.pageSize = pageSize;
    policyPagination.page = 1;
    fetchPolicies();
  }

  function resetPolicyForm() {
    policyForm.id = 0;
    policyForm.name = '';
    policyForm.description = '';
    policyForm.enabled = true;
    policyForm.isDefault = false;
    policyForm.engineMode = 'detectiononly';
    policyForm.auditEngine = 'relevantonly';
    policyForm.auditLogFormat = 'json';
    policyForm.auditRelevantStatus = '^(?:5|4(?!04))';
    policyForm.requestBodyAccess = true;
    policyForm.requestBodyLimit = 10 * 1024 * 1024;
    policyForm.requestBodyNoFilesLimit = 1024 * 1024;
    policyForm.config = '';
  }

  function handleAddPolicy() {
    policyModalMode.value = 'add';
    resetPolicyForm();
    policyModalVisible.value = true;
  }

  function handleEditPolicy(row: WafPolicyItem) {
    policyModalMode.value = 'edit';
    policyForm.id = row.id;
    policyForm.name = row.name;
    policyForm.description = row.description || '';
    policyForm.enabled = row.enabled;
    policyForm.isDefault = row.isDefault;
    policyForm.engineMode = row.engineMode;
    policyForm.auditEngine = row.auditEngine;
    policyForm.auditLogFormat = row.auditLogFormat;
    policyForm.auditRelevantStatus = row.auditRelevantStatus || '^(?:5|4(?!04))';
    policyForm.requestBodyAccess = row.requestBodyAccess;
    policyForm.requestBodyLimit = row.requestBodyLimit;
    policyForm.requestBodyNoFilesLimit = row.requestBodyNoFilesLimit;
    policyForm.config = row.config || '';
    policyModalVisible.value = true;
  }

  function buildPolicyPayload() {
    return {
      name: policyForm.name.trim(),
      description: policyForm.description.trim(),
      enabled: policyForm.enabled,
      isDefault: policyForm.isDefault,
      engineMode: policyForm.engineMode,
      auditEngine: policyForm.auditEngine,
      auditLogFormat: policyForm.auditLogFormat,
      auditRelevantStatus: policyForm.auditRelevantStatus.trim(),
      requestBodyAccess: policyForm.requestBodyAccess,
      requestBodyLimit: Number(policyForm.requestBodyLimit),
      requestBodyNoFilesLimit: Number(policyForm.requestBodyNoFilesLimit),
      config: policyForm.config.trim()
    };
  }

  async function handleSubmitPolicy() {
    await policyFormRef.value?.validate();
    policySubmitting.value = true;
    try {
      const payload = buildPolicyPayload();
      const request =
        policyModalMode.value === 'add' ? createWafPolicy(payload) : updateWafPolicy(policyForm.id, payload);

      const { error } = await request;
      if (!error) {
        message.success(policyModalMode.value === 'add' ? '策略创建成功' : '策略更新成功');
        policyModalVisible.value = false;
        await fetchPolicies();
        await fetchPolicyRevisions(getCurrentRevisionPolicyId?.());
      }
    } finally {
      policySubmitting.value = false;
    }
  }

  function handleDeletePolicy(row: WafPolicyItem) {
    deleteWafPolicy(row.id).then(async ({ error }) => {
      if (!error) {
        message.success('策略删除成功');
        if (policyPreviewPolicyName.value === row.name) {
          policyPreviewPolicyName.value = '';
          policyPreviewDirectives.value = '';
        }
        await fetchPolicies();
        await fetchPolicyRevisions(getCurrentRevisionPolicyId?.());
      }
    });
  }

  async function handlePreviewPolicy(row: WafPolicyItem) {
    policyPreviewLoading.value = true;
    try {
      const { data, error } = await previewWafPolicy(row.id);
      if (!error && data) {
        policyPreviewPolicyName.value = row.name;
        policyPreviewDirectives.value = data.directives || '';
        message.success('已生成策略预览');
      }
    } finally {
      policyPreviewLoading.value = false;
    }
  }

  async function handleValidatePolicy(row: WafPolicyItem) {
    const { error } = await validateWafPolicy(row.id);
    if (!error) {
      message.success(`策略 ${row.name} 校验通过`);
    }
  }

  function handlePublishPolicy(row: WafPolicyItem) {
    const isBlockingMode = row.engineMode === 'on';
    const highRiskParanoia = Number(row.crsParanoiaLevel || 0) >= 3;
    const warningParts: string[] = [];

    if (isBlockingMode) {
      warningParts.push('当前为 On（阻断）模式');
    }
    if (highRiskParanoia) {
      warningParts.push(`CRS PL=${row.crsParanoiaLevel}`);
    }

    dialog.warning({
      title: warningParts.length ? '高风险发布确认' : '发布确认',
      content: warningParts.length
        ? `策略 ${row.name} ${warningParts.join('，')}，发布后可能引发误拦截，确认继续发布吗？`
        : `确认发布策略 ${row.name} 吗？`,
      positiveText: '确认发布',
      negativeText: '取消',
      async onPositiveClick() {
        const { error } = await publishWafPolicy(row.id);
        if (!error) {
          message.success('策略发布成功');
          await fetchPolicies();
          await fetchPolicyRevisions(getCurrentRevisionPolicyId?.());
        }
      }
    });
  }

  async function fetchPolicyRevisions(policyId?: number) {
    policyRevisionLoading.value = true;
    try {
      const { data, error } = await fetchWafPolicyRevisionList({
        page: policyRevisionPagination.page as number,
        pageSize: policyRevisionPagination.pageSize as number,
        policyId
      });
      if (!error && data) {
        const list = data.list || [];
        await ensureUserNamesByIds?.(list.map(item => item.operator));
        policyRevisionTable.value = list;
        policyRevisionPagination.itemCount = data.total || 0;
      }
    } finally {
      policyRevisionLoading.value = false;
    }
  }

  function handlePolicyRevisionPageChange(page: number) {
    policyRevisionPagination.page = page;
    fetchPolicyRevisions(getCurrentRevisionPolicyId?.());
  }

  function handlePolicyRevisionPageSizeChange(pageSize: number) {
    policyRevisionPagination.pageSize = pageSize;
    policyRevisionPagination.page = 1;
    fetchPolicyRevisions(getCurrentRevisionPolicyId?.());
  }

  function handleRollbackPolicyRevision(row: WafPolicyRevisionItem) {
    dialog.warning({
      title: '策略回滚确认',
      content: `确认回滚到策略 ${row.policyId} 的版本 v${row.version} 吗？`,
      positiveText: '确认回滚',
      negativeText: '取消',
      async onPositiveClick() {
        const { error } = await rollbackWafPolicy({ revisionId: row.id });
        if (!error) {
          message.success('策略回滚成功');
          await fetchPolicies();
          await fetchPolicyRevisions(getCurrentRevisionPolicyId?.());
        }
      }
    });
  }

  function getDefaultPolicyId() {
    const preferred = policyTable.value.find(item => item.isDefault) || policyTable.value[0];
    return Number(preferred?.id || 0);
  }

  return {
    policyQuery,
    policyLoading,
    policyTable,
    policyPagination,
    policyModalVisible,
    policyModalMode,
    policySubmitting,
    policyFormRef,
    policyForm,
    policyModalTitle,
    policyPreviewLoading,
    policyPreviewPolicyName,
    policyPreviewDirectives,
    policyRevisionLoading,
    policyRevisionTable,
    policyRevisionPagination,
    crsPolicyOptions,

    fetchPolicies,
    resetPolicyQuery,
    handlePolicyPageChange,
    handlePolicyPageSizeChange,
    resetPolicyForm,
    handleAddPolicy,
    handleEditPolicy,
    buildPolicyPayload,
    handleSubmitPolicy,
    handleDeletePolicy,
    handlePreviewPolicy,
    handleValidatePolicy,
    handlePublishPolicy,
    fetchPolicyRevisions,
    handlePolicyRevisionPageChange,
    handlePolicyRevisionPageSizeChange,
    handleRollbackPolicyRevision,
    getDefaultPolicyId
  };
}
