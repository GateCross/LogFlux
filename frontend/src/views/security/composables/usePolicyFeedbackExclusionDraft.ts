import { type Ref, computed, ref } from 'vue';
import type { WafPolicyFalsePositiveFeedbackItem } from '@/service/api/caddy-observe';
import type {
  WafPolicyItem,
  WafPolicyRemoveType,
  WafPolicyScopeType,
  WafRuleExclusionPayload
} from '@/service/api/caddy-policy';
import {
  buildExclusionCandidateKey,
  collectExclusionCandidatesFromFeedbackSuggestion,
  parseExclusionCandidateKey,
  parseExclusionFromFeedbackSuggestion
} from '../policy-feedback-draft';

type MessageApi = {
  success: (content: string) => void;
  warning: (content: string) => void;
};

export type PolicyFeedbackExclusionDraft = {
  feedbackId: number;
  policyId: number;
  policyName: string;
  name: string;
  description: string;
  scopeType: WafPolicyScopeType;
  host: string;
  path: string;
  method: string;
  removeType: WafPolicyRemoveType;
  removeValue: string;
  candidates: Array<{
    removeType: WafPolicyRemoveType;
    removeValue: string;
  }>;
  baseline: {
    policyId: number;
    scopeType: WafPolicyScopeType;
    host: string;
    path: string;
    method: string;
    removeType: WafPolicyRemoveType;
    removeValue: string;
  };
};

export type PolicyFeedbackExclusionDraftDiffItem = {
  field: string;
  before: string;
  after: string;
};

interface UsePolicyFeedbackExclusionDraftOptions {
  message: MessageApi;
  policyTable: Ref<WafPolicyItem[]>;
  resetExclusionForm: () => void;
  getDefaultPolicyId: () => number;
  mapPolicyNameById: (policyId: number) => string;
  mapScopeTypeLabel: (scopeType: WafPolicyScopeType) => string;
  openExclusionEditor: (payload: WafRuleExclusionPayload, focusRemoveValue: boolean) => void;
  navigateToPolicyExclusion: () => void | Promise<void>;
}

