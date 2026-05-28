<script setup lang="ts">
import type { FormInst, FormRules } from 'naive-ui';

defineProps<{
  show: boolean;
  form: {
    target: string;
    version: string;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  handleSubmitRollback: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" title="回滚版本" class="w-520px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="110">
      <NFormItem label="回滚目标" path="target">
        <NRadioGroup v-model:value="form.target">
          <NSpace>
            <NRadio value="last_good">last_good</NRadio>
            <NRadio value="version">指定版本</NRadio>
          </NSpace>
        </NRadioGroup>
      </NFormItem>
      <NFormItem v-if="form.target === 'version'" label="版本号" path="version">
        <NInput v-model:value="form.version" placeholder="例如：v4.23.0" />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="warning" :loading="submitting" @click="handleSubmitRollback">确认回滚</NButton>
      </div>
    </template>
  </NModal>
</template>
