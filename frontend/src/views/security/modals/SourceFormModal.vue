<script setup lang="ts">
import type { FormInst, FormRules } from 'naive-ui';
import { authTypeOptions, modeOptions } from '../security-options';

defineProps<{
  show: boolean;
  form: {
    name: string;
    kind: string;
    mode: string;
    url: string;
    checksumUrl: string;
    proxyUrl: string;
    authType: string;
    authSecret: string;
    schedule: string;
    enabled: boolean;
    autoCheck: boolean;
    autoDownload: boolean;
    autoActivate: boolean;
    meta: string;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  title: string;
  handleSubmitSource: () => void;
  applyDefaultSource: () => void;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" :title="title" class="w-720px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="120">
      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="名称" path="name">
          <NInput v-model:value="form.name" placeholder="例如：official-crs" />
        </NFormItemGi>
        <NFormItemGi label="类型" path="kind">
          <NInput value="crs" disabled />
        </NFormItemGi>
        <NFormItemGi label="模式" path="mode">
          <NSelect v-model:value="form.mode" :options="modeOptions" />
        </NFormItemGi>
        <NFormItemGi label="鉴权类型" path="authType">
          <NSelect v-model:value="form.authType" :options="authTypeOptions" />
        </NFormItemGi>
      </NGrid>

      <NFormItem label="默认源">
        <div class="flex flex-wrap gap-2">
          <NButton size="small" secondary @click="applyDefaultSource">应用 CRS 默认源</NButton>
        </div>
      </NFormItem>

      <NFormItem v-if="form.mode === 'remote'" label="源地址" path="url">
        <NInput
          v-model:value="form.url"
          placeholder="https://api.github.com/repos/coreruleset/coreruleset/releases/latest"
        />
      </NFormItem>

      <NFormItem v-if="form.mode === 'remote'" label="校验地址" path="checksumUrl">
        <NInput v-model:value="form.checksumUrl" placeholder="可选，SHA256 清单地址" />
      </NFormItem>

      <NFormItem v-if="form.mode === 'remote'" label="代理地址" path="proxyUrl">
        <NInput v-model:value="form.proxyUrl" placeholder="可选，例如：http://127.0.0.1:7890" />
      </NFormItem>

      <NFormItem v-if="form.authType !== 'none'" label="鉴权密钥" path="authSecret">
        <NInput
          v-model:value="form.authSecret"
          type="password"
          show-password-on="mousedown"
          placeholder="Token 或 user:password"
        />
      </NFormItem>

      <NFormItem label="调度表达式" path="schedule">
        <NInput v-model:value="form.schedule" placeholder="例如：0 0 */6 * * *" />
      </NFormItem>

      <NFormItem label="附加元数据" path="meta">
        <NInput
          v-model:value="form.meta"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 5 }"
          placeholder="JSON 字符串，可选"
        />
      </NFormItem>

      <NGrid cols="2" x-gap="12">
        <NFormItemGi label="启用">
          <NSwitch v-model:value="form.enabled" />
        </NFormItemGi>
        <NFormItemGi label="自动检查">
          <NSwitch v-model:value="form.autoCheck" />
        </NFormItemGi>
        <NFormItemGi label="自动下载">
          <NSwitch v-model:value="form.autoDownload" />
        </NFormItemGi>
        <NFormItemGi label="自动激活">
          <NSwitch v-model:value="form.autoActivate" />
        </NFormItemGi>
      </NGrid>
    </NForm>

    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmitSource">保存</NButton>
      </div>
    </template>
  </NModal>
</template>
