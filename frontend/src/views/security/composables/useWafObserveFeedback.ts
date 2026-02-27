import { reactive, ref, type Ref } from 'vue';
import type { FormInst } from 'naive-ui';
import {
  batchUpdateWafPolicyFalsePositiveFeedbackStatus,
  createWafPolicyFalsePositiveFeedback,
  updateWafPolicyFalsePositiveFeedbackStatus,
  type WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload,
  type WafPolicyFalsePositiveFeedbackItem,
  type WafPolicyFalsePositiveFeedbackPayload,
  type WafPolicyFalsePositiveFeedbackStatusUpdatePayload
} from '@/service/api/caddy-observe';

type MessageApi = {
  success: (content: string) => void;
  warning: (content: string) => void;
  error: (content: string) => void;
};

interface UseWafObserveFeedbackOptions {
  message: MessageApi;
  policyStatsQuery: {
    policyId: number | '' | null;
    host: string;
    path: string;
    method: string;
  };
  policyFeedbackAssigneeFilter: Ref<string>;
  policyFeedbackCheckedRowKeys: Ref<number[]>;
  policyFeedbackPagination: { page?: number };
  resetPolicyFeedbackSelection: () => void;
  fetchPolicyFalsePositiveFeedbacks: () => void | Promise<void>;
}

