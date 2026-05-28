<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import type { FormInst, FormRules, InputInst } from 'naive-ui';
import { methodOptions, removeTypeOptions, scopeTypeOptions } from '../security-options';

const props = defineProps<{
  show: boolean;
  form: {
    policyId: number;
    name: string;
    description: string;
    enabled: boolean;
    scopeType: string;
    host: string;
    path: string;
    method: string | null;
    removeType: string;
    removeValue: string;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  title: string;
  crsPolicyOptions: Array<{ label: string; value: number }>;
  shouldFocusRemoveValue: boolean;
  handleSubmitExclusion: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  focusedRemoveValue: [];
}>();

const removeValueInputRef = ref<InputInst | null>(null);

watch(
  () => props.show,
  value => {
    if (value && props.shouldFocusRemoveValue) {
      nextTick(() => {
        removeValueInputRef.value?.focus();
        emit('focusedRemoveValue');
      });
    }
  }
);
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
  <NModal :show="show" preset="card" :title="title" class="w-760px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="140">
      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="规则名称" path="name">
          <NInput v-model:value="form.name" placeholder="例如：ignore-login-fp" />
        </NFormItemGi>
        <NFormItemGi label="关联策略" path="policyId">
          <NSelect v-model:value="form.policyId" :options="crsPolicyOptions" />
        </NFormItemGi>
        <NFormItemGi label="作用域" path="scopeType">
          <NSelect v-model:value="form.scopeType" :options="scopeTypeOptions" />
        </NFormItemGi>
        <NFormItemGi label="移除类型" path="removeType">
          <NSelect v-model:value="form.removeType" :options="removeTypeOptions" />
        </NFormItemGi>
        <NFormItemGi v-if="form.scopeType !== 'global'" label="Host" path="host">
          <NInput v-model:value="form.host" placeholder="例如：app.example.com" />
        </NFormItemGi>
        <NFormItemGi v-if="form.scopeType === 'route'" label="Path" path="path">
          <NInput v-model:value="form.path" placeholder="例如：/api/login" />
        </NFormItemGi>
        <NFormItemGi v-if="form.scopeType === 'route'" label="Method" path="method">
          <NSelect v-model:value="form.method" :options="methodOptions" clearable placeholder="可选" />
        </NFormItemGi>
        <NFormItemGi label="是否启用">
          <NSwitch v-model:value="form.enabled" />
        </NFormItemGi>
      </NGrid>

      <NFormItem label="移除值" path="removeValue">
        <NInput
          ref="removeValueInputRef"
          v-model:value="form.removeValue"
          :placeholder="form.removeType === 'id' ? '例如：920350' : '例如：attack-sqli'"
        />
      </NFormItem>
      <NFormItem label="描述" path="description">
        <NInput v-model:value="form.description" placeholder="可选，记录误报场景与原因" />
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmitExclusion">保存</NButton>
      </div>
    </template>
  </NModal>
</template>
