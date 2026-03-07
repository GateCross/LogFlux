import { computed, nextTick, reactive, ref, watch } from 'vue';
import type { FormInst, FormRules, InputInst, PaginationProps } from 'naive-ui';
import {
  type WafPolicyRemoveType,
  type WafPolicyScopeType,
  type WafRuleExclusionItem,
  type WafRuleExclusionPayload,
  createWafRuleExclusion,
  deleteWafRuleExclusion,
  fetchWafRuleExclusionList,
  updateWafRuleExclusion
} from '@/service/api/caddy-policy';

type MessageApi = {
  success: (content: string) => void;
};

interface UseWafExclusionOptions {
  message: MessageApi;
  getDefaultPolicyId: () => number;
}

export function useWafExclusion(options: UseWafExclusionOptions) {
  const { message, getDefaultPolicyId } = options;

  const exclusionQuery = reactive({
    policyId: null as number | null,
    scopeType: '' as '' | WafPolicyScopeType | null,
    name: ''
  });

  const exclusionLoading = ref(false);
  const exclusionTable = ref<WafRuleExclusionItem[]>([]);
  const exclusionPagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 20,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100]
  });

  const exclusionModalVisible = ref(false);
  const exclusionModalMode = ref<'add' | 'edit'>('add');
  const exclusionSubmitting = ref(false);
  const exclusionFormRef = ref<FormInst | null>(null);
  const exclusionRemoveValueInputRef = ref<InputInst | null>(null);
  const shouldFocusExclusionRemoveValue = ref(false);
  const exclusionForm = reactive({
    id: 0,
    policyId: 0,
    name: '',
    description: '',
    enabled: true,
    scopeType: 'global' as WafPolicyScopeType,
    host: '',
    path: '',
    method: '' as string | null,
    removeType: 'id' as WafPolicyRemoveType,
    removeValue: ''
  });

  const exclusionModalTitle = computed(() => (exclusionModalMode.value === 'add' ? '新增规则例外' : '编辑规则例外'));
  const exclusionRules: FormRules = {
    policyId: {
      validator(_rule, value: number) {
        if (!Number(value)) return new Error('请选择关联策略');
        return true;
      },
      trigger: 'change'
    },
    scopeType: { required: true, message: '请选择作用域', trigger: 'change' },
    removeType: {
      required: true,
      message: '请选择移除类型',
      trigger: 'change'
    },
    removeValue: { required: true, message: '请输入移除值', trigger: 'blur' },
    host: {
      validator(_rule, value: string) {
        if (exclusionForm.scopeType === 'site' && !String(value || '').trim()) {
          return new Error('站点作用域必须填写 host');
        }
        return true;
      },
      trigger: ['blur', 'input']
    },
    path: {
      validator(_rule, value: string) {
        if (exclusionForm.scopeType === 'route' && !String(value || '').trim()) {
          return new Error('路由作用域必须填写 path');
        }
        return true;
      },
      trigger: ['blur', 'input']
    }
  };

  async function fetchExclusions() {
    exclusionLoading.value = true;
    try {
      const { data, error } = await fetchWafRuleExclusionList({
        page: exclusionPagination.page as number,
        pageSize: exclusionPagination.pageSize as number,
        policyId: exclusionQuery.policyId || undefined,
        scopeType: exclusionQuery.scopeType || undefined,
        name: exclusionQuery.name.trim() || undefined
      });
      if (!error && data) {
        exclusionTable.value = data.list || [];
        exclusionPagination.itemCount = data.total || 0;
      }
    } finally {
      exclusionLoading.value = false;
    }
  }

  function resetExclusionQuery() {
    exclusionQuery.policyId = null;
    exclusionQuery.scopeType = '';
    exclusionQuery.name = '';
    exclusionPagination.page = 1;
    fetchExclusions().catch(() => undefined);
  }

  function handleExclusionPageChange(page: number) {
    exclusionPagination.page = page;
    fetchExclusions().catch(() => undefined);
  }

  function handleExclusionPageSizeChange(pageSize: number) {
    exclusionPagination.pageSize = pageSize;
    exclusionPagination.page = 1;
    fetchExclusions().catch(() => undefined);
  }

  function resetExclusionForm() {
    exclusionForm.id = 0;
    exclusionForm.policyId = getDefaultPolicyId();
    exclusionForm.name = '';
    exclusionForm.description = '';
    exclusionForm.enabled = true;
    exclusionForm.scopeType = 'global';
    exclusionForm.host = '';
    exclusionForm.path = '';
    exclusionForm.method = '';
    exclusionForm.removeType = 'id';
    exclusionForm.removeValue = '';
  }

  function handleAddExclusion() {
    exclusionModalMode.value = 'add';
    resetExclusionForm();
    exclusionModalVisible.value = true;
  }

  function handleEditExclusion(row: WafRuleExclusionItem) {
    exclusionModalMode.value = 'edit';
    exclusionForm.id = row.id;
    exclusionForm.policyId = row.policyId;
    exclusionForm.name = row.name || '';
    exclusionForm.description = row.description || '';
    exclusionForm.enabled = row.enabled;
    exclusionForm.scopeType = row.scopeType;
    exclusionForm.host = row.host || '';
    exclusionForm.path = row.path || '';
    exclusionForm.method = row.method || '';
    exclusionForm.removeType = row.removeType;
    exclusionForm.removeValue = row.removeValue || '';
    exclusionModalVisible.value = true;
  }

  function buildExclusionPayload(): WafRuleExclusionPayload {
    return {
      policyId: Number(exclusionForm.policyId),
      name: exclusionForm.name.trim(),
      description: exclusionForm.description.trim(),
      enabled: exclusionForm.enabled,
      scopeType: exclusionForm.scopeType,
      host: exclusionForm.host.trim(),
      path: exclusionForm.path.trim(),
      method: String(exclusionForm.method || '').trim(),
      removeType: exclusionForm.removeType,
      removeValue: exclusionForm.removeValue.trim()
    };
  }

  async function handleSubmitExclusion() {
    await exclusionFormRef.value?.validate();
    exclusionSubmitting.value = true;
    try {
      const payload = buildExclusionPayload();
      const request =
        exclusionModalMode.value === 'add'
          ? createWafRuleExclusion(payload)
          : updateWafRuleExclusion(exclusionForm.id, payload);
      const { error } = await request;
      if (!error) {
        message.success(exclusionModalMode.value === 'add' ? '规则例外创建成功' : '规则例外更新成功');
        exclusionModalVisible.value = false;
        fetchExclusions().catch(() => undefined);
      }
    } finally {
      exclusionSubmitting.value = false;
    }
  }

  function handleDeleteExclusion(row: WafRuleExclusionItem) {
    deleteWafRuleExclusion(row.id).then(({ error }) => {
      if (!error) {
        message.success('规则例外删除成功');
        fetchExclusions().catch(() => undefined);
      }
    });
  }

  watch(
    () => exclusionForm.scopeType,
    value => {
      if (value === 'global') {
        exclusionForm.host = '';
        exclusionForm.path = '';
        exclusionForm.method = '';
      } else if (value === 'site') {
        exclusionForm.path = '';
        exclusionForm.method = '';
      }
    }
  );

  watch(exclusionModalVisible, value => {
    if (!value || !shouldFocusExclusionRemoveValue.value) {
      return;
    }
    nextTick(() => {
      exclusionRemoveValueInputRef.value?.focus();
      shouldFocusExclusionRemoveValue.value = false;
    });
  });

  return {
    exclusionQuery,
    exclusionLoading,
    exclusionTable,
    exclusionPagination,
    exclusionModalVisible,
    exclusionModalMode,
    exclusionSubmitting,
    exclusionFormRef,
    exclusionRemoveValueInputRef,
    shouldFocusExclusionRemoveValue,
    exclusionForm,
    exclusionModalTitle,
    exclusionRules,
    fetchExclusions,
    resetExclusionQuery,
    handleExclusionPageChange,
    handleExclusionPageSizeChange,
    resetExclusionForm,
    handleAddExclusion,
    handleEditExclusion,
    handleSubmitExclusion,
    handleDeleteExclusion
  };
}
