import { type Ref, computed, reactive, ref, watch } from 'vue';
import type { FormInst, FormRules } from 'naive-ui';
import {
  type WafPolicyCrsTemplate,
  type WafPolicyItem,
  publishWafPolicy,
  updateWafPolicy
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

interface UseWafCrsTuningOptions {
  message: MessageApi;
  dialog: DialogApi;
  activeTab: Ref<string>;
  policyTable: Ref<WafPolicyItem[]>;
  crsTemplatePresetMap: Record<
    Exclude<WafPolicyCrsTemplate, 'custom'>,
    {
      crsParanoiaLevel: number;
      crsInboundAnomalyThreshold: number;
      crsOutboundAnomalyThreshold: number;
    }
  >;
  previewPolicy: (policy: WafPolicyItem) => Promise<void> | void;
  validatePolicy: (policy: WafPolicyItem) => Promise<void> | void;
  fetchPolicies: () => Promise<void> | void;
  fetchPolicyRevisions: (policyId?: number) => Promise<void> | void;
  resetPolicyRevisionPage?: () => void;
}

export function useWafCrsTuning(options: UseWafCrsTuningOptions) {
  const {
    message,
    dialog,
    activeTab,
    policyTable,
    crsTemplatePresetMap,
    previewPolicy,
    validatePolicy,
    fetchPolicies,
    fetchPolicyRevisions,
    resetPolicyRevisionPage
  } = options;

  const crsTuningSubmitting = ref(false);
  const crsTuningFormRef = ref<FormInst | null>(null);
  const crsTuningForm = reactive({
    policyId: 0,
    crsTemplate: 'low_fp' as WafPolicyCrsTemplate,
    crsParanoiaLevel: 1,
    crsInboundAnomalyThreshold: 10,
    crsOutboundAnomalyThreshold: 8
  });

  const crsTuningRules: FormRules = {
    policyId: {
      validator(_rule, value: number) {
        if (!Number(value)) {
          return new Error('请选择策略');
        }
        return true;
      },
      trigger: 'change'
    },
    crsParanoiaLevel: {
      validator(_rule, value: number) {
        const num = Number(value);
        if (!Number.isFinite(num) || num < 1 || num > 4) {
          return new Error('PL 必须在 1 到 4 之间');
        }
        return true;
      },
      trigger: ['blur', 'change']
    },
    crsInboundAnomalyThreshold: {
      validator(_rule, value: number) {
        const num = Number(value);
        if (!Number.isFinite(num) || num < 1 || num > 20) {
          return new Error('Inbound 阈值必须在 1 到 20 之间');
        }
        return true;
      },
      trigger: ['blur', 'change']
    },
    crsOutboundAnomalyThreshold: {
      validator(_rule, value: number) {
        const num = Number(value);
        if (!Number.isFinite(num) || num < 1 || num > 20) {
          return new Error('Outbound 阈值必须在 1 到 20 之间');
        }
        return true;
      },
      trigger: ['blur', 'change']
    }
  };

  function inferCrsTemplateByValues(
    crsParanoiaLevel: number,
    crsInboundAnomalyThreshold: number,
    crsOutboundAnomalyThreshold: number
  ): WafPolicyCrsTemplate {
    for (const [template, preset] of Object.entries(crsTemplatePresetMap) as Array<
      [
        Exclude<WafPolicyCrsTemplate, 'custom'>,
        {
          crsParanoiaLevel: number;
          crsInboundAnomalyThreshold: number;
          crsOutboundAnomalyThreshold: number;
        }
      ]
    >) {
      if (
        preset.crsParanoiaLevel === crsParanoiaLevel &&
        preset.crsInboundAnomalyThreshold === crsInboundAnomalyThreshold &&
        preset.crsOutboundAnomalyThreshold === crsOutboundAnomalyThreshold
      ) {
        return template;
      }
    }
    return 'custom';
  }

  function getCurrentRevisionPolicyId() {
    return activeTab.value === 'crs' ? crsTuningForm.policyId || undefined : undefined;
  }

  function syncCrsTuningFromPolicy(policy: WafPolicyItem | null | undefined) {
    if (!policy) {
      crsTuningForm.policyId = 0;
      crsTuningForm.crsTemplate = 'low_fp';
      crsTuningForm.crsParanoiaLevel = 1;
      crsTuningForm.crsInboundAnomalyThreshold = 10;
      crsTuningForm.crsOutboundAnomalyThreshold = 8;
      return;
    }

    const crsParanoiaLevel = Number(policy.crsParanoiaLevel || 1);
    const crsInboundAnomalyThreshold = Number(policy.crsInboundAnomalyThreshold || 10);
    const crsOutboundAnomalyThreshold = Number(policy.crsOutboundAnomalyThreshold || 8);
    const inferredTemplate = inferCrsTemplateByValues(
      crsParanoiaLevel,
      crsInboundAnomalyThreshold,
      crsOutboundAnomalyThreshold
    );

    crsTuningForm.policyId = policy.id;
    crsTuningForm.crsParanoiaLevel = crsParanoiaLevel;
    crsTuningForm.crsInboundAnomalyThreshold = crsInboundAnomalyThreshold;
    crsTuningForm.crsOutboundAnomalyThreshold = crsOutboundAnomalyThreshold;
    crsTuningForm.crsTemplate = (policy.crsTemplate as WafPolicyCrsTemplate) || inferredTemplate;
  }

  function syncCrsTuningFromPolicyTable() {
    if (!policyTable.value.length) {
      syncCrsTuningFromPolicy(null);
      return;
    }

    const current = policyTable.value.find(item => item.id === crsTuningForm.policyId);
    if (current) {
      syncCrsTuningFromPolicy(current);
      return;
    }

    const preferred = policyTable.value.find(item => item.isDefault) || policyTable.value[0];
    syncCrsTuningFromPolicy(preferred);
  }

  function handleCrsPolicyChange(policyId: number | null) {
    const policy = policyTable.value.find(item => item.id === Number(policyId || 0));
    syncCrsTuningFromPolicy(policy);
    resetPolicyRevisionPage?.();
    Promise.resolve(fetchPolicyRevisions(getCurrentRevisionPolicyId())).catch(() => undefined);
  }

  function handleRefreshCrsPolicy() {
    Promise.resolve(fetchPolicies()).catch(() => undefined);
    Promise.resolve(fetchPolicyRevisions(getCurrentRevisionPolicyId())).catch(() => undefined);
  }

  function applyCrsTemplatePreset(template: Exclude<WafPolicyCrsTemplate, 'custom'>) {
    const preset = crsTemplatePresetMap[template];
    crsTuningForm.crsTemplate = template;
    crsTuningForm.crsParanoiaLevel = preset.crsParanoiaLevel;
    crsTuningForm.crsInboundAnomalyThreshold = preset.crsInboundAnomalyThreshold;
    crsTuningForm.crsOutboundAnomalyThreshold = preset.crsOutboundAnomalyThreshold;
  }

  function buildCrsTuningPayload() {
    const crsParanoiaLevel = Number(crsTuningForm.crsParanoiaLevel);
    const crsInboundAnomalyThreshold = Number(crsTuningForm.crsInboundAnomalyThreshold);
    const crsOutboundAnomalyThreshold = Number(crsTuningForm.crsOutboundAnomalyThreshold);
    const inferredTemplate = inferCrsTemplateByValues(
      crsParanoiaLevel,
      crsInboundAnomalyThreshold,
      crsOutboundAnomalyThreshold
    );

    return {
      crsTemplate: inferredTemplate,
      crsParanoiaLevel,
      crsInboundAnomalyThreshold,
      crsOutboundAnomalyThreshold
    };
  }

  function getCurrentCrsPolicy() {
    return policyTable.value.find(item => item.id === crsTuningForm.policyId) || null;
  }

  function hasPendingCrsTuningChanges() {
    const policy = getCurrentCrsPolicy();
    if (!policy) {
      return false;
    }

    const payload = buildCrsTuningPayload();
    const currentTemplate = inferCrsTemplateByValues(
      Number(policy.crsParanoiaLevel || 1),
      Number(policy.crsInboundAnomalyThreshold || 10),
      Number(policy.crsOutboundAnomalyThreshold || 8)
    );

    return (
      Number(payload.crsParanoiaLevel) !== Number(policy.crsParanoiaLevel) ||
      Number(payload.crsInboundAnomalyThreshold) !== Number(policy.crsInboundAnomalyThreshold) ||
      Number(payload.crsOutboundAnomalyThreshold) !== Number(policy.crsOutboundAnomalyThreshold) ||
      payload.crsTemplate !== currentTemplate
    );
  }

  async function persistCrsTuning(showSuccessMessage = true) {
    await crsTuningFormRef.value?.validate();
    if (!crsTuningForm.policyId) {
      message.warning('请先选择策略');
      return false;
    }

    const { error } = await updateWafPolicy(crsTuningForm.policyId, buildCrsTuningPayload());
    if (error) {
      return false;
    }

    if (showSuccessMessage) {
      message.success('CRS 调优参数已保存');
    }
    await fetchPolicies();
    await fetchPolicyRevisions(getCurrentRevisionPolicyId());
    return true;
  }

  async function handleSaveCrsTuning() {
    crsTuningSubmitting.value = true;
    try {
      await persistCrsTuning(true);
    } finally {
      crsTuningSubmitting.value = false;
    }
  }

  async function handlePreviewCrsTuning() {
    if (!crsTuningForm.policyId) {
      message.warning('请先选择策略');
      return;
    }
    if (hasPendingCrsTuningChanges()) {
      message.warning('当前调优参数尚未保存，请先点击“保存调优参数”');
      return;
    }

    const policy = getCurrentCrsPolicy();
    if (policy) {
      await previewPolicy(policy);
    }
  }

  async function handleValidateCrsTuning() {
    if (!crsTuningForm.policyId) {
      message.warning('请先选择策略');
      return;
    }
    if (hasPendingCrsTuningChanges()) {
      message.warning('当前调优参数尚未保存，请先点击“保存调优参数”');
      return;
    }

    const policy = getCurrentCrsPolicy();
    if (policy) {
      await validatePolicy(policy);
    }
  }

  function handlePublishCrsTuning() {
    if (!crsTuningForm.policyId) {
      message.warning('请先选择策略');
      return;
    }

    const policy = policyTable.value.find(item => item.id === crsTuningForm.policyId);
    if (!policy) {
      message.warning('未找到对应策略，请先刷新');
      return;
    }

    const highRisk = Number(crsTuningForm.crsParanoiaLevel) >= 3;
    const content = highRisk
      ? `当前 PL=${crsTuningForm.crsParanoiaLevel}，误拦截风险较高。确认保存调优参数并发布策略 ${policy.name} 吗？`
      : `确认保存调优参数并发布策略 ${policy.name} 吗？`;

    dialog.warning({
      title: highRisk ? '高风险调优发布确认' : 'CRS 调优发布确认',
      content,
      positiveText: '确认发布',
      negativeText: '取消',
      async onPositiveClick() {
        crsTuningSubmitting.value = true;
        try {
          if (hasPendingCrsTuningChanges()) {
            const persisted = await persistCrsTuning(false);
            if (!persisted) {
              return;
            }
          }

          const { error } = await publishWafPolicy(crsTuningForm.policyId);
          if (!error) {
            message.success('CRS 调优参数发布成功');
            await fetchPolicies();
            await fetchPolicyRevisions(getCurrentRevisionPolicyId());
          }
        } finally {
          crsTuningSubmitting.value = false;
        }
      }
    });
  }

  watch(
    () => [
      crsTuningForm.crsParanoiaLevel,
      crsTuningForm.crsInboundAnomalyThreshold,
      crsTuningForm.crsOutboundAnomalyThreshold
    ],
    values => {
      const [crsParanoiaLevel, crsInboundAnomalyThreshold, crsOutboundAnomalyThreshold] = values.map(value =>
        Number(value)
      );
      if (
        !Number.isFinite(crsParanoiaLevel) ||
        !Number.isFinite(crsInboundAnomalyThreshold) ||
        !Number.isFinite(crsOutboundAnomalyThreshold)
      ) {
        return;
      }
      crsTuningForm.crsTemplate = inferCrsTemplateByValues(
        crsParanoiaLevel,
        crsInboundAnomalyThreshold,
        crsOutboundAnomalyThreshold
      );
    }
  );

  const hasPolicyWorkspaceDraft = computed(() => hasPendingCrsTuningChanges());

  return {
    crsTuningSubmitting,
    crsTuningFormRef,
    crsTuningForm,
    crsTuningRules,
    hasPolicyWorkspaceDraft,
    inferCrsTemplateByValues,
    syncCrsTuningFromPolicy,
    syncCrsTuningFromPolicyTable,
    getCurrentRevisionPolicyId,
    handleCrsPolicyChange,
    handleRefreshCrsPolicy,
    applyCrsTemplatePreset,
    handleSaveCrsTuning,
    handlePreviewCrsTuning,
    handleValidateCrsTuning,
    handlePublishCrsTuning
  };
}