export function usePolicyFeedbackExclusionDraft(options: UsePolicyFeedbackExclusionDraftOptions) {
  const {
    message,
    resetExclusionForm,
    getDefaultPolicyId,
    mapPolicyNameById,
    mapScopeTypeLabel,
    openExclusionEditor,
    navigateToPolicyExclusion
  } = options;

  const policyFeedbackExclusionDraftModalVisible = ref(false);
  const policyFeedbackExclusionDraft = ref<PolicyFeedbackExclusionDraft | null>(null);
  const policyFeedbackExclusionDraftCandidateKey = ref('');

  const policyFeedbackExclusionCandidateOptions = computed(() => {
    const candidates = policyFeedbackExclusionDraft.value?.candidates || [];
    return candidates.map(item => ({
      label: `${item.removeType === 'id' ? 'removeById' : 'removeByTag'}: ${item.removeValue}`,
      value: buildExclusionCandidateKey(item.removeType, item.removeValue)
    }));
  });

  const policyFeedbackExclusionDraftDiffItems = computed<PolicyFeedbackExclusionDraftDiffItem[]>(() => {
    const draft = policyFeedbackExclusionDraft.value;
    if (!draft) {
      return [];
    }

    const baseline = draft.baseline;
    const diffItems: PolicyFeedbackExclusionDraftDiffItem[] = [];
    const appendDiff = (field: string, beforeValue: string, afterValue: string) => {
      const beforeText = String(beforeValue || '').trim();
      const afterText = String(afterValue || '').trim();
      if (beforeText === afterText) {
        return;
      }
      diffItems.push({
        field,
        before: beforeText,
        after: afterText
      });
    };

    const baselinePolicyName = mapPolicyNameById(Number(baseline.policyId || 0)) || String(baseline.policyId || '');
    const currentPolicyName = mapPolicyNameById(Number(draft.policyId || 0)) || String(draft.policyId || '');
    appendDiff('关联策略', baselinePolicyName, currentPolicyName);
    appendDiff('作用域', mapScopeTypeLabel(baseline.scopeType), mapScopeTypeLabel(draft.scopeType));
    appendDiff('Host', baseline.host, draft.host);
    appendDiff('Path', baseline.path, draft.path);
    appendDiff(
      'Method',
      baseline.method,
      String(draft.method || '')
        .trim()
        .toUpperCase()
    );
    appendDiff(
      '移除类型',
      baseline.removeType === 'id' ? 'removeById' : 'removeByTag',
      draft.removeType === 'id' ? 'removeById' : 'removeByTag'
    );
    appendDiff('移除值', baseline.removeValue, draft.removeValue);
    return diffItems;
  });

  function buildExclusionDraftFromFeedback(row: WafPolicyFalsePositiveFeedbackItem): PolicyFeedbackExclusionDraft {
    const policyId = Number(row.policyId || 0) > 0 ? Number(row.policyId) : getDefaultPolicyId();
    const host = String(row.host || '').trim();
    const path = String(row.path || '').trim();
    const method =
      String(row.method || '')
        .trim()
        .toUpperCase() || '';
    let scopeType: WafPolicyScopeType = 'global';
    if (path) {
      scopeType = 'route';
    } else if (host) {
      scopeType = 'site';
    }
    const candidates = collectExclusionCandidatesFromFeedbackSuggestion(row.suggestion || '');
    const parsed = parseExclusionFromFeedbackSuggestion(row.suggestion || '');
    const reason = String(row.reason || '').trim();
    const suggestion = String(row.suggestion || '').trim();

    return {
      feedbackId: Number(row.id || 0),
      policyId,
      policyName: mapPolicyNameById(policyId),
      name: `fp-${Number(row.id || 0) || Date.now()}`,
      description: suggestion ? `来源反馈#${row.id}：${reason}；建议：${suggestion}` : `来源反馈#${row.id}：${reason}`,
      scopeType,
      host,
      path,
      method,
      removeType: parsed.removeType,
      removeValue: parsed.removeValue,
      candidates,
      baseline: {
        policyId,
        scopeType,
        host,
        path,
        method,
        removeType: parsed.removeType,
        removeValue: parsed.removeValue
      }
    };
  }

  function handleCreateExclusionDraftFromFeedback(row: WafPolicyFalsePositiveFeedbackItem) {
    const draft = buildExclusionDraftFromFeedback(row);
    policyFeedbackExclusionDraft.value = draft;
    policyFeedbackExclusionDraftCandidateKey.value = draft.removeValue
      ? buildExclusionCandidateKey(draft.removeType, draft.removeValue)
      : '';
    policyFeedbackExclusionDraftModalVisible.value = true;
  }

  function handlePolicyFeedbackExclusionCandidateChange(value: string) {
    const draft = policyFeedbackExclusionDraft.value;
    if (!draft) {
      return;
    }
    const selected = parseExclusionCandidateKey(value);
    if (!selected) {
      return;
    }
    draft.removeType = selected.removeType;
    draft.removeValue = selected.removeValue;
  }

  function handlePolicyFeedbackExclusionDraftScopeChange(scopeType: WafPolicyScopeType) {
    const draft = policyFeedbackExclusionDraft.value;
    if (!draft) {
      return;
    }
    draft.scopeType = scopeType;
    if (scopeType === 'global') {
      draft.host = '';
      draft.path = '';
      draft.method = '';
    } else if (scopeType === 'site') {
      draft.path = '';
      draft.method = '';
    }
  }

  async function handleConfirmPolicyFeedbackExclusionDraft() {
    const draft = policyFeedbackExclusionDraft.value;
    if (!draft) {
      message.warning('例外草稿为空');
      return;
    }
    if (!Number(draft.policyId || 0)) {
      message.warning('请选择关联策略');
      return;
    }
    if (draft.scopeType === 'site' && !String(draft.host || '').trim()) {
      message.warning('站点作用域必须填写 Host');
      return;
    }
    if (draft.scopeType === 'route' && !String(draft.path || '').trim()) {
      message.warning('路由作用域必须填写 Path');
      return;
    }
    if (!String(draft.name || '').trim()) {
      message.warning('请填写规则名称');
      return;
    }

    resetExclusionForm();
    openExclusionEditor(
      {
        policyId: Number(draft.policyId),
        name: String(draft.name || '').trim(),
        description: draft.description,
        enabled: true,
        scopeType: draft.scopeType,
        host: String(draft.host || '').trim(),
        path: String(draft.path || '').trim(),
        method: String(draft.method || '')
          .trim()
          .toUpperCase(),
        removeType: draft.removeType,
        removeValue: String(draft.removeValue || '').trim()
      },
      !draft.removeValue
    );

    policyFeedbackExclusionDraftModalVisible.value = false;
    policyFeedbackExclusionDraft.value = null;
    policyFeedbackExclusionDraftCandidateKey.value = '';
    await navigateToPolicyExclusion();

    if (!draft.removeValue) {
      message.warning('已生成例外草稿，请补充移除值（removeById / removeByTag）后保存');
    } else {
      message.success('已根据误报反馈生成例外草稿');
    }
  }

  return {
    policyFeedbackExclusionDraftModalVisible,
    policyFeedbackExclusionDraft,
    policyFeedbackExclusionDraftCandidateKey,
    policyFeedbackExclusionCandidateOptions,
    policyFeedbackExclusionDraftDiffItems,
    handleCreateExclusionDraftFromFeedback,
    handlePolicyFeedbackExclusionCandidateChange,
    handlePolicyFeedbackExclusionDraftScopeChange,
    handleConfirmPolicyFeedbackExclusionDraft
  };
}
