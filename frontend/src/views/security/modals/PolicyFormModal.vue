<script setup lang="ts">
import type { FormInst, FormRules } from 'naive-ui';
import { policyAuditEngineOptions, policyAuditLogFormatOptions, policyEngineModeOptions } from '../security-options';

defineProps<{
  show: boolean;
  form: {
    name: string;
    description: string;
    enabled: boolean;
    isDefault: boolean;
    engineMode: string;
    auditEngine: string;
    auditLogFormat: string;
    auditRelevantStatus: string;
    requestBodyAccess: boolean;
    requestBodyLimit: number;
    requestBodyNoFilesLimit: number;
    config: string;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  title: string;
  handleSubmitPolicy: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" :title="title" class="w-760px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="150">
      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="策略名称" path="name">
          <NInput v-model:value="form.name" placeholder="例如：default-runtime-policy" />
        </NFormItemGi>
        <NFormItemGi label="是否默认策略">
          <NSwitch v-model:value="form.isDefault" />
        </NFormItemGi>
        <NFormItemGi label="引擎模式" path="engineMode">
          <NSelect v-model:value="form.engineMode" :options="policyEngineModeOptions" />
        </NFormItemGi>
        <NFormItemGi label="审计模式" path="auditEngine">
          <NSelect v-model:value="form.auditEngine" :options="policyAuditEngineOptions" />
        </NFormItemGi>
        <NFormItemGi label="审计日志格式" path="auditLogFormat">
          <NSelect v-model:value="form.auditLogFormat" :options="policyAuditLogFormatOptions" />
        </NFormItemGi>
        <NFormItemGi label="请求体访问">
          <NSwitch v-model:value="form.requestBodyAccess" />
        </NFormItemGi>
        <NFormItemGi label="启用策略">
          <NSwitch v-model:value="form.enabled" />
        </NFormItemGi>
      </NGrid>

      <NFormItem label="描述" path="description">
        <NInput v-model:value="form.description" placeholder="可选，记录策略用途与变更说明" />
      </NFormItem>

      <NFormItem label="审计状态匹配" path="auditRelevantStatus">
        <NInput v-model:value="form.auditRelevantStatus" placeholder="例如：^(?:5|4(?!04))" />
      </NFormItem>

      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="请求体限制（字节）" path="requestBodyLimit">
          <NInputNumber
            v-model:value="form.requestBodyLimit"
            :show-button="false"
            :min="1"
            :max="1024 * 1024 * 1024"
            class="w-full"
          />
        </NFormItemGi>
        <NFormItemGi label="无文件请求体限制（字节）" path="requestBodyNoFilesLimit">
          <NInputNumber
            v-model:value="form.requestBodyNoFilesLimit"
            :show-button="false"
            :min="1"
            :max="1024 * 1024 * 1024"
            class="w-full"
          />
        </NFormItemGi>
      </NGrid>

      <NFormItem label="扩展配置(JSON)" path="config">
        <NInput
          v-model:value="form.config"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 6 }"
          placeholder='可选，例如：{"custom_tag":"runtime"}'
        />
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmitPolicy">保存</NButton>
      </div>
    </template>
  </NModal>
</template>
