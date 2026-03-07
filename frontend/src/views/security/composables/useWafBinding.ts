import { computed, reactive, ref, watch } from 'vue';
import type { FormInst, FormRules, PaginationProps } from 'naive-ui';
import {
  type WafPolicyBindingItem,
  type WafPolicyBindingPayload,
  type WafPolicyScopeType,
  createWafPolicyBinding,
  deleteWafPolicyBinding,
  fetchWafPolicyBindingList,
  updateWafPolicyBinding
} from '@/service/api/caddy-policy';

type MessageApi = {
  success: (content: string) => void;
};

export interface BindingConflictGroup {
  scopeType: string;
  host: string;
  path: string;
  method: string;
  priority: number;
  count: number;
}

export interface BindingEffectiveItem {
  id: number;
  order: number;
  policyId: number;
  policyName: string;
  scopeType: string;
  host: string;
  path: string;
  method: string;
  priority: number;
}

interface UseWafBindingOptions {
  message: MessageApi;
  getDefaultPolicyId: () => number;
  mapPolicyNameById: (policyId: number) => string;
}

export function useWafBinding(options: UseWafBindingOptions) {
  const { message, getDefaultPolicyId, mapPolicyNameById } = options;

  const bindingQuery = reactive({
    policyId: null as number | null,
    scopeType: '' as '' | WafPolicyScopeType | null,
    name: ''
  });

  const bindingLoading = ref(false);
  const bindingTable = ref<WafPolicyBindingItem[]>([]);
  const bindingPagination = reactive<PaginationProps>({
    page: 1,
    pageSize: 20,
    itemCount: 0,
    showSizePicker: true,
    pageSizes: [10, 20, 50, 100]
  });

  const bindingModalVisible = ref(false);
  const bindingModalMode = ref<'add' | 'edit'>('add');
  const bindingSubmitting = ref(false);
  const bindingFormRef = ref<FormInst | null>(null);
  const bindingForm = reactive({
    id: 0,
    policyId: 0,
    name: '',
    description: '',
    enabled: true,
    scopeType: 'global' as WafPolicyScopeType,
    host: '',
    path: '',
    method: '' as string | null,
    priority: 100
  });

  const bindingModalTitle = computed(() => (bindingModalMode.value === 'add' ? '新增策略绑定' : '编辑策略绑定'));
  const bindingRules: FormRules = {
    policyId: {
      validator(_rule, value: number) {
        if (!Number(value)) return new Error('请选择关联策略');
        return true;
      },
      trigger: 'change'
    },
    scopeType: { required: true, message: '请选择作用域', trigger: 'change' },
    priority: {
      validator(_rule, value: number) {
        const num = Number(value);
        if (!Number.isFinite(num) || num < 1 || num > 1000) {
          return new Error('优先级必须在 1 到 1000 之间');
        }
        return true;
      },
      trigger: ['blur', 'change']
    },
    host: {
      validator(_rule, value: string) {
        if (bindingForm.scopeType === 'site' && !String(value || '').trim()) {
          return new Error('站点作用域必须填写 host');
        }
        return true;
      },
      trigger: ['blur', 'input']
    },
    path: {
      validator(_rule, value: string) {
        if (bindingForm.scopeType === 'route' && !String(value || '').trim()) {
          return new Error('路由作用域必须填写 path');
        }
        return true;
      },
      trigger: ['blur', 'input']
    }
  };

  const bindingConflictGroups = computed<BindingConflictGroup[]>(() => {
    const groups = new Map<string, BindingConflictGroup>();
    bindingTable.value
      .filter(item => item.enabled)
      .forEach(item => {
        const key = [
          item.scopeType || '',
          String(item.host || '').toLowerCase(),
          item.path || '',
          String(item.method || '').toUpperCase(),
          Number(item.priority || 0)
        ].join('|');
        const current = groups.get(key);
        if (!current) {
          groups.set(key, {
            scopeType: item.scopeType,
            host: item.host || '',
            path: item.path || '',
            method: item.method || '',
            priority: Number(item.priority || 0),
            count: 1
          });
        } else {
          current.count += 1;
        }
      });

    return Array.from(groups.values())
      .filter(item => item.count > 1)
      .sort((a, b) => b.count - a.count || a.priority - b.priority);
  });

  const bindingEffectivePreview = computed<BindingEffectiveItem[]>(() => {
    const scopeWeightMap: Record<string, number> = {
      global: 1,
      site: 2,
      route: 3
    };

    const sorted = [...bindingTable.value]
      .filter(item => item.enabled)
      .sort((a, b) => {
        const scopeWeightA = scopeWeightMap[a.scopeType] || 99;
        const scopeWeightB = scopeWeightMap[b.scopeType] || 99;
        if (scopeWeightA !== scopeWeightB) return scopeWeightA - scopeWeightB;
        if (a.priority !== b.priority) return a.priority - b.priority;
        return a.id - b.id;
      });

    return sorted.map((item, index) => ({
      id: item.id,
      order: index + 1,
      policyId: item.policyId,
      policyName: mapPolicyNameById(item.policyId),
      scopeType: item.scopeType,
      host: item.host || '',
      path: item.path || '',
      method: item.method || '',
      priority: item.priority
    }));
  });

  async function fetchBindings() {
    bindingLoading.value = true;
    try {
      const { data, error } = await fetchWafPolicyBindingList({
        page: bindingPagination.page as number,
        pageSize: bindingPagination.pageSize as number,
        policyId: bindingQuery.policyId || undefined,
        scopeType: bindingQuery.scopeType || undefined,
        name: bindingQuery.name.trim() || undefined
      });
      if (!error && data) {
        bindingTable.value = data.list || [];
        bindingPagination.itemCount = data.total || 0;
      }
    } finally {
      bindingLoading.value = false;
    }
  }

  function resetBindingQuery() {
    bindingQuery.policyId = null;
    bindingQuery.scopeType = '';
    bindingQuery.name = '';
    bindingPagination.page = 1;
    fetchBindings().catch(() => undefined);
  }

  function handleBindingPageChange(page: number) {
    bindingPagination.page = page;
    fetchBindings().catch(() => undefined);
  }

  function handleBindingPageSizeChange(pageSize: number) {
    bindingPagination.pageSize = pageSize;
    bindingPagination.page = 1;
    fetchBindings().catch(() => undefined);
  }

  function resetBindingForm() {
    bindingForm.id = 0;
    bindingForm.policyId = getDefaultPolicyId();
    bindingForm.name = '';
    bindingForm.description = '';
    bindingForm.enabled = true;
    bindingForm.scopeType = 'global';
    bindingForm.host = '';
    bindingForm.path = '';
    bindingForm.method = '';
    bindingForm.priority = 100;
  }

  function handleAddBinding() {
    bindingModalMode.value = 'add';
    resetBindingForm();
    bindingModalVisible.value = true;
  }

  function handleEditBinding(row: WafPolicyBindingItem) {
    bindingModalMode.value = 'edit';
    bindingForm.id = row.id;
    bindingForm.policyId = row.policyId;
    bindingForm.name = row.name || '';
    bindingForm.description = row.description || '';
    bindingForm.enabled = row.enabled;
    bindingForm.scopeType = row.scopeType;
    bindingForm.host = row.host || '';
    bindingForm.path = row.path || '';
    bindingForm.method = row.method || '';
    bindingForm.priority = row.priority;
    bindingModalVisible.value = true;
  }

  function buildBindingPayload(): WafPolicyBindingPayload {
    return {
      policyId: Number(bindingForm.policyId),
      name: bindingForm.name.trim(),
      description: bindingForm.description.trim(),
      enabled: bindingForm.enabled,
      scopeType: bindingForm.scopeType,
      host: bindingForm.host.trim(),
      path: bindingForm.path.trim(),
      method: String(bindingForm.method || '').trim(),
      priority: Number(bindingForm.priority)
    };
  }

  async function handleSubmitBinding() {
    await bindingFormRef.value?.validate();
    bindingSubmitting.value = true;
    try {
      const payload = buildBindingPayload();
      const request =
        bindingModalMode.value === 'add'
          ? createWafPolicyBinding(payload)
          : updateWafPolicyBinding(bindingForm.id, payload);
      const { error } = await request;
      if (!error) {
        message.success(bindingModalMode.value === 'add' ? '策略绑定创建成功' : '策略绑定更新成功');
        bindingModalVisible.value = false;
        fetchBindings().catch(() => undefined);
      }
    } finally {
      bindingSubmitting.value = false;
    }
  }

  function handleDeleteBinding(row: WafPolicyBindingItem) {
    deleteWafPolicyBinding(row.id).then(({ error }) => {
      if (!error) {
        message.success('策略绑定删除成功');
        fetchBindings().catch(() => undefined);
      }
    });
  }

  watch(
    () => bindingForm.scopeType,
    value => {
      if (value === 'global') {
        bindingForm.host = '';
        bindingForm.path = '';
        bindingForm.method = '';
      } else if (value === 'site') {
        bindingForm.path = '';
        bindingForm.method = '';
      }
    }
  );

  return {
    bindingQuery,
    bindingLoading,
    bindingTable,
    bindingPagination,
    bindingModalVisible,
    bindingModalMode,
    bindingSubmitting,
    bindingFormRef,
    bindingForm,
    bindingModalTitle,
    bindingRules,
    bindingConflictGroups,
    bindingEffectivePreview,
    fetchBindings,
    resetBindingQuery,
    handleBindingPageChange,
    handleBindingPageSizeChange,
    resetBindingForm,
    handleAddBinding,
    handleEditBinding,
    handleSubmitBinding,
    handleDeleteBinding
  };
}
