<script setup lang="ts">
import type { FormInst, FormRules } from 'naive-ui';
import { methodOptions, scopeTypeOptions } from '../security-options';

defineProps<{
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
    priority: number;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  title: string;
  crsPolicyOptions: Array<{ label: string; value: number }>;
  handleSubmitBinding: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" :title="title" class="w-760px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="140">
      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="绑定名称" path="name">
          <NInput v-model:value="form.name" placeholder="例如：site-main-binding" />
        </NFormItemGi>
        <NFormItemGi label="关联策略" path="policyId">
          <NSelect v-model:value="form.policyId" :options="crsPolicyOptions" />
        </NFormItemGi>
        <NFormItemGi label="作用域" path="scopeType">
          <NSelect v-model:value="form.scopeType" :options="scopeTypeOptions" />
        </NFormItemGi>
        <NFormItemGi label="优先级" path="priority">
          <NInputNumber v-model:value="form.priority" :show-button="false" :min="1" :max="1000" class="w-full" />
        </NFormItemGi>
        <NFormItemGi v-if="form.scopeType !== 'global'" label="Host" path="host">
          <NInput v-model:value="form.host" placeholder="例如：app.example.com" />
        </NFormItemGi>
        <NFormItemGi v-if="form.scopeType === 'route'" label="Path" path="path">
          <NInput v-model:value="form.path" placeholder="例如：/api" />
        </NFormItemGi>
        <NFormItemGi v-if="form.scopeType === 'route'" label="Method" path="method">
          <NSelect v-model:value="form.method" :options="methodOptions" clearable placeholder="可选" />
        </NFormItemGi>
        <NFormItemGi label="是否启用">
          <NSwitch v-model:value="form.enabled" />
        </NFormItemGi>
      </NGrid>

      <NFormItem label="描述" path="description">
        <NInput v-model:value="form.description" placeholder="可选，记录生效范围和意图" />
      </NFormItem>
    </NForm>
    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmitBinding">保存</NButton>
      </div>
    </template>
  </NModal>
</template>
