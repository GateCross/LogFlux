<script setup lang="ts">
import type { FormInst, FormRules } from 'naive-ui';
import { policyFeedbackStatusOptions } from '../security-options';

defineProps<{
  show: boolean;
  form: {
    id: number;
    feedbackStatus: string;
    processNote: string;
    assignee: string;
    dueAt: string;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  handleSubmitPolicyFeedbackProcess: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" title="处理误报反馈" class="w-640px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="120">
      <NFormItem label="处理状态" path="feedbackStatus">
        <NSelect v-model:value="form.feedbackStatus" :options="policyFeedbackStatusOptions" />
      </NFormItem>
      <NFormItem label="责任人" path="assignee">
        <NInput v-model:value="form.assignee" placeholder="可选，例如 alice" />
      </NFormItem>
      <NFormItem label="截止时间" path="dueAt">
        <NInput v-model:value="form.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
      </NFormItem>
      <NFormItem label="处理备注" path="processNote">
        <NInput
          v-model:value="form.processNote"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          placeholder="可选，记录确认依据或处理结果"
        />
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="warning" :loading="submitting" @click="handleSubmitPolicyFeedbackProcess">保存状态</NButton>
      </div>
    </template>
  </NModal>
</template>
