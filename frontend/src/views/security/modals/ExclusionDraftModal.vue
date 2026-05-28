<script setup lang="ts">
import { NAlert, NButton, NForm, NFormItem, NFormItemGi, NGrid, NInput, NModal, NSelect } from 'naive-ui';
import type { WafPolicyScopeType } from '@/service/api/caddy-policy';
import { methodOptions, removeTypeOptions, scopeTypeOptions } from '../security-options';

interface ExclusionDraft {
  feedbackId: number;
  policyId: number;
  name: string;
  description: string;
  scopeType: string;
  host: string;
  path: string;
  method: string | null;
  removeType: string;
  removeValue: string;
}

interface DiffItem {
  field: string;
  before: string;
  after: string;
}

defineProps<{
  show: boolean;
  draft: ExclusionDraft | null;
  crsPolicyOptions: Array<{ label: string; value: number }>;
  draftCandidateKey: string;
  candidateOptions: Array<{ label: string; value: string }>;
  diffItems: DiffItem[];
  handleConfirmPolicyFeedbackExclusionDraft: () => void;
  handlePolicyFeedbackExclusionCandidateChange: (value: string) => void;
  handlePolicyFeedbackExclusionDraftScopeChange: (scopeType: WafPolicyScopeType) => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal
    :show="show"
    preset="card"
    title="确认生成例外草稿"
    class="w-760px"
    @update:show="emit('update:show', $event)"
  >
    <div v-if="draft" class="space-y-3">
      <div class="text-sm text-gray-600">来源反馈 #{{ draft.feedbackId }}</div>
      <NForm :model="draft" label-placement="left" label-width="120">
        <NGrid cols="2" x-gap="12">
          <NFormItemGi label="关联策略">
            <NSelect v-model:value="draft.policyId" :options="crsPolicyOptions" />
          </NFormItemGi>
          <NFormItemGi label="作用域">
            <NSelect
              v-model:value="draft.scopeType"
              :options="scopeTypeOptions"
              @update:value="handlePolicyFeedbackExclusionDraftScopeChange"
            />
          </NFormItemGi>
          <NFormItemGi v-if="draft.scopeType !== 'global'" label="Host">
            <NInput v-model:value="draft.host" placeholder="例如：app.example.com" />
          </NFormItemGi>
          <NFormItemGi v-if="draft.scopeType === 'route'" label="Path">
            <NInput v-model:value="draft.path" placeholder="例如：/api/login" />
          </NFormItemGi>
          <NFormItemGi v-if="draft.scopeType === 'route'" label="Method">
            <NSelect v-model:value="draft.method" :options="methodOptions" clearable placeholder="可选" />
          </NFormItemGi>
          <NFormItemGi label="移除类型">
            <NSelect v-model:value="draft.removeType" :options="removeTypeOptions" />
          </NFormItemGi>
        </NGrid>
        <NFormItem label="规则名称">
          <NInput v-model:value="draft.name" />
        </NFormItem>
      </NForm>
      <NAlert type="info" :show-icon="false">
        <template #header>草稿差异对比</template>
        <div v-if="diffItems.length === 0" class="text-xs text-gray-500">当前草稿与原反馈关键字段一致</div>
        <ul v-else class="text-xs text-gray-600 leading-6">
          <li v-for="item in diffItems" :key="item.field">
            {{ item.field }}：{{ item.before || '空' }} ->
            {{ item.after || '空' }}
          </li>
        </ul>
      </NAlert>
      <div v-if="candidateOptions.length > 1">
        <div class="mb-1 text-xs text-gray-500">候选移除值（建议文本匹配到多个候选）</div>
        <NSelect
          :value="draftCandidateKey"
          :options="candidateOptions"
          placeholder="请选择 remove 值候选"
          @update:value="handlePolicyFeedbackExclusionCandidateChange"
        />
      </div>
      <div>
        <div class="text-xs text-gray-500">移除值</div>
        <NInput
          v-model:value="draft.removeValue"
          :placeholder="draft.removeType === 'id' ? '例如：920350' : '例如：attack-sqli'"
        />
      </div>
      <div>
        <div class="text-xs text-gray-500">描述草稿</div>
        <NInput v-model:value="draft.description" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" />
      </div>
      <NAlert v-if="!draft.removeValue" type="warning" :show-icon="true">
        建议文本未解析到可用的 remove 值，请在下一步表单中补充后再保存。
      </NAlert>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="primary" @click="handleConfirmPolicyFeedbackExclusionDraft">确认生成</NButton>
      </div>
    </template>
  </NModal>
</template>
