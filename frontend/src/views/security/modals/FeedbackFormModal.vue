<script setup lang="ts">
import type { FormInst, FormRules } from 'naive-ui';
import { methodOptions } from '../security-options';

defineProps<{
  show: boolean;
  form: {
    policyId: number | null;
    host: string;
    path: string;
    method: string | null;
    status: number;
    assignee: string;
    dueAt: string;
    sampleUri: string;
    reason: string;
    suggestion: string;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  crsPolicyOptions: Array<{ label: string; value: number }>;
  handleSubmitPolicyFeedback: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" title="标记误报反馈" class="w-760px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="130">
      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="关联策略" path="policyId">
          <NSelect
            v-model:value="form.policyId"
            :options="crsPolicyOptions"
            clearable
            placeholder="可选，不填表示全部策略"
          />
        </NFormItemGi>
        <NFormItemGi label="状态码" path="status">
          <NInputNumber v-model:value="form.status" :show-button="false" :min="100" :max="599" class="w-full" />
        </NFormItemGi>
        <NFormItemGi label="责任人" path="assignee">
          <NInput v-model:value="form.assignee" placeholder="可选，例如 alice" />
        </NFormItemGi>
        <NFormItemGi label="截止时间" path="dueAt">
          <NInput v-model:value="form.dueAt" placeholder="可选，YYYY-MM-DD HH:mm:ss" />
        </NFormItemGi>
        <NFormItemGi label="Host" path="host">
          <NInput v-model:value="form.host" placeholder="可选，例如 app.example.com" />
        </NFormItemGi>
        <NFormItemGi label="Path" path="path">
          <NInput v-model:value="form.path" placeholder="可选，例如 /api/login" />
        </NFormItemGi>
        <NFormItemGi label="Method" path="method">
          <NSelect v-model:value="form.method" :options="methodOptions" clearable placeholder="可选" />
        </NFormItemGi>
        <NFormItemGi label="示例 URI" path="sampleUri">
          <NInput v-model:value="form.sampleUri" placeholder="可选，记录原始 URI 便于复盘" />
        </NFormItemGi>
      </NGrid>
      <NFormItem label="误报原因" path="reason">
        <NInput
          v-model:value="form.reason"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          placeholder="必填：为何判断为误报"
        />
      </NFormItem>
      <NFormItem label="建议动作" path="suggestion">
        <NInput
          v-model:value="form.suggestion"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 4 }"
          placeholder="可选：例如建议添加 removeById、放宽阈值或补白名单"
        />
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="warning" :loading="submitting" @click="handleSubmitPolicyFeedback">提交反馈</NButton>
      </div>
    </template>
  </NModal>
</template>
