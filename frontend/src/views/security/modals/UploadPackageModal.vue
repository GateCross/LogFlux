<script setup lang="ts">
import type { FormInst, FormRules, UploadFileInfo } from 'naive-ui';

defineProps<{
  show: boolean;
  form: {
    kind: string;
    version: string;
    checksum: string;
    activateNow: boolean;
    file: File | null;
  };
  formRef: FormInst | null;
  rules: FormRules;
  submitting: boolean;
  handleSubmitUpload: () => void;
  handleBeforeUpload: (data: { file: UploadFileInfo }) => boolean;
  handleRemoveUpload: () => boolean;
}>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();
</script>

<!-- eslint-disable vue/no-mutating-props, vue/no-unused-refs, vue/no-unused-properties -->
<template>
  <NModal :show="show" preset="card" title="上传规则包" class="w-640px" @update:show="emit('update:show', $event)">
    <NForm ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="110">
      <NFormItem label="类型" path="kind">
        <NInput value="crs" disabled />
      </NFormItem>
      <NFormItem label="版本号" path="version">
        <NInput v-model:value="form.version" placeholder="例如：v4.23.0-custom.1" />
      </NFormItem>
      <NFormItem label="SHA256" path="checksum">
        <NInput v-model:value="form.checksum" placeholder="可选，建议填写" />
      </NFormItem>
      <NFormItem label="立即激活" path="activateNow">
        <NSwitch v-model:value="form.activateNow" />
      </NFormItem>
      <NFormItem label="规则包" path="file">
        <NUpload
          :default-upload="false"
          :max="1"
          :show-file-list="true"
          accept=".zip,.tar.gz"
          @before-upload="handleBeforeUpload"
          @remove="handleRemoveUpload"
        >
          <NButton>选择文件</NButton>
        </NUpload>
      </NFormItem>
    </NForm>

    <template #footer>
      <div class="flex justify-end gap-2">
        <NButton @click="emit('update:show', false)">取消</NButton>
        <NButton type="primary" :loading="submitting" @click="handleSubmitUpload">上传并入库</NButton>
      </div>
    </template>
  </NModal>
</template>