export function useWafObserveFeedback(options: UseWafObserveFeedbackOptions) {
  const {
    message,
    policyStatsQuery,
    policyFeedbackAssigneeFilter,
    policyFeedbackCheckedRowKeys,
    policyFeedbackPagination,
    resetPolicyFeedbackSelection,
    fetchPolicyFalsePositiveFeedbacks
  } = options;

  const policyFeedbackModalVisible = ref(false);
  const policyFeedbackSubmitting = ref(false);
  const policyFeedbackFormRef = ref<FormInst | null>(null);
  const policyFeedbackForm = reactive({
    policyId: null as number | null,
    host: '',
    path: '',
    method: '' as string | null,
    status: 403,
    assignee: '',
    dueAt: '',
    sampleUri: '',
    reason: '',
    suggestion: ''
  });

  const policyFeedbackProcessModalVisible = ref(false);
  const policyFeedbackProcessSubmitting = ref(false);
  const policyFeedbackProcessFormRef = ref<FormInst | null>(null);
  const policyFeedbackProcessForm = reactive({
    id: 0,
    feedbackStatus: 'confirmed' as 'pending' | 'confirmed' | 'resolved',
    processNote: '',
    assignee: '',
    dueAt: ''
  });

  const policyFeedbackBatchProcessModalVisible = ref(false);
  const policyFeedbackBatchProcessSubmitting = ref(false);
  const policyFeedbackBatchProcessFormRef = ref<FormInst | null>(null);
  const policyFeedbackBatchProcessForm = reactive({
    feedbackStatus: 'confirmed' as 'pending' | 'confirmed' | 'resolved',
    processNote: '',
    assignee: '',
    dueAt: ''
  });

  function resetPolicyFeedbackForm() {
    policyFeedbackForm.policyId = policyStatsQuery.policyId ? Number(policyStatsQuery.policyId) : null;
    policyFeedbackForm.host = policyStatsQuery.host.trim();
    policyFeedbackForm.path = policyStatsQuery.path.trim();
    policyFeedbackForm.method = policyStatsQuery.method.trim().toUpperCase() || null;
    policyFeedbackForm.status = 403;
    policyFeedbackForm.assignee = policyFeedbackAssigneeFilter.value.trim();
    policyFeedbackForm.dueAt = '';
    policyFeedbackForm.sampleUri = '';
    policyFeedbackForm.reason = '';
    policyFeedbackForm.suggestion = '';
  }

  function openPolicyFeedbackModal() {
    resetPolicyFeedbackForm();
    policyFeedbackModalVisible.value = true;
  }

  function openPolicyFeedbackProcessModal(row: WafPolicyFalsePositiveFeedbackItem) {
    policyFeedbackProcessForm.id = Number(row.id || 0);
    policyFeedbackProcessForm.feedbackStatus = (row.feedbackStatus || 'pending') as 'pending' | 'confirmed' | 'resolved';
    policyFeedbackProcessForm.processNote = row.processNote || '';
    policyFeedbackProcessForm.assignee = row.assignee || '';
    policyFeedbackProcessForm.dueAt = row.dueAt || '';
    policyFeedbackProcessModalVisible.value = true;
  }

  async function handleSubmitPolicyFeedbackProcess() {
    await policyFeedbackProcessFormRef.value?.validate();
    if (!policyFeedbackProcessForm.id) {
      message.error('反馈记录无效');
      return;
    }
    policyFeedbackProcessSubmitting.value = true;
    try {
      const payload: WafPolicyFalsePositiveFeedbackStatusUpdatePayload = {
        feedbackStatus: policyFeedbackProcessForm.feedbackStatus,
        processNote: policyFeedbackProcessForm.processNote.trim() || undefined,
        assignee: policyFeedbackProcessForm.assignee.trim() || undefined,
        dueAt: policyFeedbackProcessForm.dueAt.trim() || undefined
      };
      const { error } = await updateWafPolicyFalsePositiveFeedbackStatus(policyFeedbackProcessForm.id, payload);
      if (!error) {
        message.success('误报反馈状态已更新');
        policyFeedbackProcessModalVisible.value = false;
        fetchPolicyFalsePositiveFeedbacks();
      }
    } finally {
      policyFeedbackProcessSubmitting.value = false;
    }
  }

  function resetPolicyFeedbackBatchProcessForm() {
    policyFeedbackBatchProcessForm.feedbackStatus = 'confirmed';
    policyFeedbackBatchProcessForm.processNote = '';
    policyFeedbackBatchProcessForm.assignee = policyFeedbackAssigneeFilter.value.trim();
    policyFeedbackBatchProcessForm.dueAt = '';
  }

  function openPolicyFeedbackBatchProcessModal() {
    if (!policyFeedbackCheckedRowKeys.value.length) {
      message.warning('请先选择要处理的反馈记录');
      return;
    }
    resetPolicyFeedbackBatchProcessForm();
    policyFeedbackBatchProcessModalVisible.value = true;
  }

  async function handleSubmitPolicyFeedbackBatchProcess() {
    await policyFeedbackBatchProcessFormRef.value?.validate();
    const selectedIDs = Array.from(
      new Set(policyFeedbackCheckedRowKeys.value.map(id => Number(id)).filter(id => Number.isInteger(id) && id > 0))
    );
    if (!selectedIDs.length) {
      message.warning('未选择可处理的反馈记录');
      return;
    }

    policyFeedbackBatchProcessSubmitting.value = true;
    try {
      const payload: WafPolicyFalsePositiveFeedbackBatchStatusUpdatePayload = {
        ids: selectedIDs,
        feedbackStatus: policyFeedbackBatchProcessForm.feedbackStatus,
        processNote: policyFeedbackBatchProcessForm.processNote.trim() || undefined,
        assignee: policyFeedbackBatchProcessForm.assignee.trim() || undefined,
        dueAt: policyFeedbackBatchProcessForm.dueAt.trim() || undefined
      };
      const { data, error } = await batchUpdateWafPolicyFalsePositiveFeedbackStatus(payload);
      if (!error) {
        const affectedCount = Number(data?.affectedCount || 0);
        message.success(affectedCount > 0 ? `批量处理完成，已更新 ${affectedCount} 条反馈` : '批量处理完成');
        policyFeedbackBatchProcessModalVisible.value = false;
        resetPolicyFeedbackSelection();
        fetchPolicyFalsePositiveFeedbacks();
      }
    } finally {
      policyFeedbackBatchProcessSubmitting.value = false;
    }
  }

  async function handleSubmitPolicyFeedback() {
    await policyFeedbackFormRef.value?.validate();
    policyFeedbackSubmitting.value = true;
    try {
      const payload: WafPolicyFalsePositiveFeedbackPayload = {
        policyId: policyFeedbackForm.policyId || undefined,
        host: policyFeedbackForm.host.trim() || undefined,
        path: policyFeedbackForm.path.trim() || undefined,
        method: String(policyFeedbackForm.method || '')
          .trim()
          .toUpperCase() || undefined,
        status: Number(policyFeedbackForm.status || 403),
        assignee: policyFeedbackForm.assignee.trim() || undefined,
        dueAt: policyFeedbackForm.dueAt.trim() || undefined,
        sampleUri: policyFeedbackForm.sampleUri.trim() || undefined,
        reason: policyFeedbackForm.reason.trim(),
        suggestion: policyFeedbackForm.suggestion.trim() || undefined
      };
      const { error } = await createWafPolicyFalsePositiveFeedback(payload);
      if (!error) {
        message.success('误报反馈已提交');
        policyFeedbackModalVisible.value = false;
        policyFeedbackPagination.page = 1;
        resetPolicyFeedbackSelection();
        fetchPolicyFalsePositiveFeedbacks();
      }
    } finally {
      policyFeedbackSubmitting.value = false;
    }
  }

  return {
    policyFeedbackModalVisible,
    policyFeedbackSubmitting,
    policyFeedbackFormRef,
    policyFeedbackForm,
    policyFeedbackProcessModalVisible,
    policyFeedbackProcessSubmitting,
    policyFeedbackProcessFormRef,
    policyFeedbackProcessForm,
    policyFeedbackBatchProcessModalVisible,
    policyFeedbackBatchProcessSubmitting,
    policyFeedbackBatchProcessFormRef,
    policyFeedbackBatchProcessForm,
    openPolicyFeedbackModal,
    openPolicyFeedbackProcessModal,
    openPolicyFeedbackBatchProcessModal,
    handleSubmitPolicyFeedback,
    handleSubmitPolicyFeedbackProcess,
    handleSubmitPolicyFeedbackBatchProcess,
    resetPolicyFeedbackForm
  };
}
